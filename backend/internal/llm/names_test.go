package llm

import (
	"strings"
	"testing"
)

func TestPickAllowedExcludesBlocked(t *testing.T) {
	names := []string{"Iris Vance", "Marcus Black", "Jane Doe"}
	got, err := pickAllowed(names)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Jane Doe" {
		t.Fatalf("got %q, want the only allowed name %q", got, "Jane Doe")
	}
}

func TestPickAllowedFallsBackToBlocked(t *testing.T) {
	names := []string{"Iris Vance", "Marcus Black"}
	got, err := pickAllowed(names)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocked := false
	for _, sub := range blockedNames {
		if strings.Contains(got, sub) {
			blocked = true
			break
		}
	}
	if !blocked {
		t.Fatalf("expected fallback to a blocked name, got %q", got)
	}
}

func TestPickAllowedNoNames(t *testing.T) {
	if _, err := pickAllowed(nil); err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}
