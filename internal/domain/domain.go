// Package domain holds the core types shared by storage and transport layers.
// Money is always stored as integer cents.
package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

// ValidationError maps field names to human-readable problems.
type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	slices.Sort(parts)
	return "validation failed: " + strings.Join(parts, "; ")
}

func NewValidationError(field, msg string) *ValidationError {
	return &ValidationError{Fields: map[string]string{field: msg}}
}

type validator map[string]string

func (v validator) check(ok bool, field, msg string) {
	if !ok {
		if _, exists := v[field]; !exists {
			v[field] = msg
		}
	}
}

func (v validator) err() error {
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Fields: v}
}

// normalizeTags trims, lowercases and de-duplicates tags, preserving order.
func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}
