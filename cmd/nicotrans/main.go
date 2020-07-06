package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"gitea.chriswiegman.com/chriswiegman/goodhosts"
	"github.com/hype5/nicotrans-go/pkg/certificate"
	"github.com/hype5/nicotrans-go/pkg/nico"
	"github.com/hype5/nicotrans-go/pkg/system"
	"github.com/hype5/nicotrans-go/pkg/translator"
	"github.com/op/go-logging"
)

var serverIP = flag.String("ip", "127.0.0.1", "서버 주소")
var serverPort = flag.Int("port", 443, "서버 포트")
var serverCertPath = flag.String("cert", "server.crt", "서버 SSL 인증서 경로")
var serverPrivPath = flag.String("cert-privatekey", "server.key", "서버 SSL 인증서 키 경로")
var serverCreate = flag.Bool("cert-create", true, "서버 SSL 인증서가 존재하지 않을 때 생성할지?")
var editHosts = flag.Bool("edit-hosts", true, "호스트 파일에 자동으로 아이피를 추가할지?")
var langPlatform = flag.String("lang-platform", "papago", "사용될 번역기 종류")
var langSource = flag.String("lang-source", "ja", "번역할 언어 2자리 코드")
var langTarget = flag.String("lang-target", "ko", "번역될 언어 2자리 코드")

var log = logging.MustGetLogger("nicotrans")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`,
)

func handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api.json/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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
	switch *langPlatform {
	case "papago":
		e = <-translator.WithPapagoAsChunks(&chunks, *langSource, *langTarget)
	default:
		log.Warningf("%s 값은 번역 플랫폼이 아닙니다", *langPlatform)
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

	// 로거 만들기
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.ERROR, "nicotrans")
	logging.SetBackend(backendLeveled, backendFormatter)

	addr := fmt.Sprintf("%s:%d", *serverIP, *serverPort)

	// 인증서 불러오기
	var cert []byte
	var priv interface{}
	var e error

	cert, priv, e = certificate.ImportPemBlock(*serverCertPath, *serverPrivPath)

	if e == nil {
		// 깔끔한 코드를 위한 빈 공간
	} else if *serverCreate {
		log.Error("인증서를 불러올 수 없습니다:", e)
		log.Info("새 인증서를 생성합니다")

		cert, priv, e = certificate.Create([]string{"nmsg.nicovideo.jp"})
		if e != nil {
			log.Panic("인증서 생성에 실패했습니다:", e)
		}

		if e := certificate.Export(cert, priv, *serverCertPath, *serverPrivPath); e != nil {
			log.Error("인증서를 저장할 수 없습니다:", e)
		}
	} else {
		log.Panic("인증서를 불러올 수 없습니다:", e)
	}

	// 윈도우 환경에선 호스트 자동으로 수정해주기
	if runtime.GOOS == "windows" && *editHosts {
		target := *serverIP
		if target == "0.0.0.0" {
			target = "127.0.0.1"
		}

		var root bool
		var hosts goodhosts.Hosts
		var e error

		hosts, e = goodhosts.NewHosts()
		if e != nil {
			log.Error("호스트 파일을 열 수 없습니다", e)
		} else {
			if !hosts.Has(target, "nmsg.nicovideo.jp") {
				log.Warning("호스트 파일에 포워딩에 필요한 엔트리가 존재하지 않습니다")

				root, e = system.HasRoot()

				if e != nil {
					// 관리자 권한을 불러올 수 없다면 오류 메세지 출력하기
					log.Error("사용자 정보를 확인하는 중 오류가 발생했습니다:", e)
				} else if root {
					// 관리자 권한으로 실행했다면 호스트 수정하기
					hosts.Add(target, "nmsg.nicovideo.jp")

					if e = hosts.Flush(); e != nil {
						log.Error("호스트 파일을 저장할 수 없습니다:", e)
					}
				} else {
					// 관리자 권한으로 다시 열기
					log.Info("호스트 파일 수정을 위해 관리자 권한으로 다시 실행합니다")

					if e = system.RunMeElevated(); e != nil {
						log.Error("관리자 권한으로 다시 실행할 수 없습니다:", e)
					}
				}
			}
		}

		// 호스트 파일 수정에 실패했다면 수동 수정 안내 메세지 출력하기
		if e != nil {
			log.Info("호스트 파일을 수동으로 편집하기 위해선 다음 과정을 따라주세요")
			log.Info("\t1) 메모장 같은 편집기를 관리자 권한으로 엽니다")
			log.Info("\t2) %WINDIR%/System32/drivers/etc/hosts 파일을 엽니다")
			log.Info("\t3) 가장 아래에 다음 줄을 추가합니다")
			log.Infof("\t\t%s nmsg.nicovideo.jp", target)
		}
	}

	// 서버 만들기
	server := http.Server{
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{
				{
					Certificate: [][]byte{cert},
					PrivateKey:  priv,
				},
			},
		},
		Handler: http.HandlerFunc(handle),
	}

	log.Infof("니코트랜스를 실행합니다: %s", addr)

	if e := server.ListenAndServeTLS("", ""); e != nil {
		log.Panic("서버를 여는 중 오류가 발생했습니다", e)
	}
}
