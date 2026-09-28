package auth

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"sync"
	"time"
)

const (
	NonceTTL = 30 * time.Second
	NonceMax = 32
)

// NonceStore は hello で出した nonce を覚えておく。
// 盗み聞きした要求をそのまま送り直されても通らないよう、1 回使ったら捨てる。
type NonceStore struct {
	mu     sync.Mutex
	now    func() time.Time
	rand   io.Reader
	issued []issuedNonce // 出した順。上限を超えたら先頭(古いもの)から捨てる
}

type issuedNonce struct {
	value string
	at    time.Time
}

func NewNonceStore(now func() time.Time) *NonceStore {
	return &NonceStore{now: now, rand: rand.Reader}
}

func (s *NonceStore) Issue() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(s.rand, buf); err != nil {
		return "", err
	}
	v := hex.EncodeToString(buf)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropExpired()
	s.issued = append(s.issued, issuedNonce{v, s.now()})
	if len(s.issued) > NonceMax {
		s.issued = s.issued[len(s.issued)-NonceMax:]
	}
	return v, nil
}

// Take は nonce が生きていれば true を返し、成否を問わずその場で捨てる。
// 署名が違っても捨てるのは、同じ nonce で何度も試されないようにするため。
func (s *NonceStore) Take(v string) bool {
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

func (s *NonceStore) dropExpired() {
	now := s.now()
	keep := s.issued[:0]
	for _, n := range s.issued {
		if now.Sub(n.at) < NonceTTL {
			keep = append(keep, n)
		}
	}
	s.issued = keep
}
