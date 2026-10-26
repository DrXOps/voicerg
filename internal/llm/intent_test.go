package llm

import (
	"strings"
	"testing"
)

func TestStripMarkdown(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"```json\n{\"a\":1}\n```", "{\"a\":1}"},
		{"{\"a\":1}", "{\"a\":1}"},
		{"", ""},
		{"```json\n{}\n```", "{}"},
	}
	for _, c := range cases {
		if got := stripMarkdown(c.in); got != c.want {
			t.Errorf("stripMarkdown(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFallbackHeuristic(t *testing.T) {
	ic := NewIntentClassifier(nil, "model", "prompt", 0.6)
	ic.prompt = "system"

	it := ic.fallbackHeuristic("поехали писать коленку")
	if it.Intent != "start_recording" {
		t.Errorf("fallback start_recording: got %q", it.Intent)
	}

	it = ic.fallbackHeuristic("закончил")
	if it.Intent != "stop_recording" {
		t.Errorf("fallback stop_recording: got %q", it.Intent)
	}

	it = ic.fallbackHeuristic("что-то непонятное")
	if it.Intent != "unknown" {
		t.Errorf("fallback unknown: got %q", it.Intent)
	}
}

func TestBuildQuery(t *testing.T) {
	ic := NewIntentClassifier(nil, "model", "prompt", 0.6)
	q := ic.buildQuery("опа поехали", []string{"Idle", "Idle", "Done"})
	if !strings.Contains(q, "опа поехали") {
		t.Error("query missing text")
	}
	if !strings.Contains(q, "Idle") {
		t.Error("query missing context")
	}
}

func TestParseResponse(t *testing.T) {
	ic := NewIntentClassifier(nil, "model", "prompt", 0.6)
	it, err := ic.parseResponse(`{"intent":"insert_block","payload":{"block":"ЗАКЛЮЧЕНИЕ"},"confidence":0.95}`)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if it.Intent != "insert_block" {
		t.Errorf("intent = %q", it.Intent)
	}
	if it.Confidence != 0.95 {
		t.Errorf("confidence = %v", it.Confidence)
	}
}
