package cli

import (
	"strings"
	"testing"
)

func FuzzFilterFastExportStream(f *testing.F) {
	f.Add("blob\nmark :1\ndata 2\nhi\n")
	f.Add("commit refs/heads/main\nmark :1\nauthor A <a@e.c> 1 +0000\ncommitter A <a@e.c> 1 +0000\ndata 0\nM 100644 :1 a.txt\n")
	f.Add("tag v1\nfrom :1\ntagger T <t@e.c> 1 +0000\ndata 3\nmsg\n")
	f.Add("reset refs/heads/x\nfrom :1\n")
	f.Add("data 9999999999999999999999\n")
	f.Add("blob\ndata -1\n")
	f.Add("blob\ndata 999999999\n")
	f.Add("")
	f.Fuzz(func(t *testing.T, in string) {
		f := &resolvedFilters{
			paths:   []string{"keep"},
			renames: []pathRename{{old: "a", new: "b"}},
		}
		rule, err := compileReplaceRule(`x==>y`)
		if err == nil {
			f.replaces = []replaceRule{rule}
		}
		var out strings.Builder
		_ = filterFastExportStream(strings.NewReader(in), &out, f, &filterStats{}, false)
	})
}
