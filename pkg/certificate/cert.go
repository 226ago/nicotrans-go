package certificate

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

func exportPublicKey(priv interface{}) (interface{}, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	default:
		return nil, fmt.Errorf("Unsupported key: %T", k)
	}
}

func exportPemBlock(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, e := x509.MarshalECPrivateKey(k)
		if e != nil {
			return nil, e
		}

		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, fmt.Errorf("Unsupported key: %T", k)
	}
}

// Create 메소드는 인증서와 키를 생성합니다
func Create(template *x509.Certificate) (*x509.Certificate, interface{}, error) {
	priv, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		return nil, nil, e
	}

	// template := &x509.Certificate{
	// 	SerialNumber: big.NewInt(1),
	// 	Subject: pkix.Name{
	// 		Organization: []string{"NicoTrans"},
	// 	},
	// 	DNSNames:    names,
	// 	NotBefore:   time.Now(),
	// 	NotAfter:    time.Now().AddDate(10, 0, 0),
	// 	KeyUsage:    x509.KeyUsageDigitalSignature,
	// 	ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	// 	IsCA:        true,
	// }

	certBytes, e := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if e != nil {
		return nil, nil, e
	}

	cert, e := x509.ParseCertificate(certBytes)
	if e != nil {
		return nil, nil, e
	}

	return cert, priv, nil
}

// Export 인증서와 키를 저장합니다
func Export(cert *x509.Certificate, priv interface{}, certPath string, privPath string) error {
	// 저장될 파일 열기
	certFile, e := os.OpenFile(certPath, os.O_CREATE|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}

	defer certFile.Close()

	privFile, e := os.OpenFile(privPath, os.O_CREATE|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}

	defer privFile.Close()

	// PEM 블록 만들기
	privBlock, e := exportPemBlock(priv)
	if e != nil {
		return e
	}

	// 저장하기
	if e := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); e != nil {
		return e
	}

	if e := pem.Encode(privFile, privBlock); e != nil {
		return e
	}

	return nil
}

// Import 인증서와 키를 파일에서 불러옵니다
func Import(certPath string, privPath string) (*x509.Certificate, interface{}, error) {
	// 파일 불러오기
	certFile, e := ioutil.ReadFile(certPath)
	if e != nil {
		return nil, nil, e
	}

	privFile, e := ioutil.ReadFile(privPath)
	if e != nil {
		return nil, nil, e
	}

	// PEM 블록 디코딩하기
	certBlock, _ := pem.Decode(certFile)
	if certBlock == nil {
		return nil, nil, errors.New("인증서의 PEM 블록이 잘못됐습니다")
	}

	privBlock, _ := pem.Decode(privFile)
	if privBlock == nil {
		return nil, nil, errors.New("개인 키의 PEM 블록이 잘못됐습니다")
	}

	// 파싱하기
	cert, e := x509.ParseCertificate(certBlock.Bytes)
	if e != nil {
		return nil, nil, e
	}

	var priv interface{}

	switch privBlock.Type {
	case "RSA PRIVATE KEY":
		priv, e = x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	case "EC PRIVATE KEY":
		priv, e = x509.ParseECPrivateKey(privBlock.Bytes)
	default:
		e = fmt.Errorf("Unsupported key: %T", priv)
	}

	if e != nil {
		return nil, nil, e
	}

	return cert, priv, nil
}
