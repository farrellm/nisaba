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
			name: "unclosed tag does not loop or panic",
			in:   "<a>ok</a><b>never closed",
			want: map[string]string{"a": "ok"},
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
