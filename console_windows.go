package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetConsoleProcessList = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleProcessList")

// pauseIfOwnConsole は、この画面(コンソール)を自分だけが使っているとき
// (「アプリ」一覧のアンインストールから起動されたときなど)に Enter を待つ。
// 待たないと、結果を読む前に窓が閉じてしまう。
func pauseIfOwnConsole(out io.Writer) {
	var ids [2]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&ids[0])), 2)
	if n != 1 {
		return
	}
	fmt.Fprint(out, "\nEnter キーを押すと閉じます")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
