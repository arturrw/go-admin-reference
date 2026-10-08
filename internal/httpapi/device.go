package httpapi

import (
	"net/http"
	"strings"
)

// deviceLabel describes who is signing in, for the activity log and new-device
// alerts: "Chrome on Windows (203.0.113.7)". It is deliberately coarse, so a
// browser update doesn't look like a new device.
func deviceLabel(r *http.Request) string {
	ua := r.UserAgent()
	has := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(ua, s) {
				return true
			}
		}
		return false
	}
	browser := "Unknown browser"
	switch {
	case has("Edg/", "EdgA/", "EdgiOS/"):
		browser = "Edge"
	case has("OPR/"):
		browser = "Opera"
	case has("Firefox/", "FxiOS/"):
		browser = "Firefox"
	case has("Chrome/", "CriOS/"):
		browser = "Chrome"
	case has("Safari/"):
		browser = "Safari"
	case has("curl/"):
		browser = "curl"
	case has("Go-http-client"):
		browser = "Go client"
	}
	os := ""
	switch {
	case has("Android"):
		os = "Android"
	case has("iPhone", "iPad", "iOS"):
		os = "iOS"
	case has("Windows"):
		os = "Windows"
	case has("Mac OS X", "Macintosh"):
		os = "macOS"
	case has("Linux", "X11"):
		os = "Linux"
	}
	label := browser
	if os != "" {
		label += " on " + os
	}
	return label + " (" + clientIP(r) + ")"
}
