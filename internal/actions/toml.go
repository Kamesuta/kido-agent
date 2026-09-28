package actions

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type kidoToml struct {
	Name    *string `toml:"name"`
	Icon    *string `toml:"icon"`
	Confirm *string `toml:"confirm"`
	Run     *string `toml:"run"`
}

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// readToml は kido.toml を読んで a に反映する。無ければ何もしない。
// 読めなければ押せない操作にする(利用者の意図と違うものを動かさないため)。
func readToml(a *Action) kidoToml {
	var cfg kidoToml
	data, err := os.ReadFile(filepath.Join(a.Dir, TomlName))
	if errors.Is(err, fs.ErrNotExist) {
		return cfg
	}
	if err != nil {
		a.fail("kido.toml が読めません: "+err.Error(), "ファイルの権限を確かめてください")
		return cfg
	}
	// Windows のメモ帳は BOM 付きで保存することがあるので、先に外す。
	meta, err := toml.Decode(string(bytes.TrimPrefix(data, utf8BOM)), &cfg)
	if err != nil {
		a.fail("kido.toml の書き方が違います: "+err.Error(),
			"文字は \"…\" で囲みます。例: name = \"ゲーム起動\"")
		return kidoToml{}
	}
	for _, k := range meta.Undecoded() {
		a.warn("kido.toml の知らないキーは無視します: " + k.String() + "(使えるのは name icon confirm run)")
	}
	applyToml(a, cfg)
	return cfg
}

func applyToml(a *Action, cfg kidoToml) {
	if cfg.Name != nil && strings.TrimSpace(*cfg.Name) != "" {
		a.Name = strings.TrimSpace(*cfg.Name)
	}
	if cfg.Icon != nil {
		if validIcon(*cfg.Icon) {
			a.Icon = *cfg.Icon
		} else {
			a.warn("icon の名前が使えないので省きます: \"" + *cfg.Icon +
				"\"(英小文字・数字・- だけ、40 文字まで。https://lucide.dev/icons の名前)")
		}
	}
	if cfg.Confirm != nil {
		c, cut := truncateRunes(*cfg.Confirm, MaxConfirm)
		if cut {
			a.warn("confirm が長いので 200 文字で切ります")
		}
		a.Confirm = &c
	}
}
