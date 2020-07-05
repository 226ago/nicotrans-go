package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"internal/translator"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/op/go-logging"
)

// Chat 댓글 구조
type Chat struct {
	index   int
	content string
}

// FetchMessageResult 메세지 응답 구조
type FetchMessageResult struct {
	payload []interface{}
	chats   []Chat
}

var serverIP = flag.String("ip", "0.0.0.0", "서버 주소")
var serverPort = flag.Int("port", 443, "서버 포트")
var serverCert = flag.String("sslcert", "server.crt", "서버 SSL 인증서 경로")
var serverCertKey = flag.String("sslkey", "server.key", "서버 SSL 인증서 키 경로")
var translatorType = flag.String("translator", "papago", "사용될 번역기 종류")
var langSource = flag.String("langsrc", "ja", "번역할 언어 2자리 코드")
var langTarget = flag.String("langtarget", "ko", "번역될 언어 2자리 코드")

var log = logging.MustGetLogger("nicotrans")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

var dial = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
	DualStack: true,
}

var pattern = regexp.MustCompile(`(?m)^§\n([^§]+)`)

func fetchMessage(data io.Reader) <-chan FetchMessageResult {
	resolve := make(chan FetchMessageResult)

	go func() {
		res, e := http.Post("https://nmsg.nicovideo.jp/api.json/", "text/plain", data)
		if e != nil {
			panic(e)
		}

		defer res.Body.Close()
		body, e := ioutil.ReadAll(res.Body)
		if e != nil {
			panic(e)
		}

		var payload []interface{}
		var chats []Chat

		if e := json.Unmarshal(body, &payload); e != nil {
			panic(e)
		}

		for i, v := range payload {
			chat, ok := v.(map[string]interface{})["chat"]
			if !ok {
				continue
			}

			content, ok := chat.(map[string]interface{})["content"]
			if !ok || content == "" {
				continue
			}

			chats = append(chats, Chat{
				index:   i,
				content: content.(string),
			})
		}

		resolve <- FetchMessageResult{
			payload: payload,
			chats:   chats,
		}
	}()

	return resolve
}

func chunkize(chats *[]Chat) []bytes.Buffer {
	var chunks = []bytes.Buffer{{}}

	for _, chat := range *chats {
		idx := len(chunks) - 1
		item := fmt.Sprintf("§\n%s\n", chat.content)
		nextChunkLength := chunks[idx].Len() + len(item)

		// 번역기가 받을 수 있는 최대 길이를 넘으면 다음 청크로 이동하기
		if nextChunkLength > 5000 {
			idx++
		}

		// 청크가 존재하지 않는다면 빈 버퍼 추가하기
		if len(chunks) <= idx {
			chunks = append(chunks, bytes.Buffer{})
		}

		chunks[idx].WriteString(item)
	}

	return chunks
}

func handleDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// TODO: DNS 뜯어오기
	if addr == "nmsg.nicovideo.jp:443" {
		addr = "133.152.39.27:443"
	}
	return dial.DialContext(ctx, network, addr)
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	log.Infof("%s - %s", r.RemoteAddr, r.Referer())

	message := <-fetchMessage(r.Body)
	chunks := chunkize(&message.chats)

	log.Infof("%s - %s - 코멘트 %d개", r.RemoteAddr, r.Referer(), len(message.chats))

	// 번역하기
	var sequences []translator.TranslatedSequence

	switch *translatorType {
	case "papago":
		sequences = <-translator.WithPapagoAsChunks(chunks, *langSource, *langTarget)
	default:
		log.Warningf("%s 값은 번역기로 사용할 수 없습니다", *translatorType)
	}

	if len(sequences) > 0 {
		// 번역된 시퀀스를 바이트로 버퍼로
		var translated bytes.Buffer

		for _, seq := range sequences {
			translated.WriteString(seq.Translated)
			translated.WriteString("\n")
		}

		// 기존 코멘트를 번역된 내용으로 대체하기
		for line, groups := range pattern.FindAllSubmatch(translated.Bytes(), -1) {
			index := message.chats[line].index
			content := string(groups[1])
			// ㅋㅋㅋㅋㅋㅋㅋ
			message.payload[index].(map[string]interface{})["chat"].(map[string]interface{})["content"] = content
		}
	}

	payload, e := json.Marshal(message.payload)
	if e != nil {
		panic(e)
	}

	w.Write(payload)
}

func main() {
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *serverIP, *serverPort)
	addrTarget := *serverIP
	if addrTarget == "0.0.0.0" {
		addrTarget = "127.0.0.1"
	}

	// 로거 만들기
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.ERROR, "nicotrans")
	logging.SetBackend(backendLeveled, backendFormatter)

	log.Infof("니코트랜스를 실행합니다: %s", addr)
	log.Infof("코멘트를 뜯어오기 위해선 호스트 파일에 다음 줄을 추가해야합니다\n")
	log.Infof("\t%s nmsg.nicovideo.jp\n\n", addrTarget)

	// 웹 서버 시작하기
	http.DefaultTransport.(*http.Transport).DialContext = handleDialContext
	http.HandleFunc("/api.json/", handle)

	if e := http.ListenAndServeTLS(addr, *serverCert, *serverCertKey, nil); e != nil {
		panic(e)
	}
}
