// Package auth provides password hashing and server-side sessions.
//
// Passwords use PBKDF2-HMAC-SHA256 from the standard library. Sessions are
// opaque random tokens sent to the browser as an HttpOnly cookie; the server
// keeps them in memory (MemorySessions) or in Postgres (postgres.Sessions).
package auth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CookieName = "goadmin_session"
	iterations = 120_000
	keyLen     = 32
)

var b64 = base64.RawStdEncoding

// HashPassword returns an encoded "pbkdf2-sha256$iter$salt$key" string.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, keyLen)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iterations, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// CheckPassword reports whether password matches the encoded hash.
func CheckPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 {
		return false
	}
	salt, err1 := b64.DecodeString(parts[2])
	want, err2 := b64.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

var ErrNoSession = errors.New("no session")

// NewToken returns a random session token for the cookie.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken is what gets persisted: a database leak can't be replayed as cookies.
func HashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// MemorySessions keeps sessions in process memory; they vanish on restart.
// The Postgres store provides a persistent implementation.
type session struct {
	memberID int64
	expires  time.Time
}

type MemorySessions struct {
	mu  sync.Mutex
	m   map[string]session
	ttl time.Duration
	now func() time.Time
}

func NewMemorySessions(ttl time.Duration) *MemorySessions {
	return &MemorySessions{m: map[string]session{}, ttl: ttl, now: time.Now}
}

func (s *MemorySessions) TTL() time.Duration { return s.ttl }

// Create starts a session and returns its token.
func (s *MemorySessions) Create(_ context.Context, memberID int64) (string, error) {
	token, err := NewToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[token] = session{memberID: memberID, expires: s.now().Add(s.ttl)}
	return token, nil
}

// Lookup returns the member for a valid token and extends its lifetime.
func (s *MemorySessions) Lookup(_ context.Context, token string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[token]
	if !ok {
		return 0, ErrNoSession
	}
	now := s.now()
	if now.After(sess.expires) {
		delete(s.m, token)
		return 0, ErrNoSession
	}
	sess.expires = now.Add(s.ttl)
	s.m[token] = sess
	return sess.memberID, nil
}

func (s *MemorySessions) Delete(_ context.Context, token string) {
	s.mu.Lock()
	delete(s.m, token)
	s.mu.Unlock()
}

// DeleteMember ends every session of a member (e.g. after removal or suspension).
func (s *MemorySessions) DeleteMember(_ context.Context, memberID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, sess := range s.m {
		if sess.memberID == memberID {
			delete(s.m, t)
		}
	}
}
