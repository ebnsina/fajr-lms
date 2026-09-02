package notify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPChannel talks to any gateway that takes a request and returns a body.
// Local SMS providers differ only in field names and where they put the
// credentials, so one configurable channel covers most of them.
type HTTPChannel struct {
	ChannelName string
	Method      string
	// URL and Body may contain {to}, {message} and {sender}, filled per send.
	URL      string
	Body     string
	Headers  map[string]string
	Sender   string
	Encoding string // "form", "json" or "" for a URL-only gateway
	// SuccessContains, when set, must appear in a 2xx body for a send to count.
	SuccessContains string
	Client          *http.Client
}

func (c HTTPChannel) Name() string {
	if c.ChannelName == "" {
		return "http"
	}
	return c.ChannelName
}

func (c HTTPChannel) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c HTTPChannel) Send(ctx context.Context, m Message) error {
	if strings.TrimSpace(m.To) == "" {
		return fmt.Errorf("notify: message has no destination")
	}
	if strings.TrimSpace(c.URL) == "" {
		return fmt.Errorf("notify: channel %q has no url configured", c.Name())
	}

	method := strings.ToUpper(c.Method)
	if method == "" {
		method = http.MethodPost
	}

	endpoint := c.fill(c.URL, m, url.QueryEscape)
	var body io.Reader
	contentType := ""

	switch strings.ToLower(c.Encoding) {
	case "json":
		body, contentType = strings.NewReader(c.fill(c.Body, m, jsonEscape)), "application/json"
	case "form":
		form := url.Values{}
		for _, pair := range strings.Split(c.fill(c.Body, m, url.QueryEscape), "&") {
			key, value, found := strings.Cut(pair, "=")
			if !found {
				continue
			}
			decoded, err := url.QueryUnescape(value)
			if err != nil {
				decoded = value
			}
			form.Set(key, decoded)
		}
		body, contentType = strings.NewReader(form.Encode()), "application/x-www-form-urlencoded"
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("notify: build %s request: %w", c.Name(), err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client().Do(req)
	if err != nil {
		return fmt.Errorf("notify: reach %s: %w", c.Name(), err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("notify: read %s response: %w", c.Name(), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("notify: %s returned %d: %s", c.Name(), resp.StatusCode, truncate(raw))
	}
	// Several gateways answer 200 with a failure in the body.
	if c.SuccessContains != "" && !bytes.Contains(raw, []byte(c.SuccessContains)) {
		return fmt.Errorf("notify: %s rejected the message: %s", c.Name(), truncate(raw))
	}
	return nil
}

func (c HTTPChannel) fill(template string, m Message, escape func(string) string) string {
	return strings.NewReplacer(
		"{to}", escape(m.To),
		"{message}", escape(m.Body),
		"{sender}", escape(c.Sender),
	).Replace(template)
}

// jsonEscape quotes a value for interpolation inside a JSON template.
func jsonEscape(v string) string {
	var b strings.Builder
	for _, r := range v {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func truncate(raw []byte) string {
	if len(raw) > 200 {
		return string(raw[:200])
	}
	return string(raw)
}
