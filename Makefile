run:
	go run main.go

cert:
	openssl genrsa -out server.key 2048
	openssl req -new \
		-subj "/O=NicoTrans" \
		-addext "subjectAltName = DNS:nmsg.nicovideo.jp" \
		-x509 -sha256 -days 3650 -key server.key -out server.crt \

build:
	mkdir -p dist
	go build -ldflags "-s -w" main.go

all:
	make build
