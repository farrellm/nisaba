package handler

import (
	"reflect"
	"testing"
)

func TestParseTopLevelTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{
			name: "single tag",
			in:   "<summary>hello</summary>",
			want: map[string]string{"summary": "hello"},
		},
		{
			name: "multiple sibling tags",
			in:   "<a>one</a><b>two</b>",
			want: map[string]string{"a": "one", "b": "two"},
		},
		{
			name: "nested tags kept verbatim",
			in:   "<outer><inner>x</inner></outer>",
			want: map[string]string{"outer": "<inner>x</inner>"},
		},
		{
			name: "same-name nesting matches outermost close",
			in:   "<a>1<a>2</a>3</a>",
			want: map[string]string{"a": "1<a>2</a>3"},
		},
		{
			name: "surrounding and interleaved text ignored",
			in:   "intro <a>x</a> middle <b>y</b> end",
			want: map[string]string{"a": "x", "b": "y"},
		},
		{
			name: "self-closing yields empty value",
			in:   "<done/>",
			want: map[string]string{"done": ""},
		},
		{
			name: "empty body yields empty value",
			in:   "<note></note>",
			want: map[string]string{"note": ""},
		},
		{
			name: "unclosed last tag is auto-closed at EOF",
			in:   "<a>ok</a><b>never closed",
			want: map[string]string{"a": "ok", "b": "never closed"},
		},
		{
			name: "unclosed tag captures nested markup verbatim",
			in:   "<a>x <b>y</b> z",
			want: map[string]string{"a": "x <b>y</b> z"},
		},
		{
			name: "empty unclosed tag at EOF yields empty value",
			in:   "<a>",
			want: map[string]string{"a": ""},
		},
		{
			name: "opening-tag attributes ignored for key",
			in:   `<item id="1" class="x">body</item>`,
			want: map[string]string{"item": "body"},
		},
		{
			name: "last occurrence wins",
			in:   "<a>first</a><a>second</a>",
			want: map[string]string{"a": "second"},
		},
		{
			name: "no tags",
			in:   "just plain text",
			want: map[string]string{},
		},
		{
			name: "less-than that is not a tag is ignored",
			in:   "1 < 2 and <a>x</a>",
			want: map[string]string{"a": "x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTopLevelTags(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTopLevelTags(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

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
