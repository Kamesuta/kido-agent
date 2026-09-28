package main

import (
	"os"
	"path/filepath"
	"sync"
)

const logLimit = 1 << 20

// cappedLog はログを 1MB で 1 世代だけ回す。常駐アプリは何か月も動き続けるので、
// ログがディスクを食い続けないようにする。調べるには直近の分があれば足りる。
type cappedLog struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
}

func openLog(dir string) (*cappedLog, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	l := &cappedLog{path: filepath.Join(dir, "kido-agent.log")}
	return l, l.open()
}

func (l *cappedLog) open() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	l.f, l.size = f, info.Size()
	return nil
}

func (l *cappedLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.size+int64(len(p)) > logLimit {
		l.f.Close()
		os.Rename(l.path, l.path+".1") // Windows でも Go の Rename は上書きできる
		if err := l.open(); err != nil {
			return 0, err
		}
	}
	n, err := l.f.Write(p)
	l.size += int64(n)
	return n, err
}
