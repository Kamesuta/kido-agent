package launch

import "testing"

func TestBatchCmdLine(t *testing.T) {
	got := batchCmdLine(`C:\Windows\system32\cmd.exe`, `C:\Users\a\KidoButtons\60_A&B(1)\start.bat`)
	want := `"C:\Windows\system32\cmd.exe" /d /v:off /s /c "call "C:\Users\a\KidoButtons\60_A&B(1)\start.bat""`
	if got != want {
		t.Fatalf("%s", got)
	}
}

func TestCmdPathProblem(t *testing.T) {
	for _, p := range []string{`C:\K\60_A&B(1)\s.bat`, `C:\K\60 あ\s.bat`, `C:\K\60!\s.bat`} {
		if CmdPathProblem(p) != "" {
			t.Errorf("%s は渡せるはず", p)
		}
	}
	for _, p := range []string{`C:\K\60_100%\s.bat`, `C:\K\60_%PATH%\s.bat`, `C:\K\60_a^b\s.bat`} {
		if CmdPathProblem(p) == "" {
			t.Errorf("%s は渡せないはず", p)
		}
	}
}
