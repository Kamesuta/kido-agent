//go:build !windows

package uninstall

import "io"

// StopDaemons は Mac・Linux では何もしない。常駐アプリは LaunchAgent・systemd --user の
// 1 つだけで、自動起動を外す(bootout・disable --now)時に止まる。手足役も生まれない。
func StopDaemons(io.Writer) {}
