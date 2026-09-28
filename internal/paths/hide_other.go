//go:build !windows

package paths

// Mac・Linux はドット付きの名前だけで隠れる。
func hide(string) error { return nil }
