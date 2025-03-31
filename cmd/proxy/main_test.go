package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestAddMirrorConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	configPath := tempDir + "/etc/containerd/certs.d"

	sc := &ServerCmd{
		AddMirrorConfiguration:    false,
		Mirrors:                   []string{"https://localhost:5000"},
		Hosts:                     []string{"https://mcr.microsoft.com"},
		ContainerdHostsConfigPath: configPath,
	}

	ctx := context.Background()

	if err := addMirrorConfiguration(ctx, sc); err != nil {
		t.Fatalf("failed to add mirror configuration: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected config directory to be empty, but it exists: %v", err)
	}

	sc.AddMirrorConfiguration = true
	if err := addMirrorConfiguration(ctx, sc); err != nil {
		t.Fatalf("failed to add mirror configuration: %v", err)
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config directory to exist, but it does not: %v", err)
	}
	hostDir := configPath + "/mcr.microsoft.com"
	if _, err := os.Stat(hostDir); os.IsNotExist(err) {
		t.Fatalf("expected host directory to exist, but it does not: %v", err)
	}

	mirrorFile := hostDir + "/hosts.toml"
	if _, err := os.Stat(mirrorFile); os.IsNotExist(err) {
		t.Fatalf("expected mirror file to exist, but it does not: %v", err)
	}
	contents, err := os.ReadFile(mirrorFile)
	if err != nil {
		t.Fatalf("failed to read mirror file: %v", err)
	}
	if !strings.Contains(string(contents), "https://localhost:5000") {
		t.Fatalf("expected mirror file to contain the mirror URL, but it does not: %v", err)
	}

}

func TestToUrls(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "single URL",
			args: []string{"http://example.com"},
			want: []string{"http://example.com"},
		},
		{
			name: "multiple URLs",
			args: []string{"http://example.com", "https://example.org"},
			want: []string{"http://example.com", "https://example.org"},
		},
		{
			name: "invalid URL",
			args: []string{"invalid-url"},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toUrls(tt.args)
			if err != nil {
				t.Errorf("toUrls() error = %v", err)
				return
			}
			if len(tt.want) == 0 {
				// Throw an error if we expect no valid URLs and at least some have a valid hostname.
				for _, u := range got {
					if u.Hostname() != "" {
						t.Fatalf("toUrls() got = %v, want %v", got, tt.want)
					}
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("toUrls() got = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i].String() != tt.want[i] {
					t.Errorf("toUrls() got = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}
