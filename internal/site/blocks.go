// Package site validates the blocks a page is built from. Every field is plain
// text: nothing here is ever rendered as markup, so a page cannot carry script.
package site

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Block is one section of a page. Fields are shared across types rather than
// modelled per type, so a page is one flat list to store, edit and render.
type Block struct {
	Type     string `json:"type"`
	Heading  string `json:"heading,omitempty"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	CTALabel string `json:"cta_label,omitempty"`
	CTAHref  string `json:"cta_href,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Items    []Item `json:"items,omitempty"`
}

// Item is one entry inside a block that lists things.
type Item struct {
	Title string `json:"title"`
	Text  string `json:"text,omitempty"`
}

const (
	MaxBlocks = 40
	MaxItems  = 20
	maxShort  = 200
	maxLong   = 4000
)

// needsItems is the set of types built from a list; the rest are prose.
var kinds = map[string]struct{ needsHeading, needsItems bool }{
	"hero":     {needsHeading: true},
	"richtext": {},
	"features": {needsItems: true},
	"faq":      {needsItems: true},
	"stats":    {needsItems: true},
	"notices":  {needsItems: true},
	"courses":  {},
	"cta":      {needsHeading: true},
}

// Kinds lists the block types a page may use, for the editor to offer.
func Kinds() []string {
	return []string{"hero", "richtext", "features", "faq", "stats", "notices", "courses", "cta"}
}

// Parse checks raw JSON from a client and returns the blocks it describes.
// The error is worded for the person editing the page.
func Parse(raw []byte) ([]Block, error) {
	if len(raw) == 0 {
		return []Block{}, nil
	}
	var blocks []Block
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("this page's sections are not readable")
	}
	if len(blocks) > MaxBlocks {
		return nil, fmt.Errorf("a page holds at most %d sections", MaxBlocks)
	}
	for i := range blocks {
		if err := clean(&blocks[i]); err != nil {
			return nil, fmt.Errorf("section %d: %w", i+1, err)
		}
	}
	if blocks == nil {
		blocks = []Block{}
	}
	return blocks, nil
}

func clean(b *Block) error {
	rules, ok := kinds[b.Type]
	if !ok {
		return fmt.Errorf("%q is not a kind of section", b.Type)
	}

	b.Heading = trim(b.Heading, maxShort)
	b.Text = trim(b.Text, maxLong)
	b.CTALabel = trim(b.CTALabel, 60)

	var err error
	if b.ImageURL, err = link(b.ImageURL); err != nil {
		return fmt.Errorf("the image address %w", err)
	}
	if b.CTAHref, err = link(b.CTAHref); err != nil {
		return fmt.Errorf("the button address %w", err)
	}
	if b.CTALabel != "" && b.CTAHref == "" {
		return fmt.Errorf("the button needs somewhere to go")
	}
	if b.Limit < 0 || b.Limit > 24 {
		return fmt.Errorf("show between 0 and 24 courses")
	}

	if rules.needsHeading && b.Heading == "" {
		return fmt.Errorf("needs a heading")
	}
	if !rules.needsItems {
		b.Items = nil
		return nil
	}
	if len(b.Items) == 0 {
		return fmt.Errorf("needs at least one entry")
	}
	if len(b.Items) > MaxItems {
		return fmt.Errorf("holds at most %d entries", MaxItems)
	}
	for i := range b.Items {
		b.Items[i].Title = trim(b.Items[i].Title, maxShort)
		b.Items[i].Text = trim(b.Items[i].Text, maxLong)
		if b.Items[i].Title == "" {
			return fmt.Errorf("entry %d needs a title", i+1)
		}
	}
	return nil
}

func trim(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return strings.TrimSpace(s[:max])
	}
	return s
}

// link keeps addresses to plain http(s) or a path on this site, so no scheme
// can smuggle script into a rendered page.
func link(raw string) (string, error) {
	raw = trim(raw, 500)
	if raw == "" {
		return "", nil
	}
	if strings.HasPrefix(raw, "/") {
		return raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("must be a web address or a path starting with /")
	}
	return u.String(), nil
}

// Encode returns the storable JSON for a validated list.
func Encode(blocks []Block) ([]byte, error) { return json.Marshal(blocks) }
