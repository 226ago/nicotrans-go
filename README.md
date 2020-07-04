nicotrans-go
---
니코니코동화 코멘트 번역기

## 소개
파파고 번역기의 비공식 API 를 사용해 니코니코동화 코멘트를 번역합니다.

## 설치

### 호스트 수정

* 호스트 파일 열기
  * Windows: `%windir%\System32\drivers\etc\hosts`
  * Linux: `/etc/hosts`
  * Android: `/system/etc/hosts`
* `127.0.0.1 nmsg.nicovideo.jp` 줄 추가하기

### 인증서 설치

* 니코트랜스 실행 시 나오는 경로 확인하기 (기본: `server.crt`)
* 루트 인증 기관으로 인증서 설치하기
  * Windows: `인증서 파일 오른쪽 클릭 후 인증서 설치 클릭`

## 사용법

```
Usage of nicotrans:
  -certfile string
        인증서 파일 경로 (default "server.crt")
  -ip string
        웹 서버 아이피 (default "0.0.0.0")
  -keyfile string
        인증서 개인 키 파일 경로 (default "server.key")
  -port int
        웹 서버 포트 (default 443)
```