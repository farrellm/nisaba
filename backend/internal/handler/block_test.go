package handler

import (
	"reflect"
	"testing"
)

func TestApplyRenames(t *testing.T) {
	tests := []struct {
		name    string
		updates map[string]string
		renames map[string]string
		want    map[string]string
	}{
		{
			name:    "source present is moved and original dropped",
			updates: map[string]string{"revised_outline": "x", "keep": "y"},
			renames: map[string]string{"revised_outline": "outline"},
			want:    map[string]string{"outline": "x", "keep": "y"},
		},
		{
			name:    "source absent is a no-op",
			updates: map[string]string{"keep": "y"},
			renames: map[string]string{"revised_outline": "outline"},
			want:    map[string]string{"keep": "y"},
		},
		{
			name:    "existing target is overwritten by source",
			updates: map[string]string{"rewritten_story": "new", "story": "old"},
			renames: map[string]string{"rewritten_story": "story"},
			want:    map[string]string{"story": "new"},
		},
		{
			name:    "nil renames is a no-op",
			updates: map[string]string{"a": "b"},
			renames: nil,
			want:    map[string]string{"a": "b"},
		},
		{
			name:    "empty renames is a no-op",
			updates: map[string]string{"a": "b"},
			renames: map[string]string{},
			want:    map[string]string{"a": "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyRenames(tt.updates, tt.renames)
			if !reflect.DeepEqual(tt.updates, tt.want) {
				t.Errorf("applyRenames() = %v, want %v", tt.updates, tt.want)
			}
		})
	}
}
