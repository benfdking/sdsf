package gitrepo

import "testing"

func TestNormalizeRemote(t *testing.T) {
	tests := map[string]string{
		"git@github.com:benfdking/sdsf.git":         "https://github.com/benfdking/sdsf",
		"git@github.com:benfdking/sdsf":             "https://github.com/benfdking/sdsf",
		"ssh://git@github.com/benfdking/sdsf.git":   "https://github.com/benfdking/sdsf",
		"git://github.com/benfdking/sdsf.git":       "https://github.com/benfdking/sdsf",
		"https://github.com/benfdking/sdsf.git":     "https://github.com/benfdking/sdsf",
		"https://github.com/benfdking/sdsf":         "https://github.com/benfdking/sdsf",
		"  https://github.com/benfdking/sdsf.git  ": "https://github.com/benfdking/sdsf",
		"https://github.com/benfdking/sdsf/":        "https://github.com/benfdking/sdsf",
		"git@gitlab.com:group/sub/project.git":      "https://gitlab.com/group/sub/project",
		"":                                          "",
	}

	for remote, want := range tests {
		if got := NormalizeRemote(remote); got != want {
			t.Errorf("NormalizeRemote(%q) = %q, want %q", remote, got, want)
		}
	}
}
