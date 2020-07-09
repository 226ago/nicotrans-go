package certificate

import (
	"crypto/x509"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	crypt32                              = syscall.NewLazyDLL("crypt32.dll")
	procCertAddEncodedCertificateToStore = crypt32.NewProc("CertAddEncodedCertificateToStore")
)

// InstallAsRootCA 인증서를 신뢰할 수 있는 루트 인증 기관으로 설치합니다
func InstallAsRootCA(cert *x509.Certificate) error {
	store, e := syscall.CertOpenStore(
		windows.CERT_STORE_PROV_SYSTEM, // LPCSTR lpszStoreProvider
		0,                              // DWORD dwEncodingType
		0,                              // HCRYPTPROV_LEGACY hCryptProv
		windows.CERT_STORE_OPEN_EXISTING_FLAG|windows.CERT_SYSTEM_STORE_CURRENT_USER, // DWORD dwFlags
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("root"))),                    // *pvPara
	)
	if e != nil {
		return e
	}

	defer syscall.CertCloseStore(store, 0)

	r, _, e := procCertAddEncodedCertificateToStore.Call(
		uintptr(store), // HCERTSTORE hCertStore
		uintptr(windows.X509_ASN_ENCODING|windows.PKCS_7_ASN_ENCODING), // DWORD dwCertEncodingType
		uintptr(unsafe.Pointer(&cert.Raw[0])),                          // const BYTE*pbCertEncoded
		uintptr(uint(len(cert.Raw))),                                   // DWORD cbCertEncoded
		windows.CERT_STORE_ADD_NEW,                                     // DWORD dwAddDisposition
		0,                                                              // PCCERT_CONTEXT *ppCertContext
	)
	if r == 0 {
		return e
	}

	return nil
}
