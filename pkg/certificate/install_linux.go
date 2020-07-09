package certificate

import (
	"crypto/x509"
	"fmt"
)

// InstallAsRootCA 인증서를 신뢰할 수 있는 루트 인증 기관으로 설치합니다
func InstallAsRootCA(cert *x509.Certificate) error {
	return fmt.Errorf("윈도우 환경에서만 사용할 수 있는 메소드입니다")
}
