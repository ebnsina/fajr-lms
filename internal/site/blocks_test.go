package site

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"unknown type":      `[{"type":"iframe"}]`,
		"hero with no head": `[{"type":"hero","text":"hi"}]`,
		"empty list block":  `[{"type":"faq","items":[]}]`,
		"item with no name": `[{"type":"features","items":[{"text":"only prose"}]}]`,
		"script link":       `[{"type":"cta","heading":"Go","cta_label":"Go","cta_href":"javascript:alert(1)"}]`,
		"label with no ref": `[{"type":"cta","heading":"Go","cta_label":"Go"}]`,
		"too many courses":  `[{"type":"courses","limit":99}]`,
		"stats with none":   `[{"type":"stats","items":[]}]`,
		"notice with no id": `[{"type":"notices","items":[{"text":"12 June"}]}]`,
		"not a list":        `{"type":"hero"}`,
	}
	for name, raw := range cases {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Errorf("%s: expected a complaint, got none", name)
		}
	}
}

func TestParseKeepsGoodInput(t *testing.T) {
	raw := `[{"type":"hero","heading":"  Greenfield  ","cta_label":"Browse","cta_href":"/courses"},
	         {"type":"faq","items":[{"title":"Fees?","text":"Monthly."}]},
	         {"type":"courses","limit":6}]`
	blocks, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("want 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Heading != "Greenfield" {
		t.Errorf("heading not trimmed: %q", blocks[0].Heading)
	}
	if blocks[2].Items != nil {
		t.Error("a courses block should not keep entries")
	}
	if _, err := Encode(blocks); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestParseKeepsNumbersAndNotices(t *testing.T) {
	raw := `[{"type":"stats","items":[{"title":"1,240","text":"students"}]},
	         {"type":"notices","heading":"Notice board","items":[{"title":"Results published","text":"12 June"}]}]`
	blocks, err := Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(blocks) != 2 || blocks[0].Items[0].Title != "1,240" {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
}

func TestParseTrimsOverlongText(t *testing.T) {
	long, _ := json.Marshal([]Block{{Type: "richtext", Text: strings.Repeat("a", maxLong+50)}})
	blocks, err := Parse(long)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(blocks[0].Text) > maxLong {
		t.Errorf("text not capped: %d", len(blocks[0].Text))
	}
}

func TestParseEmptyIsAnEmptyPage(t *testing.T) {
	blocks, err := Parse(nil)
	if err != nil || len(blocks) != 0 {
		t.Fatalf("want an empty page, got %v %v", blocks, err)
	}
}
