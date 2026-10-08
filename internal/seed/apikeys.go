package seed

import (
	"crypto/rand"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
)

// APIKeys are the demo keys listed in Settings. Their secrets are random and
// thrown away, so no seeded key can be used to sign in: create a new one to
// try the API.
func APIKeys(now time.Time) []d.APIKey {
	at := func(dur time.Duration) *time.Time { t := now.Add(-dur); return &t }
	const day = 24 * time.Hour
	keys := []d.APIKey{
		{Name: "Storefront (read)", Scope: d.KeyScopeRead, Last4: "3f9a", CreatedBy: "Artur DCS", CreatedAt: now.Add(-54 * day), LastUsedAt: at(2 * time.Minute)},
		{Name: "Warehouse sync", Scope: d.KeyScopeWrite, Last4: "a71c", CreatedBy: "Mark Liu", CreatedAt: now.Add(-126 * day), LastUsedAt: at(time.Hour)},
		{Name: "CI smoke tests", Scope: d.KeyScopeRead, Last4: "0b2e", CreatedBy: "Mark Liu", CreatedAt: now.Add(-6 * day)},
	}
	for i := range keys {
		keys[i].Hash = make([]byte, 32)
		if _, err := rand.Read(keys[i].Hash); err != nil {
			panic(err)
		}
	}
	return keys
}
