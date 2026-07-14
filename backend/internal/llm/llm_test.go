package llm

import (
	"errors"
	"strings"
	"testing"
)

func TestToolBlock(t *testing.T) {
	got := toolBlock("generate_name", `{"gender":"female"}`, "Ada")
	want := "<generate_name>\narguments: {\"gender\":\"female\"}\nresult: Ada\n</generate_name>\n"
	if got != want {
		t.Errorf("toolBlock = %q, want %q", got, want)
	}
}

func TestToolResultText(t *testing.T) {
	if got := toolResultText("Ada", nil); got != "Ada" {
		t.Errorf("success = %q, want %q", got, "Ada")
	}
	if got := toolResultText("partial", errors.New("boom")); got != "error: boom" {
		t.Errorf("error = %q, want %q", got, "error: boom")
	}
	long := strings.Repeat("x", 600)
	got := toolResultText("", errors.New(long))
	want := "error: " + strings.Repeat("x", 500) + "..."
	if got != want {
		t.Errorf("long error not truncated to 500 runes: len = %d", len(got))
	}
}
