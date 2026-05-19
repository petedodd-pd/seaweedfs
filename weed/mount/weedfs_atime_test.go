package mount

import "testing"

func TestSplitPath(t *testing.T) {
	cases := []struct {
		in       string
		wantDir  string
		wantName string
	}{
		{"/bucket/object", "/bucket", "object"},
		{"/a/b/c", "/a/b", "c"},
		{"/single", "/", "single"},
		{"nameonly", "", "nameonly"},
		{"", "", ""},
	}
	for _, tc := range cases {
		gotDir, gotName := splitPath(tc.in)
		if gotDir != tc.wantDir || gotName != tc.wantName {
			t.Errorf("splitPath(%q): got (%q, %q), want (%q, %q)",
				tc.in, gotDir, gotName, tc.wantDir, tc.wantName)
		}
	}
}
