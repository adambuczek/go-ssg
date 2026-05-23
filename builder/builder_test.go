package builder

import (
	"slices"
	"testing"

	"github.com/adambuczek/go-ssg/config"
)

var testCfg = config.Config{
	Src:  "src",
	Dist: "dist",
}

func TestOutputPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "nested file", input: "src/post/my-post.md", expected: "dist/post/my-post/index.html"},
		{name: "index file", input: "src/index.md", expected: "dist/index.html"},
		{name: "nested index file", input: "src/post/index.md", expected: "dist/post/index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outputPath(tt.input, testCfg)
			if got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestSplitMeta(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		wantMeta    []byte
		wantContent []byte
		wantErr     bool
	}{
		{
			name:        "valid frontmatter",
			input:       []byte("---\ntitle: title\n---\ncontent"),
			wantMeta:    []byte("title: title\n"),
			wantContent: []byte("content"),
			wantErr:     false,
		},
		{
			name:        "no frontmatter",
			input:       []byte("content"),
			wantMeta:    nil,
			wantContent: []byte("content"),
			wantErr:     true,
		},
		{
			name:        "frontmatter not starting correctly",
			input:       []byte("\ntitle: title\n---\ncontent"),
			wantMeta:    nil,
			wantContent: []byte("\ntitle: title\n---\ncontent"),
			wantErr:     true,
		},
		{
			name:        "frontmatter not closed correctly: no close line",
			input:       []byte("---\ntitle: title\n\ncontent"),
			wantMeta:    nil,
			wantContent: []byte("---\ntitle: title\n\ncontent"),
			wantErr:     true,
		},
		{
			name:        "frontmatter not closed correctly: close line with content",
			input:       []byte("---\ntitle: title\n---content"),
			wantMeta:    nil,
			wantContent: []byte("---\ntitle: title\n---content"),
			wantErr:     true,
		},
		{
			name:        "frontmatter not closed correctly: close line not in new line",
			input:       []byte("---\ntitle: title---\ncontent"),
			wantMeta:    []byte("title: title"),
			wantContent: []byte("content"),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMeta, gotContent, err := splitMeta(tt.input)
			if slices.Compare(gotMeta, tt.wantMeta) != 0 {
				t.Errorf("got %s, want %s", gotMeta, tt.wantMeta)
			}
			if slices.Compare(gotContent, tt.wantContent) != 0 {
				t.Errorf("got %s, want %s", gotContent, tt.wantContent)
			}
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("got %s, want %v", err, tt.wantErr)
			}
		})
	}
}
