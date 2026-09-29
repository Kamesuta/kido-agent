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

	if code, err := runElevated(out, "register.ps1", "-Exe", exe, "-User", user); err != nil || code != 0 {
		fmt.Fprintf(out, "✗ 起動時タスクを登録できませんでした(終了コード %d): %v\n", code, err)
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
	if code, err := runElevated(out, "unregister.ps1"); err != nil || code != 0 {
		fmt.Fprintf(out, "✗ 起動時タスクを消せませんでした: %v\n", err)
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

// runElevated は埋め込んだ PowerShell を管理者で実行し、終了コードを返す。
func runElevated(out io.Writer, script string, args ...string) (int, error) {
	data, err := scripts.ReadFile(script)
	if err != nil {
		return -1, err
	}
	tmp := filepath.Join(os.TempDir(), "kido-"+script)
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return -1, err
	}
	defer os.Remove(tmp)
	// Start-Process -Verb RunAs で UAC を出し、-Wait で終わるのを待ち、終了コードを取る。
	argList := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmp}, args...)
	quoted := make([]string, len(argList))
	for i, a := range argList {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", "''") + "'"
	}
	ps := fmt.Sprintf(
		"$p = Start-Process powershell -Verb RunAs -Wait -PassThru -ArgumentList %s; exit $p.ExitCode",
		strings.Join(quoted, ","))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err = cmd.Run()
	if buf.Len() > 0 {
		fmt.Fprint(out, buf.String())
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), nil // スクリプト側の終了コード(0 以外は失敗)
	}
	return 0, err
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
