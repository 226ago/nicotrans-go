package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// PapagoRequestPayload 파파고 요청 페이로드
type PapagoRequestPayload struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Text   string `json:"text"`
}

// PapagoResponsePayload 파파고 응답 페이로드
type PapagoResponsePayload struct {
	TranslatedText string `json:"translatedText"`
}

// Reply 댓글
type Reply struct {
	index   int
	content string
}

func fetchTranslate(text string) string {
	data, e := json.Marshal(PapagoRequestPayload{
		Source: "ja",
		Target: "ko",
		Text:   text,
	})
	if e != nil {
		panic(e)
	}

	payload := fmt.Sprintf("data=%s", url.QueryEscape(string(data)))

	res, e := http.Post(
		"https://papago.naver.com/apis/n2mt/translate",
		"x-www-form-urlencoded",
		strings.NewReader(payload))
	if e != nil {
		panic(e)
	}

	defer res.Body.Close()
	body, e := ioutil.ReadAll(res.Body)
	if e != nil {
		panic(e)
	}

	var response PapagoResponsePayload
	if e := json.Unmarshal(body, &response); e != nil {
		panic(e)
	}

	return response.TranslatedText
}

func fetchComments(data io.Reader) ([]interface{}, []Reply) {
	res, e := http.Post("https://nmsg.nicovideo.jp/api.json/", "text/plain", data)
	if e != nil {
		panic(e)
	}

	defer res.Body.Close()
	body, e := ioutil.ReadAll(res.Body)
	if e != nil {
		panic(e)
	}

	var items []interface{}
	var replies []Reply

	if e := json.Unmarshal(body, &items); e != nil {
		panic(e)
	}

	for i, v := range items {
		item := v.(map[string]interface{})

		chat, ok := item["chat"]
		if !ok {
			continue
		}

		content, ok := chat.(map[string]interface{})["content"]
		if !ok || content == "" {
			continue
		}

		replies = append(replies, Reply{
			index:   i,
			content: content.(string),
		})
	}

	return items, replies
}

func repliesToChunks(replies *[]Reply, chunks *[]bytes.Buffer) {
	for _, reply := range *replies {
		idx := len(*chunks) - 1
		message := fmt.Sprintf("$\n%s\n", reply.content)
		nextChunkLength := (*chunks)[idx].Len() + len(message)

		if nextChunkLength > 5000 {
			idx++
		}

		// 현재 청크가 존재하지 않는다면 빈 문자열 추가하기
		if len(*chunks) <= idx {
			*chunks = append(*chunks, bytes.Buffer{})
		}

		(*chunks)[idx].WriteString(message)
	}
}

func main() {
	pattern := regexp.MustCompile(`(?m)^\$\n(.+)`)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "니코트랜스가 작동하고 있어요!")
	})

	http.HandleFunc("/api.json/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		// 댓글 불러오기
		items, replies := fetchComments(r.Body)

		chunks := []bytes.Buffer{bytes.Buffer{}}
		translated := bytes.Buffer{}

		repliesToChunks(&replies, &chunks)

		for _, chunk := range chunks {
			translatedString := fetchTranslate(chunk.String())
			translated.WriteString(translatedString)
		}

		for line, groups := range pattern.FindAllSubmatch(translated.Bytes(), -1) {
			index := replies[line].index
			content := string(groups[1])
			items[index].(map[string]interface{})["chat"].(map[string]interface{})["content"] = content
		}

		payload, e := json.Marshal(items)
		if e != nil {
			panic(e)
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	})

	ip := flag.String("ip", "0.0.0.0", "웹 서버 아이피")
	port := flag.Int("port", 443, "웹 서버 포트")
	certFile := flag.String("certfile", "server.crt", "인증서 파일 경로")
	keyFile := flag.String("keyfile", "server.key", "인증서 개인 키 파일 경로")

	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *ip, *port)
	addrTarget := *ip
	if addrTarget == "0.0.0.0" {
		addrTarget = "127.0.0.1"
	}

	fmt.Printf("니코트랜스를 실행합니다 (%s)\n", addr)
	fmt.Printf("코멘트를 뜯어오기 위해선 호스트 파일에 다음 줄을 추가해야합니다\n")
	fmt.Printf("\t%s nmsg.nicovideo.jp\n\n", addrTarget)

	fmt.Printf("또한 HTTPS 를 사용하기 때문에 인증서를 추가해줘야만 정상적으로 사용할 수 있습니다\n")
	fmt.Printf("현재 사용 중인 인증서의 위치는 다음과 같습니다\n")
	fmt.Printf("\t%s\n", *certFile)

	if e := http.ListenAndServeTLS(addr, *certFile, *keyFile, nil); e != nil {
		panic(e)
	}
}
