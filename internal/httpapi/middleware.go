package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/arturrw/go-admin-reference/internal/reqlog"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	reqInfoKey
)

func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// reqInfo is filled in by inner handlers (e.g. authorize) and read by the
// access log after the request completes.
type reqInfo struct {
	actor, role string
}

func reqInfoFrom(ctx context.Context) *reqInfo {
	info, _ := ctx.Value(reqInfoKey).(*reqInfo)
	return info
}

// capture keeps the first limit bytes written to it.
type capture struct {
	buf       []byte
	limit     int
	truncated bool
}

func (c *capture) Write(p []byte) (int, error) {
	keep := p
	if room := max(c.limit-len(c.buf), 0); len(p) > room {
		keep, c.truncated = p[:room], true
	}
	c.buf = append(c.buf, keep...)
	return len(p), nil
}

// statusRecorder captures status, size and (for errors) the response body.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	bytes   int
	errBody *capture
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
		if code >= 400 {
			r.errBody = &capture{limit: 2048}
		}
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	if r.errBody != nil {
		r.errBody.Write(b[:n])
	}
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 64 {
			var b [6]byte
			_, _ = rand.Read(b[:])
			id = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

func (s *server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.ErrorContext(r.Context(), "panic", "value", v, "stack", string(debug.Stack()), "request_id", RequestID(r.Context()))
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Polling endpoints are excluded from the request log so the UI does not
// fill it with its own heartbeats.
var unloggedPaths = map[string]bool{"/api/v1/requests": true, "/api/v1/runtime": true, "/api/v1/live": true, "/api/v1/auth/me": true}

const maxLoggedBody = 4096

func (s *server) withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		info := &reqInfo{}
		r = r.WithContext(context.WithValue(r.Context(), reqInfoKey, info))

		isAPI := strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz"
		record := isAPI && !unloggedPaths[r.URL.Path] && !strings.HasPrefix(r.URL.Path, "/api/v1/requests/")

		var body *capture
		bodyNote := ""
		if record && r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
				bodyNote = fmt.Sprintf("[multipart form, %d bytes]", r.ContentLength)
			} else {
				body = &capture{limit: maxLoggedBody}
				r.Body = struct {
					io.Reader
					io.Closer
				}{io.TeeReader(r.Body, body), r.Body}
			}
		}

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		dur := time.Since(start)

		level := slog.LevelDebug
		if record {
			level = slog.LevelInfo
			e := reqlog.Entry{
				Summary: reqlog.Summary{
					ID: RequestID(r.Context()), Time: start, Method: r.Method, Path: r.URL.RequestURI(),
					Status: rec.status, DurationMs: float64(dur.Microseconds()) / 1000, Bytes: rec.bytes,
					IP: clientIP(r), Actor: info.actor,
				},
				Query: r.URL.RawQuery, Proto: r.Proto, UserAgent: r.UserAgent(), ActorRole: info.role,
				Route:       r.Pattern,
				ReqHeaders:  sanitizeHeaders(r.Header),
				RespHeaders: sanitizeHeaders(rec.Header()),
				ReqBytes:    r.ContentLength,
				ReqBody:     bodyNote,
			}
			if e.Route == "" {
				e.Route = reqlog.RouteOf(r.Method, r.URL.Path)
			}
			if body != nil {
				e.ReqBody, e.BodyTruncate = redactJSON(body.buf), body.truncated
			}
			if rec.errBody != nil {
				e.RespBody = string(rec.errBody.buf)
			}
			s.requests.Add(e)
		}
		switch {
		case rec.status >= 500:
			level = slog.LevelError
		case rec.status >= 400 && record:
			level = slog.LevelWarn
		}
		s.log.Log(r.Context(), level, "http",
			"method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", dur,
			"bytes", rec.bytes, "actor", info.actor, "request_id", RequestID(r.Context()))
	})
}

var redactedHeaders = map[string]bool{"Cookie": true, "Set-Cookie": true, "Authorization": true, "X-Api-Key": true}

func sanitizeHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if redactedHeaders[http.CanonicalHeaderKey(k)] {
			out[k] = "[redacted]"
		} else {
			out[k] = strings.Join(v, ", ")
		}
	}
	return out
}

// redactJSON masks any field whose name contains "password" or "token".
func redactJSON(b []byte) string {
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		return string(b)
	}
	for k := range v {
		if lk := strings.ToLower(k); strings.Contains(lk, "password") || strings.Contains(lk, "token") {
			v[k] = "[redacted]"
		}
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(b)
	}
	return string(out)
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
