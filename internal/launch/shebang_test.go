package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptCommand(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		body string
		want string // 解釈器と引数を | でつなぎ、最後がパス
	}{
		{"#!/bin/bash\necho hi\n", "/bin/bash|P"},
		{"#!/usr/bin/env bash\n", "/usr/bin/env|bash|P"},
		{"#! /bin/zsh -e\n", "/bin/zsh|-e|P"},
		{"#!/usr/bin/env -S bash -e\n", "/usr/bin/env|-S bash -e|P"}, // 後ろは 1 つの引数
		{"#!/bin/bash\r\necho hi\r\n", "/bin/bash|P"},
		{"#!/bin/sh\t-x\n", "/bin/sh|-x|P"},
		{"#!/bin/bash", "/bin/bash|P"}, // 改行なしの 1 行だけ
		{"echo hi\n", "/bin/sh|P"},
		{"", "/bin/sh|P"},
		{"#!\n", "/bin/sh|P"},
		{"\xEF\xBB\xBF#!/bin/bash\n", "/bin/sh|P"}, // BOM 付きはカーネルも #! と読まない
	}
	for i, c := range cases {
		p := filepath.Join(dir, "s.sh")
		if err := os.WriteFile(p, []byte(c.body), 0o644); err != nil {
			t.Fatal(err)
		}
		name, args := scriptCommand(p)
		got := strings.ReplaceAll(strings.Join(append([]string{name}, args...), "|"), p, "P")
		if got != c.want {
			t.Errorf("%d %q: %s", i, c.body, got)
		}
	}
	if name, args := scriptCommand(filepath.Join(dir, "none.sh")); name != "/bin/sh" || len(args) != 1 {
		t.Fatalf("読めなければ /bin/sh: %s %v", name, args)
	}
}
