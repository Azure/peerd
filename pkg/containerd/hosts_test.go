// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"context"
	iofs "io/fs"
	"net/url"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestMirrorConfiguration(t *testing.T) {
	registryConfigPath := "/etc/containerd/certs.d"

	tests := []struct {
		name                string
		resolveTags         bool
		registries          []url.URL
		mirrors             []url.URL
		createConfigPathDir bool
		existingFiles       map[string]string
		expectedFiles       map[string]string
	}{
		{
			name:        "multiple mirros",
			resolveTags: true,
			registries:  stringListToUrlList(t, []string{"http://foo.bar:5000"}),
			mirrors:     stringListToUrlList(t, []string{"http://127.0.0.1:5000", "http://127.0.0.1:5001"}),
			expectedFiles: map[string]string{
				"/etc/containerd/certs.d/foo.bar:5000/hosts.toml": `server = 'http://foo.bar:5000'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true

[host.'http://127.0.0.1:5001']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
			},
		},
		{
			name:        "resolve tags disabled",
			resolveTags: false,
			registries:  stringListToUrlList(t, []string{"https://docker.io", "http://foo.bar:5000"}),
			mirrors:     stringListToUrlList(t, []string{"http://127.0.0.1:5000"}),
			expectedFiles: map[string]string{
				"/etc/containerd/certs.d/docker.io/hosts.toml": `server = 'https://registry-1.docker.io'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull']
skip_verify = true
`,
				"/etc/containerd/certs.d/foo.bar:5000/hosts.toml": `server = 'http://foo.bar:5000'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull']
skip_verify = true
`,
			},
		},
		{
			name:                "config path directory does not exist",
			resolveTags:         true,
			registries:          stringListToUrlList(t, []string{"https://docker.io", "http://foo.bar:5000"}),
			mirrors:             stringListToUrlList(t, []string{"http://127.0.0.1:5000"}),
			createConfigPathDir: false,
			expectedFiles: map[string]string{
				"/etc/containerd/certs.d/docker.io/hosts.toml": `server = 'https://registry-1.docker.io'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
				"/etc/containerd/certs.d/foo.bar:5000/hosts.toml": `server = 'http://foo.bar:5000'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
			},
		},
		{
			name:                "config path directory does exist",
			resolveTags:         true,
			registries:          stringListToUrlList(t, []string{"https://docker.io", "http://foo.bar:5000"}),
			mirrors:             stringListToUrlList(t, []string{"http://127.0.0.1:5000"}),
			createConfigPathDir: true,
			expectedFiles: map[string]string{
				"/etc/containerd/certs.d/docker.io/hosts.toml": `server = 'https://registry-1.docker.io'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
				"/etc/containerd/certs.d/foo.bar:5000/hosts.toml": `server = 'http://foo.bar:5000'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
			},
		},
		{
			name:                "config path directory contains configuration",
			resolveTags:         true,
			registries:          stringListToUrlList(t, []string{"https://docker.io", "http://foo.bar:5000"}),
			mirrors:             stringListToUrlList(t, []string{"http://127.0.0.1:5000"}),
			createConfigPathDir: true,
			existingFiles: map[string]string{
				"/etc/containerd/certs.d/docker.io/hosts.toml": "Hello World",
				"/etc/containerd/certs.d/ghcr.io/hosts.toml":   "Foo Bar",
			},
			expectedFiles: map[string]string{
				"/etc/containerd/certs.d/_backup/docker.io/hosts.toml": "Hello World",
				"/etc/containerd/certs.d/_backup/ghcr.io/hosts.toml":   "Foo Bar",
				"/etc/containerd/certs.d/docker.io/hosts.toml": `server = 'https://registry-1.docker.io'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
				"/etc/containerd/certs.d/foo.bar:5000/hosts.toml": `server = 'http://foo.bar:5000'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
			},
		},
		{
			name:                "config path directory contains backup",
			resolveTags:         true,
			registries:          stringListToUrlList(t, []string{"https://docker.io", "http://foo.bar:5000"}),
			mirrors:             stringListToUrlList(t, []string{"http://127.0.0.1:5000"}),
			createConfigPathDir: true,
			existingFiles: map[string]string{
				"/etc/containerd/certs.d/_backup/docker.io/hosts.toml": "Hello World",
				"/etc/containerd/certs.d/_backup/ghcr.io/hosts.toml":   "Foo Bar",
				"/etc/containerd/certs.d/test.txt":                     "test",
				"/etc/containerd/certs.d/foo":                          "bar",
			},
			expectedFiles: map[string]string{
				"/etc/containerd/certs.d/_backup/docker.io/hosts.toml": "Hello World",
				"/etc/containerd/certs.d/_backup/ghcr.io/hosts.toml":   "Foo Bar",
				"/etc/containerd/certs.d/docker.io/hosts.toml": `server = 'https://registry-1.docker.io'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
				"/etc/containerd/certs.d/foo.bar:5000/hosts.toml": `server = 'http://foo.bar:5000'

[host]
[host.'http://127.0.0.1:5000']
capabilities = ['pull', 'resolve']
skip_verify = true
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.createConfigPathDir {
				err := fs.Mkdir(registryConfigPath, 0755)
				require.NoError(t, err)
			}

			for k, v := range tt.existingFiles {
				err := afero.WriteFile(fs, k, []byte(v), 0644)
				require.NoError(t, err)
			}

			err := AddHostsConfiguration(context.TODO(), fs, registryConfigPath, tt.registries, tt.mirrors, tt.resolveTags)
			require.NoError(t, err)

			if len(tt.existingFiles) == 0 {
				ok, err := afero.DirExists(fs, "/etc/containerd/certs.d/_backup")
				require.NoError(t, err)
				require.False(t, ok)
			}

			err = afero.Walk(fs, registryConfigPath, func(path string, fi iofs.FileInfo, _ error) error {
				if fi.IsDir() {
					return nil
				}
				expectedContent, ok := tt.expectedFiles[path]
				require.True(t, ok, path)
				b, err := afero.ReadFile(fs, path)
				require.NoError(t, err)
				require.Equal(t, expectedContent, string(b))
				return nil
			})
			require.NoError(t, err)
		})
	}
}

func TestMirrorConfigurationInvalidMirrorURL(t *testing.T) {
	fs := afero.NewMemMapFs()
	mirrors := stringListToUrlList(t, []string{"http://127.0.0.1:5000"})

	registries := stringListToUrlList(t, []string{"ftp://docker.io"})
	err := AddHostsConfiguration(context.TODO(), fs, "/etc/containerd/certs.d", registries, mirrors, true)
	require.EqualError(t, err, "invalid registry url, scheme must be http or https, got: ftp://docker.io")

	registries = stringListToUrlList(t, []string{"https://docker.io/foo/bar"})
	err = AddHostsConfiguration(context.TODO(), fs, "/etc/containerd/certs.d", registries, mirrors, true)
	require.EqualError(t, err, "invalid registry url, path has to be empty, got: https://docker.io/foo/bar")

	registries = stringListToUrlList(t, []string{"https://docker.io?foo=bar"})
	err = AddHostsConfiguration(context.TODO(), fs, "/etc/containerd/certs.d", registries, mirrors, true)
	require.EqualError(t, err, "invalid registry url, query has to be empty, got: https://docker.io?foo=bar")

	registries = stringListToUrlList(t, []string{"https://foo@docker.io"})
	err = AddHostsConfiguration(context.TODO(), fs, "/etc/containerd/certs.d", registries, mirrors, true)
	require.EqualError(t, err, "invalid registry url, user has to be empty, got: https://foo@docker.io")
}

func stringListToUrlList(t *testing.T, list []string) []url.URL {
	t.Helper()
	urls := []url.URL{}
	for _, item := range list {
		u, err := url.Parse(item)
		require.NoError(t, err)
		urls = append(urls, *u)
	}
	return urls
}
