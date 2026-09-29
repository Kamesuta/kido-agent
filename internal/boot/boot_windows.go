package boot

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kido-agent/internal/control"

	"golang.org/x/sys/windows"
)

//go:embed register.ps1 unregister.ps1
var scripts embed.FS

const taskName = "KidoAgent"

// enable は起動時タスクを登録する。登録は管理者(UAC)が要るので、昇格した
// PowerShell に任せる。登録できたら今の常駐を止めてタスクを起こし、ログイン中なら
// もう一度 kido-agentd を起こして手足役にする(exe の入れ替えではないが、
// 待ち受け役をタスク側へ移すため)。
func enable(out io.Writer) int {
	exe, err := daemonPath()
	if err != nil {
		fmt.Fprintln(out, "✗", err)
		return 1
	}
	user := whoami()
	fmt.Fprintln(out, "ログイン前(電源を入れただけ)でも使えるようにします。")
	fmt.Fprintln(out, "管理者の確認(UAC)が出たら「はい」を押してください。")

	// タスクが今の窓口を取れるよう、先に今の待ち受け役を止める。
	control.Call("POST", "/control/stop")
	waitPortFree()

	if msg, err := runElevated("register.ps1", "-Exe", exe, "-User", user); err != nil {
		fmt.Fprintln(out, "✗ 起動時タスクを登録できませんでした:")
		fmt.Fprintln(out, indent(msg))
		return 1
	}
	// タスクが待ち受け役として立ち上がるのを待つ。
	if !waitListener() {
		fmt.Fprintln(out, "! 登録はできましたが、待ち受け役の立ち上がりを確認できませんでした。")
		fmt.Fprintln(out, "  一度サインアウトして入り直すか、PC を再起動してください。")
		return 0
	}
	// ログイン中の画面で操作を動かせるよう、手足役として起こす。
	startHelper(exe)
	fmt.Fprintln(out, "✓ ログイン前でも使えるようにしました。")
	return 0
}

func disable(out io.Writer) int {
	exe, _ := daemonPath()
	fmt.Fprintln(out, "ログイン前には動かさないようにします。管理者の確認(UAC)が出たら「はい」を押してください。")
	if msg, err := runElevated("unregister.ps1"); err != nil {
		fmt.Fprintln(out, "✗ 起動時タスクを消せませんでした:")
		fmt.Fprintln(out, indent(msg))
		return 1
	}
	// タスク側の待ち受け役を止め、ログイン中なら普通の常駐として起こし直す。
	control.Call("POST", "/control/stop")
	waitPortFree()
	if exe != "" {
		startHelper(exe)
	}
	fmt.Fprintln(out, "✓ ログイン前には動かさないようにしました(ログイン中は今までどおり動きます)。")
	return 0
}

func status(out io.Writer) int {
	cmd := exec.Command("schtasks", "/query", "/tn", taskName)
	cmd.SysProcAttr = hidden()
	if cmd.Run() == nil {
		fmt.Fprintln(out, "ログイン前: 動きます(起動時タスク KidoAgent あり)")
	} else {
		fmt.Fprintln(out, "ログイン前: 動きません(kido-agent boot on で有効にできます)")
	}
	return 0
}

func usage(out io.Writer) int {
	fmt.Fprintln(out, "使い方: kido-agent boot [on|off]")
	return 2
}

func daemonPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "kido-agentd.exe"), nil
}

// whoami は COMPUTERNAME\USER の形の名前を返す。USERDOMAIN は SSH などで
// 食い違う実績があるので使わない。
func whoami() string {
	out, err := exec.Command("whoami").Output()
	if err != nil {
		return os.Getenv("USERNAME")
	}
	return strings.TrimSpace(string(out))
}

// runElevated は埋め込んだ PowerShell を管理者の権限で実行する。すでに昇格して
// いれば(CI ランナーなど)そのまま実行し、していなければ UAC で昇格する。
// 失敗したときは、スクリプトの出力(例外メッセージ)を err と一緒に返す。
func runElevated(script string, args ...string) (output string, err error) {
	data, e := scripts.ReadFile(script)
	if e != nil {
		return "", e
	}
	tmp := filepath.Join(os.TempDir(), "kido-"+script)
	if e := os.WriteFile(tmp, data, 0o600); e != nil {
		return "", e
	}
	defer os.Remove(tmp)

	var cmd *exec.Cmd
	if isElevated() {
		// すでに管理者。RunAs を挟むと(UAC を切った環境で)昇格の段が空振りする
		// ことがあるので、同じプロセスでそのまま実行する。
		psArgs := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmp}, args...)
		cmd = exec.Command("powershell.exe", psArgs...)
	} else {
		// 管理者でない。UAC を出して昇格し、終わるのを待って終了コードを引き継ぐ。
		argList := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmp}, args...)
		quoted := make([]string, len(argList))
		for i, a := range argList {
			quoted[i] = "'" + strings.ReplaceAll(a, "'", "''") + "'"
		}
		ps := fmt.Sprintf(
			"$p = Start-Process powershell -Verb RunAs -Wait -PassThru -ArgumentList %s; exit $p.ExitCode",
			strings.Join(quoted, ","))
		cmd = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps)
	}
	cmd.SysProcAttr = hidden()
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err = cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil && out == "" {
		out = err.Error()
	}
	return out, err
}

// isElevated は今のプロセスが管理者の権限で動いているか。
func isElevated() bool { return windows.GetCurrentProcessToken().IsElevated() }

// indent はメッセージを字下げして見やすくする。
func indent(s string) string {
	if s == "" {
		return "    (詳細なし)"
	}
	return "    " + strings.ReplaceAll(s, "\n", "\n    ")
}

func startHelper(exe string) {
	cmd := exec.Command(exe)
	cmd.SysProcAttr = hidden()
	if cmd.Start() == nil {
		go cmd.Wait()
	}
}

func waitListener() bool {
	for i := 0; i < 50; i++ {
		if _, err := control.Call("GET", "/control/status"); err == nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func waitPortFree() {
	for i := 0; i < 30; i++ {
		if _, err := control.Call("GET", "/control/status"); err != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
