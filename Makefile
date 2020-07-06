run:
	go run main.go

build:
	go build -ldflags "-s -w" cmd/nicotrans/main.go

all:
	make build
