package config

import (
	"path/filepath"
	"testing"
)

func TestProjectPath(t *testing.T) {
	t.Parallel()
	base := filepath.Join(string(filepath.Separator), "config")
	home := filepath.Join(string(filepath.Separator), "owner")
	absolute := filepath.Join(string(filepath.Separator), "projects")
	tests := []struct{ name, path, want string }{
		{"relative", "projects", filepath.Join(base, "projects")},
		{"parent", filepath.Join("..", "projects"), absolute},
		{"absolute", absolute, absolute},
		{"home", "~", home},
		{"home child", "~/projects", filepath.Join(home, "projects")},
		{"named home stays literal", "~someone", filepath.Join(base, "~someone")},
		{"variables stay literal", "$PROJECTS", filepath.Join(base, "$PROJECTS")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := projectPath(tt.path, base, home); got != tt.want {
				t.Fatalf("got %q; want %q", got, tt.want)
			}
		})
	}
}
