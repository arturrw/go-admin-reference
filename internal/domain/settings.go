package domain

import (
	"net/url"
	"strings"
)

// Settings are the workspace's runtime configuration, edited in Settings and
// shared by every instance through the store.
type Settings struct {
	ServiceName   string `json:"serviceName"`
	PublicBaseURL string `json:"publicBaseUrl"` // no trailing slash; "" = unset
}

func DefaultSettings() Settings { return Settings{ServiceName: "goadmin-api"} }

// SettingsPatch changes only the fields that are set.
type SettingsPatch struct {
	ServiceName   *string `json:"serviceName"`
	PublicBaseURL *string `json:"publicBaseUrl"`
}

func (p *SettingsPatch) Normalize() {
	if p.ServiceName != nil {
		v := strings.TrimSpace(*p.ServiceName)
		p.ServiceName = &v
	}
	if p.PublicBaseURL != nil {
		v := strings.TrimRight(strings.TrimSpace(*p.PublicBaseURL), "/")
		p.PublicBaseURL = &v
	}
}

func (p SettingsPatch) Validate() error {
	v := validator{}
	if p.ServiceName != nil {
		v.check(*p.ServiceName != "", "serviceName", "is required")
		v.check(len([]rune(*p.ServiceName)) <= 60, "serviceName", "must be at most 60 characters")
	}
	if p.PublicBaseURL != nil && *p.PublicBaseURL != "" {
		u, err := url.Parse(*p.PublicBaseURL)
		v.check(err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.RawQuery == "" && u.Fragment == "",
			"publicBaseUrl", "must be an http(s) address such as https://admin.example.com")
	}
	return v.err()
}

// Apply returns s with the patch applied.
func (s Settings) Apply(p SettingsPatch) Settings {
	if p.ServiceName != nil {
		s.ServiceName = *p.ServiceName
	}
	if p.PublicBaseURL != nil {
		s.PublicBaseURL = *p.PublicBaseURL
	}
	return s
}

// Changes describes what differs between two settings, for the audit log.
func (s Settings) Changes(next Settings) []string {
	var out []string
	if s.ServiceName != next.ServiceName {
		out = append(out, "renamed the service to “"+next.ServiceName+"”")
	}
	if s.PublicBaseURL != next.PublicBaseURL {
		if next.PublicBaseURL == "" {
			out = append(out, "cleared the public base URL")
		} else {
			out = append(out, "set the public base URL to "+next.PublicBaseURL)
		}
	}
	return out
}
