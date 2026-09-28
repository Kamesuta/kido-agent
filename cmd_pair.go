package main

import (
	"errors"
	"fmt"
	"io"
	"time"
)

func cmdPair(out io.Writer) int {
	st, err := callControl("POST", "/control/pair")
	if err != nil {
		reportControlError(out, err)
		return 1
	}
	fmt.Fprintln(out, "起動丸の本体と組みます。")
	fmt.Fprintln(out, "スマホで起動ページを開き、この PC を登録してください(10 分以内)。")
	return waitPairing(out, st, func() (controlStatus, error) {
		return callControl("GET", "/control/status")
	}, time.Sleep)
}

// waitPairing は組めるか時間切れになるまで様子を見る。
// 問い合わせと待ち方を引数にしているのは、テストで 10 分待たずに済ませるため。
func waitPairing(out io.Writer, st controlStatus, get func() (controlStatus, error), sleep func(time.Duration)) int {
	for {
		switch {
		case st.Paired:
			fmt.Fprintf(out, "\r%-40s\n", "✓ 本体とつながりました")
			return 0
		case st.State != statePairing:
			fmt.Fprintln(out)
			printPairTimeout(out)
			return 1
		}
		fmt.Fprintf(out, "\r  本体からの連絡を待っています… 残り %d:%02d ", st.Remaining/60, st.Remaining%60)
		sleep(time.Second)
		var err error
		if st, err = get(); err != nil {
			fmt.Fprintln(out)
			reportControlError(out, err)
			return 1
		}
	}
}

func printPairTimeout(out io.Writer) {
	fmt.Fprintln(out, "✗ 10 分のあいだに本体から届きませんでした。次を確かめてください。")
	fmt.Fprintln(out, "  ・スマホの起動ページで、この PC を登録したか")
	fmt.Fprintln(out, "  ・ファイアウォールの確認で「許可」を押したか")
	if hint := firewallHint(); hint != "" {
		fmt.Fprintln(out, "    "+hint)
	}
	fmt.Fprintln(out, "  ・起動丸の本体の電源が入っていて、この PC と同じネットワークにいるか")
	fmt.Fprintln(out, "もう一度組むには、次を実行してください:")
	fmt.Fprintln(out, "  kido-agent pair")
}

// reportControlError は常駐アプリに届かなかったときの案内を出す。
func reportControlError(out io.Writer, err error) {
	if !errors.Is(err, errNotRunning) {
		fmt.Fprintln(out, "✗", err)
		return
	}
	fmt.Fprintln(out, "✗ 起動丸エージェント(常駐アプリ)が動いていません。")
	fmt.Fprintln(out, "  次のどちらかで起動してから、もう一度試してください。")
	fmt.Fprintln(out, "  ・PC からサインアウトして、サインインし直す")
	fmt.Fprintln(out, "  ・"+startHint())
}
