// Copyright (c) Microsoft Corporation.
// Portions (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Licensed under the MIT License.
package distribution

import (
	"fmt"
	"regexp"

	"github.com/opencontainers/go-digest"
)

// ReferenceType is the type of reference - manifest or blob.
type ReferenceType string

const (
	ReferenceTypeManifest = "Manifest"
	ReferenceTypeBlob     = "Blob"
)

var (
	nameRegex           = regexp.MustCompile(`([a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*)`)
	tagRegex            = regexp.MustCompile(`([a-zA-Z0-9_][a-zA-Z0-9._-]{0,127})`)
	manifestRegexTag    = regexp.MustCompile(`/v2/` + nameRegex.String() + `/manifests/` + tagRegex.String() + `$`)
	manifestRegexDigest = regexp.MustCompile(`/v2/` + nameRegex.String() + `/manifests/(.*)`)
	blobsRegexDigest    = regexp.MustCompile(`/v2/` + nameRegex.String() + `/blobs/(.*)`)
)

// ParsePathComponents parses the registry, digest and reference type from a distribution path.
func ParsePathComponents(registry, path string) (string, digest.Digest, ReferenceType, error) {
	comps := manifestRegexTag.FindStringSubmatch(path)
	if len(comps) == 6 {
		if registry == "" {
			return "", "", "", fmt.Errorf("registry parameter needs to be set for tag references")
		}
		ref := fmt.Sprintf("%s/%s:%s", registry, comps[1], comps[5])
		return ref, "", ReferenceTypeManifest, nil
	}
	comps = manifestRegexDigest.FindStringSubmatch(path)
	if len(comps) == 6 {
		return "", digest.Digest(comps[5]), ReferenceTypeManifest, nil
	}
	comps = blobsRegexDigest.FindStringSubmatch(path)
	if len(comps) == 6 {
		return "", digest.Digest(comps[5]), ReferenceTypeBlob, nil
	}
	return "", "", "", fmt.Errorf("distribution path could not be parsed")
}
