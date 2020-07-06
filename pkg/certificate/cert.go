package certificate

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

func exportPublicKey(priv crypto.PrivateKey) (crypto.PublicKey, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	default:
		return nil, fmt.Errorf("Unsupported key: %T", k)
	}
}

func exportPemBlock(priv crypto.PrivateKey) (*pem.Block, error) {
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

// Create 인증서와 키를 생성합니다
func Create() ([]byte, interface{}, error) {
	priv, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		return nil, nil, e
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"NicoTrans"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	cert, e := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if e != nil {
		return nil, nil, e
	}

	return cert, priv, nil
}

// Export 인증서와 키를 저장합니다
func Export(cert []byte, priv interface{}, certPath string, privPath string) error {
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

	privBlock, e := exportPemBlock(priv)
	if e != nil {
		return e
	}

	if e := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); e != nil {
		return e
	}

	if e := pem.Encode(privFile, privBlock); e != nil {
		return e
	}

	return e
}

// ImportPemBlock Pem 블록으로 된 인증서와 키를 불러옵니다
func ImportPemBlock(certPath string, privPath string) ([]byte, interface{}, error) {
	// 파일 뜯어오기
	certFile, e := ioutil.ReadFile(certPath)
	if e != nil {
		return nil, nil, e
	}

	privFile, e := ioutil.ReadFile(privPath)
	if e != nil {
		return nil, nil, e
	}

	// PEM 블록 불러오기
	certBlock, _ := pem.Decode(certFile)
	if certBlock == nil {
		return nil, nil, errors.New("Failed to parse certificate")
	}

	privBlock, _ := pem.Decode(privFile)
	if privBlock == nil {
		return nil, nil, errors.New("Failed to parse private key")
	}

	// 파싱하기
	template, e := x509.ParseCertificate(certBlock.Bytes)
	if e != nil {
		return nil, nil, e
	}

	var priv crypto.PrivateKey

	switch privBlock.Type {
	case "RSA PRIVATE KEY":
		priv, e = x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	case "EC PRIVATE KEY":
		priv, e = x509.ParseECPrivateKey(privBlock.Bytes)
	default:
		e = errors.New("Unsupported key")
	}

	if e != nil {
		return nil, nil, e
	}

	pub, e := exportPublicKey(priv)
	if e != nil {
		return nil, nil, e
	}

	cert, e := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if e != nil {
		return nil, nil, e
	}

	return cert, &priv, nil
}
