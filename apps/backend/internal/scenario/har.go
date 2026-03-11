package scenario

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// HAR structures (subset of HAR 1.2 used by Chrome DevTools).
// Spec: root object MUST be named "log" (see W3C HAR spec).
type harRoot struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string      `json:"version"`
	Creator harCreator  `json:"creator"`
	Entries []harEntry  `json:"entries"`
}

type harCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harEntry struct {
	StartedDateTime string   `json:"startedDateTime"`
	Request         harReq   `json:"request"`
}

type harReq struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	Headers     []harHeader `json:"headers"`
	PostData    *harPostData `json:"postData,omitempty"`
}

type harHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harPostData struct {
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// harEvent is one line of our JSONL output (matches docs/events.md)
type harEventLine struct {
	Ts     string            `json:"ts"`
	Type   string            `json:"type"`
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body   string            `json:"body,omitempty"`
	Weight int               `json:"weight,omitempty"`
}

// HARToJSONL converts HAR file content (from Chrome DevTools export) to scenario JSONL.
// Preserves request order and startedDateTime as ts for timestamp-based replay.
// Request URL is converted to path (path + query) so replay can target a different base URL.
func HARToJSONL(harBytes []byte) ([]byte, error) {
	var root harRoot
	if err := json.Unmarshal(harBytes, &root); err != nil {
		return nil, fmt.Errorf("invalid HAR JSON: %w", err)
	}
	har := &root.Log

	if len(har.Entries) == 0 {
		return nil, fmt.Errorf("HAR has no entries")
	}

	var lines []string
	for i, e := range har.Entries {
		path, err := urlToPath(e.Request.URL)
		if err != nil {
			return nil, fmt.Errorf("entry %d: invalid URL %q: %w", i+1, e.Request.URL, err)
		}

		// Normalize timestamp to RFC3339 (Chrome exports ISO8601 which is compatible)
		ts := strings.TrimSpace(e.StartedDateTime)
		if ts == "" {
			ts = "1970-01-01T00:00:00Z"
		}

		headers := make(map[string]string)
		for _, h := range e.Request.Headers {
			name := strings.TrimSpace(h.Name)
			if name != "" {
				headers[name] = h.Value
			}
		}

		body := ""
		if e.Request.PostData != nil {
			body = e.Request.PostData.Text
		}

		ev := harEventLine{
			Ts:      ts,
			Type:    "http",
			Method:  strings.TrimSpace(e.Request.Method),
			Path:    path,
			Headers: headers,
			Body:    body,
			Weight:  1,
		}
		if ev.Method == "" {
			ev.Method = "GET"
		}

		line, err := json.Marshal(ev)
		if err != nil {
			return nil, fmt.Errorf("entry %d: marshal: %w", i+1, err)
		}
		lines = append(lines, string(line))
	}

	return []byte(strings.Join(lines, "\n")), nil
}

// urlToPath returns path + query from full URL for use with targetBaseUrl during replay.
func urlToPath(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	p := u.Path
	if p == "" {
		p = "/"
	}
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	return p, nil
}
