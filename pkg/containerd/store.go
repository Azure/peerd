// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/containerd/containerd"
	eventtypes "github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/platforms"
	"github.com/containerd/typeurl/v2"
	"github.com/distribution/distribution/manifest"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// DefaultSock is the default containerd socket path.
	DefaultSock = "/run/containerd/containerd.sock"

	// DefaultNamespace is the default containerd namespace for this client.
	DefaultNamespace = "k8s.io"
)

// Store is the interface for all containerd content store artifacts.
type Store interface {
	// Subscribe returns a channel of artifacts and a channel of errors.
	// Artifacts are sent on the channel as they are discovered.
	Subscribe(ctx context.Context) (<-chan Reference, <-chan error)

	// List returns a list of artifacts.
	List(ctx context.Context) ([]Reference, error)

	// Resolve returns the digest for an existing artifact.
	Resolve(ctx context.Context, ref string) (digest.Digest, error)

	// Size returns the size of the artifact.
	Size(ctx context.Context, dgst digest.Digest) (int64, error)

	// Bytes returns the artifact bytes.
	Bytes(ctx context.Context, dgst digest.Digest) ([]byte, string, error)

	// Write writes the artifact bytes to the writer.
	Write(ctx context.Context, dst io.Writer, dgst digest.Digest) error

	// Verify will verify that the client status is healthy.
	Verify(ctx context.Context) error

	// All returns a list of digests of all resources referenced in ref.
	All(ctx context.Context, ref Reference) ([]string, error)
}

// store provides an interface to the containerd content store.
type store struct {
	client   *containerd.Client
	platform platforms.MatchComparer

	// Filters for list and event subscriptions.
	// The syntax of these filters is defined here: https://github.com/containerd/containerd/blob/main/filters/filter.go
	listFilter  string
	eventFilter string
}

var _ Store = &store{}

// NewDefaultStore creates a new Store with default values for containerd socket, namespace and hosts configuration path.
func NewDefaultStore(hosts []string) (Store, error) {
	return NewStore(DefaultSock, DefaultNamespace, hosts)
}

// NewStore creates a new Store.
func NewStore(sock, ns string, hosts []string) (Store, error) {
	if sock == "" {
		return nil, fmt.Errorf("containerd socket path cannot be empty")
	}

	if ns == "" {
		return nil, fmt.Errorf("containerd namespace cannot be empty")
	}

	client, err := containerd.New(sock, containerd.WithDefaultNamespace(ns))
	if err != nil {
		return nil, fmt.Errorf("could not create containerd client: %w", err)
	}

	return newStore(hosts, client)
}

func newStore(hosts []string, client *containerd.Client) (*store, error) {
	for _, host := range hosts {
		_, err := url.Parse(host)
		if err != nil {
			return nil, err
		}
	}

	return &store{
		client:      client,
		platform:    platforms.Default(),
		listFilter:  getListFilter(hosts),
		eventFilter: getEventFilter(hosts),
	}, nil
}

// Verify will verify that the containerd service is serving at the configured socket.
func (c *store) Verify(ctx context.Context) error {
	ok, err := c.client.IsServing(ctx)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("could not reach containerd service")
	}

	return nil
}

// Subscribe provides a subscription to containerd events on the configured hosts artifacts.
// It also returns a channel of errors.
func (c *store) Subscribe(ctx context.Context) (<-chan Reference, <-chan error) {
	refChan := make(chan Reference)
	errChan := make(chan error)

	eventsChan, eventsErrChan := c.client.EventService().Subscribe(ctx, c.eventFilter)
	go func() {
		for event := range eventsChan {
			name, err := getEventImageName(event.Event)
			if err != nil {
				errChan <- err
				continue
			}

			image, err := c.client.GetImage(ctx, name)
			if err != nil {
				errChan <- err
				continue
			}

			ref, err := ParseReference(image.Name(), image.Target().Digest)
			if err != nil {
				errChan <- err
			} else {
				refChan <- ref
			}
		}
	}()

	go func() {
		for err := range eventsErrChan {
			errChan <- err
		}
	}()

	return refChan, errChan
}

