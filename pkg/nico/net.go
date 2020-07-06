package nico

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type hostCache struct {
	checked time.Time
	addr    string
}

var hostCaches = map[string]hostCache{
	"nmsg.nicovideo.jp": {},
}

var dialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
	DualStack: true,
}

var transport = &http.Transport{
	Dial: dialer.Dial,
	DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		address := strings.Split(addr, ":")

		// 수동으로 업데이트할 호스트 주소라면 호스트 캐시 확인하기
		if d, ok := hostCaches[address[0]]; ok {
			// 60초 이상 지났다면 DNS 업데이트하기
			if time.Since(d.checked).Seconds() >= 60 {
				// DNS 서버에 A 레코드로 요청하기
				c := dns.Client{}
				m := dns.Msg{}
				m.SetQuestion(address[0]+".", dns.TypeA)

				r, _, e := c.Exchange(&m, "1.1.1.1:53")
				if e != nil {
					// TODO: 더 나은 오류 핸들링
					panic(e)
				}

				// 불러온 아이피 캐시하기
				addr = r.Answer[0].(*dns.A).A.String() + ":" + address[1]
				hostCaches[address[0]] = hostCache{
					addr:    addr,
					checked: time.Now(),
				}
			} else {
				// 캐시된 호스트의 아이피 주소 사용하기
				addr = d.addr
			}
		}

		return dialer.DialContext(ctx, network, addr)
	},
}

var Net = &http.Client{Transport: transport}
