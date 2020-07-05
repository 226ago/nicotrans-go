// +build windows

package utils

import "os"

// HasRoot 관리자 권한이 있는지?
func HasRoot() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	return true
}
