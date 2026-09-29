//go:build !windows

package server

// dropPrivilege は Windows でだけ意味がある。Mac・Linux では何もしない。
func dropPrivilege() bool { return false }
