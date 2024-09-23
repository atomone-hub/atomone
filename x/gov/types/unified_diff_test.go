package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateUnifiedDiff(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		dst      string
		expected string
	}{
		{
			name:     "No changes",
			src:      "Line one\nLine two\nLine three",
			dst:      "Line one\nLine two\nLine three",
			expected: ``,
		},
		{
			name: "Line added",
			src:  "Line one\nLine two",
			dst:  "Line one\nLine two\nLine three",
			expected: `@@ -1,2 +1,3 @@
 Line one
 Line two
+Line three
`,
		},
		{
			name: "Line deleted",
			src:  "Line one\nLine two\nLine three",
			dst:  "Line one\nLine three",
			expected: `@@ -1,3 +1,2 @@
 Line one
-Line two
 Line three
`,
		},
		{
			name: "Line modified",
			src:  "Line one\nLine two\nLine three",
			dst:  "Line one\nLine two modified\nLine three",
			expected: `@@ -1,3 +1,3 @@
 Line one
-Line two
+Line two modified
 Line three
`,
		},
		{
			name: "Multiple changes",
			src:  "Line one\nLine two\nLine three",
			dst:  "Line zero\nLine one\nLine three\nLine four",
			expected: `@@ -1,3 +1,4 @@
+Line zero
 Line one
-Line two
 Line three
+Line four
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GenerateUnifiedDiff(tt.src, tt.dst)
			require.NoError(t, err)

			diffContent := strings.TrimPrefix(diff, "--- src\n+++ dst\n")
			expectedContent := strings.TrimPrefix(tt.expected, "--- src\n+++ dst\n")

			require.Equal(t, expectedContent, diffContent)
		})
	}
}

func TestApplyUnifiedDiff(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		diffStr  string
		expected string
		wantErr  bool
	}{
		{
			name: "Apply addition",
			src:  "Line one\nLine two",
			diffStr: `@@ -1,2 +1,3 @@
 Line one
 Line two
+Line three
`,
			expected: "Line one\nLine two\nLine three",
			wantErr:  false,
		},
		{
			name: "Apply deletion",
			src:  "Line one\nLine two\nLine three",
			diffStr: `@@ -1,3 +1,2 @@
 Line one
-Line two
 Line three
`,
			expected: "Line one\nLine three",
			wantErr:  false,
		},
		{
			name: "Apply modification",
			src:  "Line one\nLine two\nLine three",
			diffStr: `@@ -2 +2 @@
-Line two
+Line two modified
`,
			expected: "Line one\nLine two modified\nLine three",
			wantErr:  false,
		},
		{
			name: "Apply multiple changes",
			src:  "Line one\nLine two\nLine three",
			diffStr: `@@ -1,3 +1,4 @@
+Line zero
 Line one
-Line two
 Line three
+Line four
`,
			expected: "Line zero\nLine one\nLine three\nLine four",
			wantErr:  false,
		},
		{
			name: "Malformed diff",
			src:  "Line one\nLine two",
			diffStr: `@@ -1,2 +1,3 @@
 Line one
+Line three
`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "Context line mismatch",
			src:  "Line one\nLine two",
			diffStr: `@@ -1,2 +1,2 @@
 Line zero
 Line two
`,
			expected: "",
			wantErr:  true,
		},
		{
			name:    "Empty diff",
			src:     "Line one\nLine two",
			diffStr: ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyUnifiedDiff(tt.src, tt.diffStr)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUnifiedDiffIntegration(t *testing.T) {
	src := "Line one\nLine two\nLine three"
	dst := "Line zero\nLine one\nLine three\nLine four"

	diffStr, err := GenerateUnifiedDiff(src, dst)
	require.NoError(t, err)

	result, err := ApplyUnifiedDiff(src, diffStr)
	require.NoError(t, err)
	require.Equal(t, dst, result)
}

func TestParseUnifiedDiff(t *testing.T) {
	diffStr := `@@ -1,3 +1,4 @@
+Line zero
 Line one
-Line two
 Line three
+Line four
`

	expectedHunks := []Hunk{
		{
			SrcLine: 0,
			SrcSpan: 3,
			DstLine: 0,
			DstSpan: 4,
			Lines: []string{
				"+Line zero",
				" Line one",
				"-Line two",
				" Line three",
				"+Line four",
			},
		},
	}

	t.Run("Hunk with spans", func(t *testing.T) {
		hunks, err := ParseUnifiedDiff(diffStr)
		require.NoError(t, err)
		require.Equal(t, expectedHunks, hunks)
	})

	diffWithoutSpans := `@@ -2 +2 @@
-Line two
+Line two modified
`

	expectedHunksWithoutSpans := []Hunk{
		{
			SrcLine: 1,
			SrcSpan: 1,
			DstLine: 1,
			DstSpan: 1,
			Lines: []string{
				"-Line two",
				"+Line two modified",
			},
		},
	}

	t.Run("Hunk header without spans", func(t *testing.T) {
		hunks, err := ParseUnifiedDiff(diffWithoutSpans)
		require.NoError(t, err)
		require.Equal(t, expectedHunksWithoutSpans, hunks)
	})

	invalidDiffs := []struct {
		name     string
		diffStr  string
		wantErr  bool
		errorMsg string
	}{
		{
			name: "Invalid hunk header format",
			diffStr: `@@ invalid header @@
 Line one
`,
			wantErr:  true,
			errorMsg: "invalid hunk header",
		},
		{
			name: "Negative span in hunk header",
			diffStr: `@@ -1,-2 +1,2 @@
 Line one
`,
			wantErr:  true,
			errorMsg: "negative span",
		},
		{
			name: "Invalid line prefix",
			diffStr: `@@ -1,1 +1,1 @@
?Line one
`,
			wantErr:  true,
			errorMsg: "invalid line prefix",
		},
		{
			name: "Source line count mismatch",
			diffStr: `@@ -1,2 +1,2 @@
 Line one
+Line two
`,
			wantErr:  true,
			errorMsg: "source line count",
		},
		{
			name: "Destination line count mismatch",
			diffStr: `@@ -1,2 +1,2 @@
-Line one
 Line two
`,
			wantErr:  true,
			errorMsg: "destination line count",
		},
		{
			name:     "No hunks",
			diffStr:  ``,
			wantErr:  true,
			errorMsg: "no valid hunks",
		},
		{
			name: "Unexpected content outside hunks",
			diffStr: `Unexpected line
@@ -1,1 +1,1 @@
 Line one
`,
			wantErr:  true,
			errorMsg: "unexpected content outside of hunks",
		},
	}

	for _, tt := range invalidDiffs {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseUnifiedDiff(tt.diffStr)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestParseHunkHeader tests the parseHunkHeader function.
func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected *Hunk
		wantErr  bool
	}{
		{
			name:   "Header with spans",
			header: "@@ -1,3 +1,4 @@",
			expected: &Hunk{
				SrcLine: 0,
				SrcSpan: 3,
				DstLine: 0,
				DstSpan: 4,
			},
			wantErr: false,
		},
		{
			name:   "Header without spans",
			header: "@@ -2 +2 @@",
			expected: &Hunk{
				SrcLine: 1,
				SrcSpan: 1,
				DstLine: 1,
				DstSpan: 1,
			},
			wantErr: false,
		},
		{
			name:   "Header with zero spans",
			header: "@@ -0,0 +1,2 @@",
			expected: &Hunk{
				SrcLine: -1,
				SrcSpan: 0,
				DstLine: 0,
				DstSpan: 2,
			},
			wantErr: false,
		},
		{
			name:     "Invalid header format",
			header:   "@@ invalid header @@",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Negative span",
			header:   "@@ -1,-2 +1,2 @@",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunk, err := parseHunkHeader(tt.header)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, hunk)
			}
		})
	}
}
