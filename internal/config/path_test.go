package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestRootResolverDefaultAndOverride(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  string
	}{
		{name: "default", want: filepath.Join("/home/tester", ".kstankcli")},
		{name: "override", value: filepath.Join("/tmp", "custom", "..", "root"), set: true, want: filepath.Join("/tmp", "root")},
		{name: "tilde", value: "~/tools", set: true, want: filepath.Join("/home/tester", "tools")},
		{name: "empty override", value: "  ", set: true, want: filepath.Join("/home/tester", ".kstankcli")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := RootResolver{
				LookupEnv:   func(string) (string, bool) { return test.value, test.set },
				UserHomeDir: func() (string, error) { return "/home/tester", nil },
			}
			got, err := resolver.Resolve()
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestRootResolverHomeError(t *testing.T) {
	want := errors.New("no home")
	resolver := RootResolver{
		LookupEnv:   func(string) (string, bool) { return "", false },
		UserHomeDir: func() (string, error) { return "", want },
	}
	if _, err := resolver.Resolve(); !errors.Is(err, want) {
		t.Fatalf("got %v, want wrapped error", err)
	}
}
