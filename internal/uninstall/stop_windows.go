package uninstall

import (
	"fmt"
	"io"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// StopDaemons は自分の利用者の kido-agentd を、待ち受け役も手足役も全部止める。
// 待ち受け役に止まってもらうだけでは、ログイン中の手足役が「待ち受け役が消えた」と
// 見て窓口を取り直し、取り除いた後も動き続ける(ファイルも使用中で消せない)。
// 同じ PC の別の利用者の常駐は止めない(止める権限も無い)。
func StopDaemons(out io.Writer) {
	if n, err := stopOwnProcesses("kido-agentd.exe"); err != nil {
		fmt.Fprintln(out, "! 常駐アプリを探せませんでした:", err)
	} else if n > 0 {
		fmt.Fprintf(out, "常駐アプリを %d つ止めました。\n", n)
	}
}

// stopOwnProcesses は、名前が exe で自分と同じ利用者(SID)のプロセスを止めて、
// 終わるのを少し待つ。止めた数を返す。自分自身は止めない。
func stopOwnProcesses(exe string) (int, error) {
	me, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return 0, err
	}
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snap)
	self := windows.GetCurrentProcessId()
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	n := 0
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		name := windows.UTF16ToString(e.ExeFile[:])
		if e.ProcessID == self || !strings.EqualFold(name, exe) {
			continue
		}
		if stopIfOwn(e.ProcessID, me.User.Sid) {
			n++
		}
	}
	return n, nil
}

// stopIfOwn は、そのプロセスの持ち主が sid なら止めて、終わるまで 3 秒まで待つ。
// 別の利用者のプロセスは開けない(開けても持ち主が違う)ので、何もしない。
func stopIfOwn(pid uint32, sid *windows.SID) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|
		windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var tok windows.Token
	if windows.OpenProcessToken(h, windows.TOKEN_QUERY, &tok) != nil {
		return false
	}
	u, err := tok.GetTokenUser()
	tok.Close()
	if err != nil || !windows.EqualSid(u.User.Sid, sid) {
		return false
	}
	if windows.TerminateProcess(h, 1) != nil {
		return false
	}
	windows.WaitForSingleObject(h, 3000)
	return true
}
