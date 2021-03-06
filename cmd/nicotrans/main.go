package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
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

var hostsEdit = flag.Bool("hosts-edit", true, "호스트 파일에 자동으로 아이피를 추가할지?")

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

var queriesPattern = regexp.MustCompile(`(?m)^§(\d+)\n([^§]+)`)

func initHosts() error {
	if *hostsEdit && runtime.GOOS == "windows" {
		log.Info("호스트 파일을 확인합니다")

		hosts, e := goodhosts.NewHosts()
		if e != nil {
			return fmt.Errorf("호스트 파일을 열 수 없습니다: %s", e)
		}

		if hosts.Has(*serverIP, "nmsg.nicovideo.jp") {
			log.Info("호스트 파일에 포워딩에 필요한 항목이 존재합니다")
		} else {
			log.Info("호스트 파일에 포워딩에 필요한 항목이 존재하지 않습니다")

			r, e := system.HasRoot()
			if e != nil {
				log.Errorf("사용자 권한 정보를 불러오는데 실패했습니다: %s", e)
			} else if r {
				// 관리자 권한으로 실행했다면 호스트 수정하기
				hosts.Add(*serverIP, "nmsg.nicovideo.jp")

				if e := hosts.Flush(); e != nil {
					return fmt.Errorf("호스트 파일을 저장할 수 없습니다: %s", e)
				}

				log.Info("호스트 파일에 포워딩에 필요한 항목을 추가했습니다")
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

		if exists, e := certificate.InstallAsRootCA(cert); e == nil {
			if exists {
				log.Info("인증서가 이미 설치되어있습니다")
			} else {
				msg := []string{
					"인증서를 성공적으로 설치했습니다",
					"\t브라우저가 열려있을 때 인증서를 설치하면 캐시로 인해 인식되지 않을 수 있습니다",
					"\t코멘트가 보이지 않는다면 열린 브라우저 창을 모두 닫고 다시 열어주세요",
				}
				log.Info(strings.Join(msg, "\n"))
			}
		} else {
			return nil, nil, fmt.Errorf("인증서를 설치할 수 없습니다: %s", e)
		}
	}

	return cert, priv, nil
}

func handle(w http.ResponseWriter, r *http.Request) {
	var e error
	var status = http.StatusOK
	var prefix = fmt.Sprintf("%s - %s - %s", r.RemoteAddr, r.URL.Path, r.Referer())

	w.Header().Set("Access-Control-Allow-Origin", "*")

	defer func() {
		if e != nil {
			status = http.StatusInternalServerError
			log.Error(prefix, e)
		}

		log.Infof("%s : %d", prefix, status)

		w.WriteHeader(status)
		r.Body.Close()
	}()

	if r.URL.Path != "/api.json/" {
		status = http.StatusNotFound
		return
	}

	if r.Method != http.MethodPost {
		status = http.StatusBadRequest
		return
	}

	// 받은 데이터를 기존 API 서버로 포워딩한 뒤 데이터 불러오기
	message := <-nico.Fetch(r.Body)
	if message.Error != nil {
		e = message.Error
		return
	}

	queries := make([]string, len(message.Chats))
	for index, chat := range message.Chats {
		queries[index] = fmt.Sprintf("§%d\n%s\n", index, chat.Content)
	}

	log.Infof("%s : 코멘트 %d개", prefix, len(message.Chats))

	// 번역하기
	translated := <-translator.Translate(queries, *langPlatform, *langSource, *langTarget)
	if translated.Error != nil {
		e = translated.Error
		return
	}

	var translatedBytes bytes.Buffer
	for _, seq := range translated.Sequences {
		translatedBytes.WriteString(seq.Translated)
	}

	for _, groups := range queriesPattern.FindAllStringSubmatch(translatedBytes.String(), -1) {
		index, _ := strconv.Atoi(groups[1])
		// fmt.Printf("<<< {%d} %s\n", index, message.Chats[index].Content)
		// fmt.Printf(">>> {%d} %s\n", index, groups[2])
		message.Chats[index].Content = groups[2]
	}

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
