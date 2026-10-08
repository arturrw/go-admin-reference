package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestDeviceLabel(t *testing.T) {
	for ua, want := range map[string]string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36":               "Chrome on Windows (192.0.2.1)",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36 Edg/126.0":     "Edge on Windows (192.0.2.1)",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:127.0) Gecko/20100101 Firefox/127.0":                   "Firefox on macOS (192.0.2.1)",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 Version/17.5 Safari/604.1": "Safari on iOS (192.0.2.1)",
		"Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/126.0 Mobile Safari/537.36":                  "Chrome on Android (192.0.2.1)",
		"curl/8.5.0": "curl (192.0.2.1)",
		"":           "Unknown browser (192.0.2.1)",
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "192.0.2.1:5555"
		r.Header.Set("User-Agent", ua)
		if got := deviceLabel(r); got != want {
			t.Errorf("%q → %q, want %q", ua, got, want)
		}
	}
}
