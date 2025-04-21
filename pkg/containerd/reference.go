// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/opencontainers/go-digest"
)

var separator = regexp.MustCompile(`[:@]`)

// Reference is a reference to an OCI artifact in the content store.
type Reference interface {
	// Name is the name of the artifact.
	Name() string

	// Digest is the digest of the artifact.
	Digest() digest.Digest

	// Tag is the tag of the artifact.
	Tag() string

	// Repository is the repository of the artifact.
	Repository() string

	// Host is the host of the artifact.
	Host() string

	// String returns the string representation of the reference.
	// docker.io/library/ubuntu:latest
	// docker.io/library/ubuntu@sha256:abcdef
	String() string
}

// Reference is a reference to an OCI artifact.
type reference struct {
	// name is the name of the artifact.
	name string

	// dgst is the digest of the artifact.
	dgst digest.Digest

	// tag is the tag of the artifact.
	tag string

	// repo is the repository of the artifact.
	repo string

	// host is the host of the artifact.
	host string
}

var _ Reference = &reference{}

// Name is the name of the artifact.
func (r *reference) Name() string {
	return r.name
}

// Digest is the digest of the artifact.
func (r *reference) Digest() digest.Digest {
	return r.dgst
}

// Tag is the tag of the artifact.
func (r *reference) Tag() string {
	return r.tag
}

// Repository is the repository of the artifact.
func (r *reference) Repository() string {
	return r.repo
}

// Host is the host of the artifact.
func (r *reference) Host() string {
	return r.host
}

// String returns the string representation of the reference.
func (r *reference) String() string {
	if r.tag != "" {
		return fmt.Sprintf("%s/%s:%s", r.Host(), r.Repository(), r.Tag())
	}
	return fmt.Sprintf("%s/%s@%s", r.Host(), r.Repository(), r.Digest())
}

// ParseReference parses the given name into a reference.
// targetDigest is obtained from the containerd interface, and is used to verify the parsed digest, or to set the digest if it is not present.
func ParseReference(name string, targetDigest digest.Digest) (Reference, error) {
	if strings.Contains(name, "://") {
		return nil, fmt.Errorf("invalid reference")
	}

	u, err := url.Parse("localhost://" + name)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "localhost" {
		return nil, fmt.Errorf("invalid reference")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("hostname required")
	}

	var obj string
	if idx := separator.FindStringIndex(u.Path); idx != nil {
		// This allows us to retain the @ to signify digests or shortened digests in the object.
		obj = u.Path[idx[0]:]
		if obj[:1] == ":" {
			obj = obj[1:]
		}
		u.Path = u.Path[:idx[0]]
	}

	tag, dgst := splitTagAndDigest(obj)
	tag, _, _ = strings.Cut(tag, "@")
	repository := strings.TrimPrefix(u.Path, "/")

	if dgst == "" {
		dgst = targetDigest
	}

	if targetDigest != "" && dgst != targetDigest {
		return nil, fmt.Errorf("invalid digest, target does not match parsed digest: %v %v", name, dgst)
	}

	if repository == "" {
		return nil, fmt.Errorf("invalid repository: %v", repository)
	}

	if dgst == "" {
		return nil, fmt.Errorf("invalid digest: %v", dgst)
	}

	return &reference{
		name: name,
		host: u.Host,
		tag:  tag,
		dgst: dgst,
		repo: repository,
	}, nil
}

func splitTagAndDigest(obj string) (tag string, dgst digest.Digest) {
	parts := strings.SplitAfterN(obj, "@", 2)
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], digest.Digest(parts[1])
}
