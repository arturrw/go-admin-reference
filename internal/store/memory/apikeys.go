package memory

import (
	"bytes"
	"context"
	"slices"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// API keys, newest first. Revoked keys stay in the slice (RevokedAt set) but
// are neither listed nor accepted.

func (s *Store) ListAPIKeys(_ context.Context) ([]d.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []d.APIKey{}
	for _, k := range s.apiKeys {
		if k.RevokedAt == nil {
			out = append(out, k)
		}
	}
	return out, nil
}

func (s *Store) CreateAPIKey(_ context.Context, k d.APIKey) (d.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k.ID, k.CreatedAt = s.nextKeyID, s.now()
	s.nextKeyID++
	s.apiKeys = slices.Insert(s.apiKeys, 0, k)
	return k, nil
}

func (s *Store) APIKeyByHash(_ context.Context, hash []byte) (d.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.apiKeys {
		if k.RevokedAt == nil && bytes.Equal(k.Hash, hash) {
			return k, nil
		}
	}
	return d.APIKey{}, d.ErrNotFound
}

func (s *Store) TouchAPIKey(_ context.Context, id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := slices.IndexFunc(s.apiKeys, func(k d.APIKey) bool { return k.ID == id }); i >= 0 {
		t := s.now()
		s.apiKeys[i].LastUsedAt = &t
	}
}

func (s *Store) RevokeAPIKey(_ context.Context, id int64) (d.APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.apiKeys, func(k d.APIKey) bool { return k.ID == id && k.RevokedAt == nil })
	if i < 0 {
		return d.APIKey{}, d.ErrNotFound
	}
	t := s.now()
	s.apiKeys[i].RevokedAt = &t
	return s.apiKeys[i], nil
}
