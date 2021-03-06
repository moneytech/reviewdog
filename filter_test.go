package reviewdog

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/difffilter"
)

const diffContent = `--- sample.old.txt	2016-10-13 05:09:35.820791185 +0900
+++ sample.new.txt	2016-10-13 05:15:26.839245048 +0900
@@ -1,3 +1,4 @@
 unchanged, contextual line
-deleted line
+added line
+added line
 unchanged, contextual line
--- nonewline.old.txt	2016-10-13 15:34:14.931778318 +0900
+++ nonewline.new.txt	2016-10-13 15:34:14.868444672 +0900
@@ -1,4 +1,4 @@
 " vim: nofixeol noendofline
 No newline at end of both the old and new file
-a
-a
\ No newline at end of file
+b
+b
\ No newline at end of file
`

const diffContentAddedStrip = `diff --git a/test_added.go b/test_added.go
new file mode 100644
index 0000000..264c67e
--- /dev/null
+++ b/test_added.go
@@ -0,0 +1,3 @@
+package reviewdog
+
+var TestAdded = 14
`

func TestFilterCheckByAddedLines(t *testing.T) {
	results := []*CheckResult{
		{
			Path: "sample.new.txt",
			Lnum: 1,
		},
		{
			Path: "sample.new.txt",
			Lnum: 2,
		},
		{
			Path: "nonewline.new.txt",
			Lnum: 1,
		},
		{
			Path: "nonewline.new.txt",
			Lnum: 3,
		},
	}
	want := []*FilteredCheck{
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 1,
			},
			ShouldReport: false,
			InDiffFile:   true,
			OldPath:      "sample.old.txt",
			OldLine:      1,
		},
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 2,
			},
			ShouldReport: true,
			InDiffFile:   true,
			LnumDiff:     3,
			OldPath:      "sample.old.txt",
			OldLine:      0,
		},
		{
			CheckResult: &CheckResult{
				Path: "nonewline.new.txt",
				Lnum: 1,
			},
			ShouldReport: false,
			InDiffFile:   true,
			OldPath:      "nonewline.old.txt",
			OldLine:      1,
		},
		{
			CheckResult: &CheckResult{
				Path: "nonewline.new.txt",
				Lnum: 3,
			},
			ShouldReport: true,
			InDiffFile:   true,
			LnumDiff:     5,
			OldPath:      "nonewline.old.txt",
			OldLine:      0,
		},
	}
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContent))
	got := FilterCheck(results, filediffs, 0, "", difffilter.ModeAdded)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

// All lines that are in diff are taken into account
func TestFilterCheckByDiffContext(t *testing.T) {
	results := []*CheckResult{
		{
			Path: "sample.new.txt",
			Lnum: 1,
		},
		{
			Path: "sample.new.txt",
			Lnum: 2,
		},
		{
			Path: "sample.new.txt",
			Lnum: 3,
		},
	}
	want := []*FilteredCheck{
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 1,
			},
			ShouldReport: true,
			InDiffFile:   true,
			LnumDiff:     1,
			OldPath:      "sample.old.txt",
			OldLine:      1,
		},
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 2,
			},
			ShouldReport: true,
			InDiffFile:   true,
			LnumDiff:     3,
			OldPath:      "sample.old.txt",
			OldLine:      0,
		},
		{
			CheckResult: &CheckResult{
				Path: "sample.new.txt",
				Lnum: 3,
			},
			ShouldReport: true,
			InDiffFile:   true,
			LnumDiff:     4,
			OldPath:      "sample.old.txt",
			OldLine:      0,
		},
	}
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContent))
	got := FilterCheck(results, filediffs, 0, "", difffilter.ModeDiffContext)
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func findFileDiff(filediffs []*diff.FileDiff, path string, strip int) *diff.FileDiff {
	for _, file := range filediffs {
		if difffilter.NormalizeDiffPath(file.PathNew, strip) == path {
			return file
		}
	}
	return nil
}

func TestGetOldPosition(t *testing.T) {
	const strip = 0
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContent))
	tests := []struct {
		newPath     string
		newLine     int
		wantOldPath string
		wantOldLine int
	}{
		{
			newPath:     "sample.new.txt",
			newLine:     1,
			wantOldPath: "sample.old.txt",
			wantOldLine: 1,
		},
		{
			newPath:     "sample.new.txt",
			newLine:     2,
			wantOldPath: "sample.old.txt",
			wantOldLine: 0,
		},
		{
			newPath:     "sample.new.txt",
			newLine:     3,
			wantOldPath: "sample.old.txt",
			wantOldLine: 0,
		},
		{
			newPath:     "sample.new.txt",
			newLine:     14,
			wantOldPath: "sample.old.txt",
			wantOldLine: 13,
		},
		{
			newPath:     "not_found",
			newLine:     14,
			wantOldPath: "",
			wantOldLine: 0,
		},
	}
	for _, tt := range tests {
		fdiff := findFileDiff(filediffs, tt.newPath, strip)
		gotPath, gotLine := getOldPosition(fdiff, strip, tt.newPath, tt.newLine)
		if !(gotPath == tt.wantOldPath && gotLine == tt.wantOldLine) {
			t.Errorf("getOldPosition(..., %s, %d) = (%s, %d), want (%s, %d)",
				tt.newPath, tt.newLine, gotPath, gotLine, tt.wantOldPath, tt.wantOldLine)
		}
	}
}

func TestGetOldPosition_added(t *testing.T) {
	const strip = 1
	filediffs, _ := diff.ParseMultiFile(strings.NewReader(diffContentAddedStrip))
	path := "test_added.go"
	fdiff := findFileDiff(filediffs, path, strip)
	gotPath, _ := getOldPosition(fdiff, strip, path, 1)
	if gotPath != "" {
		t.Errorf("got %q as old path for added diff file, want empty", gotPath)
	}
}
