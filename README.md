nicotrans-go
---
니코니코동화 코멘트 번역기

## 소개

파파고 번역기의 비공식 API 를 사용해 니코니코동화 코멘트를 번역합니다.

## 설치

[설치 동영상](https://youtu.be/UP-BTlps2rk?t=60)

### 인증서 설치

* 생성된 인증서를 루트 인증 기관으로 설치하기 (기본: `server.crt`)
  * Windows: `인증서 파일 오른쪽 클릭 후 인증서 설치 클릭`

## 사용법

```
Usage of nicotrans:
  -cert string
        서버 SSL 인증서 경로 (default "server.crt")
  -cert-create
        서버 SSL 인증서가 존재하지 않을 때 생성할지? (default true)
  -cert-privatekey string
        서버 SSL 인증서 키 경로 (default "server.key")
  -edit-hosts
        호스트 파일에 자동으로 아이피를 추가할지? (default true)
  -ip string
        서버 주소 (default "127.0.0.1")
  -lang-platform string
        사용될 번역기 종류 (default "papago")
  -lang-source string
        번역할 언어 2자리 코드 (default "ja")
  -lang-target string
        번역될 언어 2자리 코드 (default "ko")
  -port int
        서버 포트 (default 443)
```

## 할 일
- [x] Naver Papago
- [ ] Google Translator
- [ ] Bing Microsoft Translator
- [ ] Yandex.Translate
- [x] 비동기화
- [x] 더 나은 오류 핸들링
- [x] 인증서 생성 및 호스트 파일 수정 자동화
- [ ] GUI?
