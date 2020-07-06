package system

import "fmt"

// HasRoot 관리자 권한이 있는지?
func HasRoot() (bool, error) {
	return false, fmt.Errorf("윈도우 외 플랫폼에선  지원하지 않는 메소드입니다")
}

// RunMeElevated 현재 프로그램을 관리자 권한으로 다시 실행합니다
func RunMeElevated() error {
	return fmt.Errorf("윈도우 외 플랫폼에선  지원하지 않는 메소드입니다")
}

