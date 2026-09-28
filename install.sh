#!/bin/sh
# 起動丸エージェントを Mac・Linux に入れる(入れ直し・更新にも使える)。
#
#   curl -fsSL https://pc.kido.page/install.sh | sh
#
# 手元の tar.gz(またはそれを展開したフォルダ)から入れるとき(リリース前の確かめなど):
#   KIDO_AGENT_TARBALL=/path/kido-agent-darwin-arm64.tar.gz sh install.sh
#
# 管理者の権限は要らない。Mac で再起動・シャットダウン用の許可を入れるときだけ sudo を聞く。
set -eu

BIN_DIR="$HOME/.local/bin"
BIN="$BIN_DIR/kido-agent"
CONTROL="http://127.0.0.1:47822/control"
LABEL="page.kido.agent"

say() { printf '%s\n' "$*"; }
die() { printf '✗ %s\n' "$*" >&2; exit 1; }

detect() {
	case "$(uname -s)" in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) die "この OS には対応していません: $(uname -s)" ;;
	esac
	case "$(uname -m)" in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) die "この CPU には対応していません: $(uname -m)" ;;
	esac
}

# fetch は入手元から実行ファイルを取り出し、$work/kido-agent に置く。
fetch() {
	src="${KIDO_AGENT_TARBALL:-}"
	if [ -z "$src" ]; then
		url="https://github.com/Kamesuta/kido-agent/releases/latest/download/kido-agent-$os-$arch.tar.gz"
		say "ダウンロードしています: $url"
		curl -fsSL "$url" -o "$work/pkg.tar.gz" ||
			die "ダウンロードできませんでした。公開前は KIDO_AGENT_TARBALL に手元の tar.gz を指定してください"
		src="$work/pkg.tar.gz"
	fi
	mkdir "$work/pkg"
	if [ -d "$src" ]; then
		cp -R "$src/." "$work/pkg/"
	else
		tar -xzf "$src" -C "$work/pkg"
	fi
	found=$(find "$work/pkg" -type f -name kido-agent | head -n 1)
	[ -n "$found" ] || die "kido-agent が見つかりません: $src"
	cp "$found" "$work/kido-agent"
	chmod 755 "$work/kido-agent"
}

# place は動いている実行ファイルを上書きせず、名前の付け替えで差し替える。
# 上書きすると動いているプロセスが壊れることがある。
place() {
	mkdir -p "$BIN_DIR"
	cp "$work/kido-agent" "$BIN.new"
	# ブラウザで落とした tar.gz から取り出すと隔離の印が付き、起動が止められるので外す
	[ "$os" = darwin ] && xattr -d com.apple.quarantine "$BIN.new" 2>/dev/null || true
	mv -f "$BIN.new" "$BIN"
}

stop_manual() {
	curl -fsS -m 3 -X POST -H 'X-Kido-Control: 1' "$CONTROL/stop" >/dev/null 2>&1 || true
}

autostart_darwin() {
	plist="$HOME/Library/LaunchAgents/$LABEL.plist"
	mkdir -p "$(dirname "$plist")"
	cat >"$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>$LABEL</string>
	<key>ProgramArguments</key><array><string>$BIN</string><string>serve</string></array>
	<key>RunAtLoad</key><true/>
	<key>KeepAlive</key><true/>
</dict>
</plist>
EOF
	launchctl bootstrap "gui/$(id -u)" "$plist"
}

autostart_linux() {
	unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
	mkdir -p "$unit_dir"
	cat >"$unit_dir/kido-agent.service" <<EOF
[Unit]
Description=起動丸エージェント

[Service]
ExecStart=$BIN serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF
	systemctl --user daemon-reload
	systemctl --user enable --now kido-agent.service
}

# sudoers_darwin は shutdown だけをパスワードなしで許す。これが無いと、
# 再起動・シャットダウンを「保存しますか」と聞くアプリに止められてしまう。
sudoers_darwin() {
	f=/etc/sudoers.d/kido-agent
	[ -f "$f" ] && return 0
	if ! (exec </dev/tty) 2>/dev/null; then
		say "! 端末から聞けないので、再起動・シャットダウン用の許可は入れませんでした"
		return 0
	fi
	say ""
	say "再起動・シャットダウンを確実に行うため、shutdown だけをパスワードなしで使えるようにします。"
	say "Mac のログインパスワードを 1 回だけ聞きます(スキップするには n)。"
	printf '入れますか? [Y/n] '
	read -r ans </dev/tty || ans=n
	case "$ans" in [nN]*)
		say "! スキップしました。再起動・シャットダウンがアプリに止められることがあります"
		return 0 ;;
	esac
	printf '%s ALL=(root) NOPASSWD: /sbin/shutdown\n' "$(id -un)" >"$work/sudoers"
	if /usr/sbin/visudo -cf "$work/sudoers" >/dev/null &&
		sudo install -m 0440 -o root -g wheel "$work/sudoers" "$f" </dev/tty; then
		say "✓ 入れました"
	else
		say "! 入れられませんでした。再起動・シャットダウンがアプリに止められることがあります"
	fi
}

# path_hint は kido-agent を名前だけで呼べるよう案内する(Mac は zsh の設定に書き足す)。
path_hint() {
	case ":$PATH:" in *":$BIN_DIR:"*) return 0 ;; esac
	if [ "$os" = darwin ] && [ "$(basename "${SHELL:-}")" = zsh ]; then
		if ! grep -qs '.local/bin' "$HOME/.zprofile"; then
			printf '\n# 起動丸エージェント(kido-agent)を名前だけで呼べるようにする\nexport PATH="$HOME/.local/bin:$PATH"\n' >>"$HOME/.zprofile"
		fi
		say "新しく開いたターミナルから kido-agent と打てます。"
	else
		say "kido-agent は $BIN にあります(次のログインからは名前だけで呼べることが多いです)。"
	fi
}

wait_agent() {
	i=0
	while [ $i -lt 50 ]; do
		status=$(curl -fsS -m 1 -H 'X-Kido-Control: 1' "$CONTROL/status" 2>/dev/null) && return 0
		sleep 0.1 2>/dev/null || sleep 1
		i=$((i + 1))
	done
	return 1
}

main() {
	detect
	work=$(mktemp -d)
	trap 'rm -rf "$work"' EXIT
	say "起動丸エージェントを入れます。"
	fetch
	# 自動起動を先に外して止める。止めるだけだと、OS がすぐ起こし直す。
	if [ "$os" = darwin ]; then
		launchctl bootout "gui/$(id -u)/$LABEL" 2>/dev/null || true
	else
		systemctl --user stop kido-agent.service 2>/dev/null || true
	fi
	stop_manual
	place
	say "入れました($("$BIN" version)): $BIN"
	[ "$os" = darwin ] && say "「受け入れる接続を許可しますか」と聞かれたら「許可」を押してください。"
	"autostart_$os"
	wait_agent || die "常駐アプリが起動しませんでした。ログ: ~/.config/kido-agent/kido-agent.log"
	[ "$os" = darwin ] && sudoers_darwin
	say ""
	# 更新で入れ直したときは組み直さない(pair は古い鍵を消してしまう)。
	case "$status" in
	*'"paired":true'*) say "起動丸の本体とはもう組めています。組み直すときは: kido-agent pair" ;;
	*) "$BIN" pair || true ;;
	esac
	"$BIN" open >/dev/null 2>&1 || true
	say ""
	say "操作フォルダ(~/Kido)を開きました。スクリプトやアプリを入れたフォルダを作ると、スマホに操作が増えます。"
	path_hint
	say "困ったときは: kido-agent check"
}

main "$@"