// List returns the list of locally found images.
func (c *store) List(ctx context.Context) ([]Reference, error) {
	imgs, err := c.client.ListImages(ctx, c.listFilter)
	if err != nil {
		return nil, err
	}

	refs := []Reference{}
	for _, img := range imgs {
		ref, err := ParseReference(img.Name(), img.Target().Digest)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// All returns a list of digests of all resources referenced in ref.
func (c *store) All(ctx context.Context, ref Reference) ([]string, error) {
	img, err := c.client.ImageService().Get(ctx, ref.Name())
	if err != nil {
		return nil, err
	}

	keys := []string{}

	err = images.Walk(ctx, images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		keys = append(keys, desc.Digest.String())

		switch desc.MediaType {
		case images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:
			var idx ocispec.Index

			b, err := content.ReadBlob(ctx, c.client.ContentStore(), desc)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(b, &idx); err != nil {
				return nil, err
			}

			var descs []ocispec.Descriptor
			for _, m := range idx.Manifests {
				if !c.platform.Match(*m.Platform) {
					continue
				}
				descs = append(descs, m)
			}
			if len(descs) == 0 {
				return nil, fmt.Errorf("could not find platform architecture in manifest: %v", desc.Digest)
			}

			// Platform matching is a bit weird in that multiple platforms can match.
			// There is however a "best" match that should be used.
			// This logic is used by Containerd to determine which layer to pull so we should use the same logic.
			sort.SliceStable(descs, func(i, j int) bool {
				if descs[i].Platform == nil {
					return false
				}
				if descs[j].Platform == nil {
					return true
				}
				return c.platform.Less(*descs[i].Platform, *descs[j].Platform)
			})
			return []ocispec.Descriptor{descs[0]}, nil

		case images.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest:
			var manifest ocispec.Manifest
			b, err := content.ReadBlob(ctx, c.client.ContentStore(), desc)
			if err != nil {
				return nil, err
			}

			if err := json.Unmarshal(b, &manifest); err != nil {
				return nil, err
			}
			keys = append(keys, manifest.Config.Digest.String())
			for _, layer := range manifest.Layers {
				keys = append(keys, layer.Digest.String())
			}
			return nil, nil

		default:
			return nil, fmt.Errorf("unexpected media type %v for digest: %v", desc.MediaType, desc.Digest)
		}
	}), img.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to walk image manifests: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no image digests found")
	}

	return keys, nil
}

// Resolve returns the digest for an existing artifact.
func (c *store) Resolve(ctx context.Context, ref string) (digest.Digest, error) {
	cImg, err := c.client.GetImage(ctx, ref)
	if err != nil {
		return "", err
	}
	return cImg.Target().Digest, nil
}

// Size returns the size of the artifact.
func (c *store) Size(ctx context.Context, dgst digest.Digest) (int64, error) {
	info, err := c.client.ContentStore().Info(ctx, dgst)
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

// Bytes returns the artifact bytes. This method should only be used for manifests.
func (c *store) Bytes(ctx context.Context, dgst digest.Digest) ([]byte, string, error) {
	b, err := content.ReadBlob(ctx, c.client.ContentStore(), ocispec.Descriptor{Digest: dgst})
	if err != nil {
		return nil, "", err
	}
	var m manifest.Versioned
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, "", err
	}

	return b, m.MediaType, nil
}

// Write writes the blob bytes to the writer.
func (c *store) Write(ctx context.Context, dst io.Writer, dgst digest.Digest) error {
	ra, err := c.client.ContentStore().ReaderAt(ctx, ocispec.Descriptor{Digest: dgst})
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := ra.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("multiple errors: %v; %v", err, closeErr)
			} else {
				err = closeErr
			}
		}
	}()

	_, err = io.Copy(dst, content.NewReader(ra))
	if err != nil {
		return err
	}

	return nil
}

// getEventImageName will get the image name from an event.
func getEventImageName(e typeurl.Any) (string, error) {
	evt, err := typeurl.UnmarshalAny(e)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal any: %w", err)
	}

	switch e := evt.(type) {
	case *eventtypes.ImageCreate:
		return e.Name, nil
	case *eventtypes.ImageUpdate:
		return e.Name, nil
	default:
		return "", fmt.Errorf("unsupported event: %v", e)
	}
}

func getListFilter(hosts []string) string {
	return fmt.Sprintf(`name~="%s"`, strings.Join(getHostNames(hosts), "|"))
}

func getEventFilter(hosts []string) string {
	return fmt.Sprintf(`topic~="/images/create|/images/update",event.name~="%s"`, strings.Join(getHostNames(hosts), "|"))
}

func getHostNames(hosts []string) []string {
	names := []string{}
	for _, host := range hosts {
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "http://" + host
		}

		u, err := url.Parse(host)
		if err != nil {
			// Use the host as is.
			names = append(names, host)
		} else {
			names = append(names, u.Host)
		}
	}
	return names
}
