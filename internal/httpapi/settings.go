package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
)

// Log levels the UI can pick. Changes apply to the running process only;
// LOG_LEVEL decides the level again on the next start.
var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func levelName(l slog.Level) string { return strings.ToLower(l.String()) }

func (s *server) getLogLevel(w http.ResponseWriter, _ *http.Request) {
	if s.level == nil {
		writeError(w, http.StatusNotImplemented, "log level is fixed in this build")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"level": levelName(s.level.Level())})
}

func (s *server) setLogLevel(w http.ResponseWriter, r *http.Request) {
	if s.level == nil {
		writeError(w, http.StatusNotImplemented, "log level is fixed in this build")
		return
	}
	var in struct {
		Level string `json:"level"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	lvl, ok := logLevels[strings.ToLower(strings.TrimSpace(in.Level))]
	if !ok {
		writeJSON(w, http.StatusUnprocessableEntity, errorBody{Error: "validation failed", Fields: map[string]string{"level": "must be debug, info, warn or error"}})
		return
	}
	prev := s.level.Level()
	s.level.Set(lvl)
	me, _ := CurrentMember(r.Context())
	// Logged at warn so the change itself is visible at every level but error.
	s.log.WarnContext(r.Context(), "log level changed", "from", levelName(prev), "to", levelName(lvl), "actor", me.Email)
	writeJSON(w, http.StatusOK, map[string]string{"level": levelName(lvl)})
}
