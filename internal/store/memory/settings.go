package memory

import (
	"context"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

func (s *Store) GetSettings(_ context.Context) (d.Settings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings, nil
}

func (s *Store) SaveSettings(_ context.Context, v d.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = v
	return nil
}
