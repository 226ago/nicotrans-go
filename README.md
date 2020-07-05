nicotrans-go
---
니코니코동화 코멘트 번역기

## 소개

파파고 번역기의 비공식 API 를 사용해 니코니코동화 코멘트를 번역합니다.

## 설치

[설치 동영상](https://youtu.be/UP-BTlps2rk)

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
  -ip string
        서버 주소 (default "0.0.0.0")
  -langsrc string
        번역할 언어 2자리 코드 (default "ja")
  -langtarget string
        번역될 언어 2자리 코드 (default "ko")
  -port int
        서버 포트 (default 443)
  -sslcert string
        서버 SSL 인증서 경로 (default "server.crt")   
  -sslkey string
        서버 SSL 인증서 키 경로 (default "server.key")
  -translator string
        사용될 번역기 종류 (default "papago")
```

## 할 일
- [x] Naver Papago
- [ ] Google Translator
- [ ] Bing Microsoft Translator
- [ ] Yandex.Translate
- [x] 비동기화
- [ ] 더 나은 오류 핸들링
- [ ] 인증서 생성 및 호스트 파일 수정 자동화
- [ ] GUI?
