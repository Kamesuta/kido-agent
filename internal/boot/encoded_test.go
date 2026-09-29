package boot

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"unicode/utf16"
)

func decodeCommand(t *testing.T, s string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(raw)%2 != 0 {
		t.Fatalf("base64: %v", err)
	}
	u := make([]uint16, len(raw)/2)
	for i := range u {
		u[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	return string(utf16.Decode(u))
}

func TestEncodedCommand(t *testing.T) {
	got := decodeCommand(t, encodedCommand("# 説明\r\nparam($Exe, $User)\r\n\r\n  # 字下げのコメント\r\nWrite-Host '起動丸' # 行末は残す\r\n",
		"-Exe", `C:\Users\O'Neil’s\kido-agentd.exe`, "-User", `PC\あ`))
	want := "& {\nparam($Exe, $User)\nWrite-Host '起動丸' # 行末は残す\n} -Exe 'C:\\Users\\O''Neil’’s\\kido-agentd.exe' -User 'PC\\あ'"
	if got != want {
		t.Fatalf("\n%s\n%s", got, want)
	}
}

// 登録のスクリプトを長いパスで包んでも、昇格の手前の命令行(外側の powershell と
// Start-Process の引数)が Windows の上限に収まる。
func TestEncodedRegisterFitsCommandLine(t *testing.T) {
	data, err := os.ReadFile("register.ps1")
	if err != nil {
		t.Fatal(err)
	}
	exe := `C:\Users\` + strings.Repeat("あ", 240) + `\AppData\Local\Programs\kido-agent\kido-agentd.exe`
	enc := encodedCommand(string(data), "-Exe", exe, "-User", `COMPUTER\`+strings.Repeat("u", 104))
	outer := `powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "$p = Start-Process powershell -Verb RunAs -Wait -PassThru -ArgumentList '-NoProfile','-ExecutionPolicy','Bypass','-EncodedCommand','` + enc + `'; exit $p.ExitCode"`
	if len(outer) > maxCommandLine/2 {
		t.Fatalf("命令行が %d 文字(上限 %d の半分を超えた)", len(outer), maxCommandLine)
	}
	t.Logf("命令行は %d 文字", len(outer))
	if !strings.Contains(decodeCommand(t, enc), "Register-ScheduledTask") {
		t.Fatal("中身が入っていない")
	}
}
