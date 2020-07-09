package main

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"gitea.chriswiegman.com/chriswiegman/goodhosts"
	"github.com/hype5/nicotrans-go/pkg/certificate"
	"github.com/hype5/nicotrans-go/pkg/nico"
	"github.com/hype5/nicotrans-go/pkg/system"
	"github.com/hype5/nicotrans-go/pkg/translator"
	"github.com/op/go-logging"
)

var serverIP = flag.String("ip", "127.0.0.1", "서버 주소")
var serverPort = flag.Int("port", 443, "서버 포트")
var certPath = flag.String("cert", "server.crt", "서버 SSL 인증서 경로")
var certPrivPath = flag.String("cert-privatekey", "server.key", "서버 SSL 인증서 키 경로")
var certCreate = flag.Bool("cert-create", true, "서버 SSL 인증서가 존재하지 않을 때 생성할지?")
var certInstall = flag.Bool("cert-install", true, "서버 SSL 인증서를 설치할지?")

var editHosts = flag.Bool("edit-hosts", true, "호스트 파일에 자동으로 아이피를 추가할지?")

var langPlatform = flag.String("lang-platform", "papago", "사용될 번역기 종류")
var langSource = flag.String("lang-source", "ja", "번역할 언어 2자리 코드")
var langTarget = flag.String("lang-target", "ko", "번역될 언어 2자리 코드")

var log = logging.MustGetLogger("nicotrans")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s}%{color:reset} %{message}`,
)

var certificateTemplate = &x509.Certificate{
	SerialNumber: new(big.Int).SetInt64(int64(time.Now().Year())),
	Subject: pkix.Name{
		Organization: []string{"NicoTrans"},
	},
	DNSNames:    []string{"nmsg.nicovideo.jp"},
	NotBefore:   time.Now(),
	NotAfter:    time.Now().AddDate(10, 0, 0),
	KeyUsage:    x509.KeyUsageDigitalSignature,
	ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	IsCA:        true,
}

func initHosts() error {
	if *editHosts && runtime.GOOS == "windows" {
		log.Info("호스트 파일을 확인합니다")

		hosts, e := goodhosts.NewHosts()
		if e != nil {
			return fmt.Errorf("호스트 파일을 열 수 없습니다: %s", e)
		}

		if !hosts.Has(*serverIP, "nmsg.nicovideo.jp") {
			r, e := system.HasRoot()
			if e != nil {
				log.Errorf("사용자 권한 정보를 불러오는데 실패했습니다: %s", e)
			} else if r {
				// 관리자 권한으로 실행했다면 호스트 수정하기
				hosts.Add(*serverIP, "nmsg.nicovideo.jp")

				if e := hosts.Flush(); e != nil {
					return fmt.Errorf("호스트 파일을 저장할 수 없습니다: %s", e)
				}
			} else {
				log.Info("호스트 파일 수정을 위해 관리자 권한 취득을 시도합니다")

				if e := system.RunMeElevated(); e != nil {
					// 관리자 권한 취득 실패
					return fmt.Errorf("호스트 파일 수정을 위한 관리자 권한 취득에 실패했습니다: %s", e)
				}

				os.Exit(0)
			}
		}
	}

	return nil
}

func initCertificate() (*x509.Certificate, interface{}, error) {
	cert, priv, e := certificate.Import(*certPath, *certPrivPath)
	if e != nil {
		log.Errorf("인증서를 불러올 수 없습니다: %s", e)

		if *certCreate {
			log.Info("새 인증서를 생성합니다")

			// 새 인증서 만들기
			if cert, priv, e = certificate.Create(certificateTemplate); e != nil {
				return nil, nil, fmt.Errorf("인증서를 생성할 수 없습니다: %s", e)
			}

			// 새로 만든 인증서 파일로 저장하기
			if e := certificate.Export(cert, priv, *certPath, *certPrivPath); e != nil {
				return nil, nil, fmt.Errorf("인증서를 저장할 수 없습니다: %s", e)
			}
		} else {
			return nil, nil, fmt.Errorf("인증서가 없으면 서버를 실행할 수 없습니다")
		}
	}

	if *certInstall && runtime.GOOS == "windows" {
		log.Info("인증서 설치를 시도합니다")

		if e := certificate.InstallAsRootCA(cert); e == nil {
			msg := []string{
				"인증서를 성공적으로 설치했습니다",
				"\t브라우저가 열려있을 때 인증서를 설치하면 캐시로 인해 인식되지 않을 수 있습니다",
				"\t코멘트가 보이지 않는다면 열린 브라우저 창을 모두 닫고 다시 열어주세요",
			}
			log.Info(strings.Join(msg, "\n"))
		} else {
			return nil, nil, fmt.Errorf("인증서를 설치할 수 없습니다: %s", e)
		}
	}

	return cert, priv, nil
}

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

	// 기록 초기화
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.ERROR, "nicotrans")
	logging.SetBackend(backendLeveled, backendFormatter)

	// 호스트 파일 초기화
	if e := initHosts(); e != nil {
		msg := []string{
			"호스트 파일을 수동으로 편집하고 싶다면 다음 과정을 따라해주세요",
			"\t1) 메모장 같은 편집기를 관리자 권한으로 엽니다",
			"\t2) %WINDIR%/System32/drivers/etc/hosts 파일을 엽니다",
			"\t3) 가장 아래에 다음 줄을 추가하고 저장합니다",
			"\t\t" + *serverIP + " nmsg.nicovideo.jp",
		}

		log.Errorf(e.Error())
		log.Info(strings.Join(msg, "\n"))
	}

	// 인증서 초기화
	cert, priv, e := initCertificate()
	if e != nil {
		log.Panic(e)
	}

	// 서버 만들기
	addr := fmt.Sprintf("%s:%d", *serverIP, *serverPort)
	server := http.Server{
		Addr: addr,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{
				{
					Certificate: [][]byte{cert.Raw},
					PrivateKey:  priv,
				},
			},
		},
		Handler: http.HandlerFunc(handle),
	}

	log.Infof("니코트랜스를 실행합니다: %s", addr)

	if e := server.ListenAndServeTLS("", ""); e != nil {
		log.Panic("서버를 여는 중 오류가 발생했습니다\n", e)
	}
}
