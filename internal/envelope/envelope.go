package envelope

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrInvalidFormat = errors.New("invalid envelope format")
	ErrMissingDSN    = errors.New("missing DSN in envelope header")
	ErrInvalidDSN    = errors.New("invalid DSN format")
)

type Header struct {
	DSN string `json:"dsn"`
}

// ParseProjectID extracts the Sentry project ID from a raw envelope body.
func ParseProjectID(body []byte) (string, error) {
	newline := bytes.IndexByte(body, '\n')
	if newline == -1 {
		return "", ErrInvalidFormat
	}

	var h Header
	if err := json.Unmarshal(body[:newline], &h); err != nil {
		return "", ErrInvalidFormat
	}

	if h.DSN == "" {
		return "", ErrMissingDSN
	}

	return extractProjectID(h.DSN)
}

// extractProjectID parses project ID from Sentry DSN.
func extractProjectID(dsn string) (string, error) {
	idx := strings.LastIndex(dsn, "/")
	if idx == -1 || idx == len(dsn)-1 {
		return "", ErrInvalidDSN
	}

	return dsn[idx+1:], nil
}
