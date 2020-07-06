package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/hype5/nicotrans-go/pkg/nico"
	"github.com/hype5/nicotrans-go/pkg/translator"
	"github.com/op/go-logging"
)

var serverIP = flag.String("ip", "0.0.0.0", "서버 주소")
var serverPort = flag.Int("port", 443, "서버 포트")
var serverCert = flag.String("sslcert", "server.crt", "서버 SSL 인증서 경로")
var serverCertKey = flag.String("sslkey", "server.key", "서버 SSL 인증서 키 경로")
var translatorType = flag.String("translator", "papago", "사용될 번역기 종류")
var langSource = flag.String("langsrc", "ja", "번역할 언어 2자리 코드")
var langTarget = flag.String("langtarget", "ko", "번역될 언어 2자리 코드")

var log = logging.MustGetLogger("nicotrans")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`,
)

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var e error

	defer func() {
		if e == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error(e)
		}

		r.Body.Close()
	}()

	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Infof("%s - %s", r.RemoteAddr, r.Referer())

	// 받은 데이터를 기존 API 서버로 포워딩한 뒤 데이터 불러오기
	message := <-nico.Fetch(r.Body)
	if message.Error != nil {
		e = message.Error
		return
	}

	chunks := nico.MessageToChunks(message, 5000)

	log.Infof("%s - %s - 코멘트 %d개", r.RemoteAddr, r.Referer(), len(message.Chats))

	// 번역하기
	switch *translatorType {
	case "papago":
		e = <-translator.WithPapagoAsChunks(&chunks, *langSource, *langTarget)
	default:
		log.Warningf("%s 값은 번역기로 사용할 수 없습니다", *translatorType)
	}

	// 번역 중 오류가 발생했다면 멈추기
	if e != nil {
		return
	}

	nico.ChunksToMessage(&message, chunks)

	// 변환한 메세지를 다시 페이로드로 바꾸기
	payload, e := nico.MessageToPayload(message)
	if e != nil {
		return
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
	http.HandleFunc("/api.json/", handle)

	if e := http.ListenAndServeTLS(addr, *serverCert, *serverCertKey, nil); e != nil {
		panic(e)
	}
}
