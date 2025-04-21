// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

const (
	backupDir = "_backup"
)

type hostFile struct {
	Server      string                `toml:"server"`
	HostConfigs map[string]hostConfig `toml:"host"`
}

type hostConfig struct {
	Capabilities []string `toml:"capabilities"`
	SkipVerify   bool     `toml:"skip_verify"`
}

// AddHostsConfiguration adds mirror configuration to containerd for the specified URLs.
// Refer to containerd registry configuration documentation for mor information about required configuration.
// https://github.com/containerd/containerd/blob/main/docs/cri/config.md#registry-configuration
// https://github.com/containerd/containerd/blob/main/docs/hosts.md#registry-configuration---examples
func AddHostsConfiguration(ctx context.Context, fs afero.Fs, configPath string, registryURLs, mirrorURLs []url.URL, resolveTags bool) error {
	log := zerolog.Ctx(ctx).With().Str("component", "containerd-mirror").Logger()

	if err := validate(registryURLs); err != nil {
		return err
	}

	// Create config path dir if it does not exist
	ok, err := afero.DirExists(fs, configPath)
	if err != nil {
		return err
	}

	if !ok {
		err := fs.MkdirAll(configPath, 0755)
		if err != nil {
			return err
		}
	}

	// Backup files and directories in config path
	backupDirPath := path.Join(configPath, backupDir)
	if _, err := fs.Stat(backupDirPath); os.IsNotExist(err) {
		files, err := afero.ReadDir(fs, configPath)
		if err != nil {
			return err
		}
		if len(files) > 0 {
			err = fs.MkdirAll(backupDirPath, 0755)
			if err != nil {
				return err
			}
			for _, fi := range files {
				oldPath := path.Join(configPath, fi.Name())
				newPath := path.Join(backupDirPath, fi.Name())
				err := fs.Rename(oldPath, newPath)
				if err != nil {
					return err
				}
				log.Info().Str("path", oldPath).Str("target", newPath).Msg("backing up Containerd host configuration")
			}
		}
	}

	// Remove all content from config path to start from a clean slate
	files, err := afero.ReadDir(fs, configPath)
	if err != nil {
		return err
	}
	for _, fi := range files {
		if fi.Name() == backupDir {
			continue
		}
		filePath := path.Join(configPath, fi.Name())
		err := fs.RemoveAll(filePath)
		if err != nil {
			return err
		}
	}

	// Write mirror configuration
	capabilities := []string{"pull"}
	if resolveTags {
		capabilities = append(capabilities, "resolve")
	}
	for _, registryURL := range registryURLs {
		// Need a special case for Docker Hub as docker.io is just an alias.
		server := registryURL.String()
		if registryURL.String() == "https://docker.io" {
			server = "https://registry-1.docker.io"
		}

		hostConfigs := map[string]hostConfig{}
		for _, u := range mirrorURLs {
			hostConfigs[u.String()] = hostConfig{Capabilities: capabilities, SkipVerify: true} // nolint: gosec. TODO avtakkar: configure TLS.
		}

		cfg := hostFile{
			Server:      server,
			HostConfigs: hostConfigs,
		}

		b, err := toml.Marshal(&cfg)
		if err != nil {
			return err
		}

		fp := path.Join(configPath, registryURL.Host, "hosts.toml")
		err = fs.MkdirAll(path.Dir(fp), 0755)
		if err != nil {
			return err
		}

		err = afero.WriteFile(fs, fp, b, 0644)
		if err != nil {
			return err
		}

		log.Info().Str("host", registryURL.String()).Str("path", fp).Msg("added containerd mirror configuration")
	}

	return nil
}

// validate validates registry URLs.
func validate(urls []url.URL) error {
	errs := []error{}
	for _, u := range urls {
		if u.Scheme != "http" && u.Scheme != "https" {
			errs = append(errs, fmt.Errorf("invalid registry url, scheme must be http or https, got: %s", u.String()))
		}

		if u.Path != "" {
			errs = append(errs, fmt.Errorf("invalid registry url, path has to be empty, got: %s", u.String()))
		}

		if len(u.Query()) != 0 {
			errs = append(errs, fmt.Errorf("invalid registry url, query has to be empty, got: %s", u.String()))
		}

		if u.User != nil {
			errs = append(errs, fmt.Errorf("invalid registry url, user has to be empty, got: %s", u.String()))
		}
	}
	return errors.Join(errs...)
}
