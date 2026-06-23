package llm

import (
	"reflect"
	"testing"
)

func TestParseLabelsExtractsSuggestionLabels(t *testing.T) {
	res := `<analysis>
Scratch reasoning about the plot, setting, and tone goes here.
</analysis>
<label_selection>brainstorming candidate labels here</label_selection>
<suggestion>
  <label>Time Travel</label>
  <label>  Dystopia  </label>
  <label></label>
  <label>Redemption</label>
</suggestion>`

	got := parseLabels(res)
	// Only the <label> tags are extracted; surrounding scratch reasoning is
	// ignored, the empty label is dropped, and inner whitespace is trimmed.
	want := []string{"Time Travel", "Dystopia", "Redemption"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseLabelsNoMatchesReturnsEmptyNonNil(t *testing.T) {
	got := parseLabels("no tags here")
	if got == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}
