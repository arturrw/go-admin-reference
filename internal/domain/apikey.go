package domain

import (
	"strings"
	"time"
)

// API key scopes: what a key may do. A "read" key acts like a viewer, a
// "write" key like an editor. Keys never get settings or team access, so a
// leaked key can't mint more keys.
const (
	KeyScopeRead  = "read"
	KeyScopeWrite = "write"
)

// KeyPrefix starts every secret, so leaked keys are easy to spot in logs.
const KeyPrefix = "ga_live_"

// APIKey is a server-to-server credential. Only a hash of the secret is
// stored; the secret itself is shown once, when the key is created.
type APIKey struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Scope      string     `json:"scope"`
	Last4      string     `json:"last4"` // for recognising the key in a list
	CreatedBy  string     `json:"createdBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	RevokedAt  *time.Time `json:"-"` // revoked keys are not listed
	Hash       []byte     `json:"-"`
}

// Masked is how a key is displayed: ga_live_••••••••3f9a.
func (k APIKey) Masked() string { return KeyPrefix + "••••••••" + k.Last4 }

// Principal is the member a request made with this key acts as.
func (k APIKey) Principal() Member {
	role := RoleViewer
	if k.Scope == KeyScopeWrite {
		role = RoleEditor
	}
	return Member{Name: "API key “" + k.Name + "”", Email: "key:" + k.Name, Role: role, Status: MemberActive, Granted: []Permission{}, Revoked: []Permission{}}
}

type APIKeyInput struct {
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

func (in *APIKeyInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	if in.Scope == "" {
		in.Scope = KeyScopeRead
	}
}

func (in APIKeyInput) Validate() error {
	v := validator{}
	v.check(in.Name != "", "name", "is required")
	v.check(len([]rune(in.Name)) <= 60, "name", "must be at most 60 characters")
	v.check(in.Scope == KeyScopeRead || in.Scope == KeyScopeWrite, "scope", "must be read or write")
	return v.err()
}
