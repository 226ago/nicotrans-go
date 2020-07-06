package system

import (
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// HasRoot 관리자 권한이 있는지?
func HasRoot() (bool, error) {
	// 윈도우 SID 불러오기
	// https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	var sid *windows.SID
	e := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if e != nil {
		return false, e
	}

	// 실행한 사용자가 관리자(SID=0) 인지?
	token := windows.Token(0)
	member, e := token.IsMember(sid)
	if e != nil {
		return false, e
	}

	return member, nil
}

// RunMeElevated 현재 프로그램을 관리자 권한으로 다시 실행합니다
func RunMeElevated() error {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	e := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if e != nil {
		return e
	}

	return nil
}
