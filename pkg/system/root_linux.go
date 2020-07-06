package system

import "os/user"

// HasRoot 관리자 권한이 있는지?
func HasRoot() bool {
	u, e := user.Current()
	if e != nil {
		panic(e)
	}

	return u.Uid == "0"
}
