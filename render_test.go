package main

import (
	"strings"
	"testing"
)

func TestCleanDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "html entities",
			input:    "A <great> project > other",
			expected: "A &lt;great&gt; project &gt; other",
		},
		{
			name:     "newlines and spaces",
			input:    "Line 1\nLine 2\n  Line 3  ",
			expected: "Line 1Line 2  Line 3",
		},
		{
			name:     "truncate 200 runes",
			input:    strings.Repeat("字", 250),
			expected: strings.Repeat("字", 200),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanDescription(tc.input)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestMakeAnchor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Go", "go"},
		{"Jupyter Notebook", "jupyter-notebook"},
		{"C++", "c++"},
		{"C#", "c#"},
		{"Vim Script", "vim-script"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := makeAnchor(tc.input)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestGenerateMarkdown(t *testing.T) {
	repos := []Repository{
		{
			NameWithOwner:  "golang/go",
			Description:    "The Go programming language <3",
			Language:       "Go",
			URL:            "https://github.com/golang/go",
			StargazerCount: 120000,
			IsPrivate:      false,
			Topics:         []string{"compiler", "go"},
		},
		{
			NameWithOwner:  "dotnet/csharplang",
			Description:    "C# language discussions",
			Language:       "C#",
			URL:            "https://github.com/dotnet/csharplang",
			StargazerCount: 10000,
			IsPrivate:      false,
			Topics:         []string{"csharp"},
		},
		{
			NameWithOwner:  "secret/private-repo",
			Description:    "Private project",
			Language:       "Python",
			URL:            "https://github.com/secret/private-repo",
			StargazerCount: 5,
			IsPrivate:      true,
			Topics:         []string{"secret"},
		},
	}

	t.Run("by language unsorted without private", func(t *testing.T) {
		md := GenerateMarkdown("myuser", repos, false, false, false)

		if strings.Contains(md, "secret/private-repo") {
			t.Errorf("private repo should not be included")
		}

		// TOC order: Go first, C# second (order of appearance)
		goIdx := strings.Index(md, "- [Go](#go)")
		csIdx := strings.Index(md, "- [C#](#c#)")
		if goIdx == -1 || csIdx == -1 || goIdx > csIdx {
			t.Errorf("TOC order mismatch: Go should appear before C#")
		}

		// Header escaping for C# -> C# #
		if !strings.Contains(md, "## C# # \n") {
			t.Errorf("expected '## C# # \\n' header")
		}

		// HTML escaping
		if !strings.Contains(md, "The Go programming language &lt;3") {
			t.Errorf("expected html escaped description")
		}

		// License
		if !strings.Contains(md, "[myuser](https://github.com/myuser)") {
			t.Errorf("expected username in license")
		}
	})

	t.Run("sorted alphabetically", func(t *testing.T) {
		md := GenerateMarkdown("myuser", repos, true, false, false)
		// C# sorted before Go alphabetically
		goIdx := strings.Index(md, "- [Go](#go)")
		csIdx := strings.Index(md, "- [C#](#c#)")
		if goIdx == -1 || csIdx == -1 || csIdx > goIdx {
			t.Errorf("Sorted TOC order mismatch: C# should appear before Go")
		}
	})

	t.Run("with private repos", func(t *testing.T) {
		md := GenerateMarkdown("myuser", repos, false, false, true)
		if !strings.Contains(md, "secret/private-repo") {
			t.Errorf("private repo should be included")
		}
	})

	t.Run("by topic", func(t *testing.T) {
		md := GenerateMarkdown("myuser", repos, false, true, false)
		if !strings.Contains(md, "- [compiler](#compiler)") {
			t.Errorf("expected compiler topic")
		}
		if !strings.Contains(md, "- [csharp](#csharp)") {
			t.Errorf("expected csharp topic")
		}
	})
}
