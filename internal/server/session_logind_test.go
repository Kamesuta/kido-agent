package server

import "testing"

func TestLocalLoginSession(t *testing.T) {
	cases := []struct {
		props string
		want  bool
	}{
		{"Type=wayland\nClass=user\nState=active\nRemote=no\n", true},
		{"Type=x11\nClass=user\nState=online\nRemote=no\n", true}, // 別の利用者に切り替えて裏にいる
		{"Type=tty\nClass=user\nState=active\nRemote=no\n", true},
		{"Type=tty\nClass=user\nState=online\nRemote=no\n", false},
		{"Type=tty\nClass=user\nState=active\nRemote=yes\n", false}, // ssh
		{"Type=wayland\nClass=greeter\nState=active\nRemote=no\n", false},
		{"Type=unspecified\nClass=background\nState=active\nRemote=no\n", false},
		{"Type=wayland\nClass=user\nState=closing\nRemote=no\n", false},
		{"Type=wayland\nClass=user-early\nState=active\nRemote=no\n", true},
		{"Type=wayland\nClass=user-incomplete\nState=active\nRemote=no\n", false},
		{"", false},
	}
	for _, c := range cases {
		if got := localLoginSession(c.props); got != c.want {
			t.Errorf("%q: %v", c.props, got)
		}
	}
}
