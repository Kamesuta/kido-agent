package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"sync"
	"time"
)

const (
	nonceTTL = 30 * time.Second
	nonceMax = 32
)

// nonceStore は hello で出した nonce を覚えておく。
// 盗み聞きした要求をそのまま送り直されても通らないよう、1 回使ったら捨てる。
type nonceStore struct {
	mu     sync.Mutex
	now    func() time.Time
	rand   io.Reader
	issued []issuedNonce // 出した順。上限を超えたら先頭(古いもの)から捨てる
}

type issuedNonce struct {
	value string
	at    time.Time
}

func newNonceStore(now func() time.Time) *nonceStore {
	return &nonceStore{now: now, rand: rand.Reader}
}

func (s *nonceStore) issue() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(s.rand, buf); err != nil {
		return "", err
	}
	v := hex.EncodeToString(buf)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropExpired()
	s.issued = append(s.issued, issuedNonce{v, s.now()})
	if len(s.issued) > nonceMax {
		s.issued = s.issued[len(s.issued)-nonceMax:]
	}
	return v, nil
}

// take は nonce が生きていれば true を返し、成否を問わずその場で捨てる。
// 署名が違っても捨てるのは、同じ nonce で何度も試されないようにするため。
func (s *nonceStore) take(v string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropExpired()
	for i, n := range s.issued {
		if n.value == v {
			s.issued = append(s.issued[:i], s.issued[i+1:]...)
			return true
		}
	}
	return false
}

func (s *nonceStore) dropExpired() {
	now := s.now()
	keep := s.issued[:0]
	for _, n := range s.issued {
		if now.Sub(n.at) < nonceTTL {
			keep = append(keep, n)
		}
	}
	s.issued = keep
}
