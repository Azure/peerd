// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package urlparser

import (
	"fmt"
	"regexp"

	"github.com/opencontainers/go-digest"
)

var (
	regexes = []*regexp.Regexp{
		// Azure Container Registry public cloud data endpoints.
		regexp.MustCompile(`https:\/\/[a-zA-Z0-9\.]+\.azurecr\.[a-z\.]+\?[a-zA-Z0-9\.\&\=\-]+\&d=sha256:([a-zA-Z0-9]{64})[.]*`),

		// Microsoft Artifact Registry public cloud data endpoints.
		regexp.MustCompile(`https:\/\/[a-zA-Z0-9]+\.data.mcr.microsoft.com\/[a-zA-Z0-9\-]+\/\/docker\/registry\/v2\/blobs\/sha256\/[a-z0-9]{2}\/([a-zA-Z0-9]{64})\/data.*`),

		// Azure Blob Storage public cloud blob endpoints.
		regexp.MustCompile(`https:\/\/[a-zA-Z0-9]+\.blob\.[a-z\.]+\/[a-zA-Z0-9\-]+\/\/docker\/registry\/v2\/blobs\/sha256\/[a-z0-9]{2}\/([a-zA-Z0-9]{64})\/data.*`),
	}
)

// parseDigestFromAzureUrl parses the digest from the given blob url or returns an error.
func parseDigestFromAzureUrl(url string) (digest.Digest, error) {
	if url == "" {
		return "", fmt.Errorf("empty url")
	}

	for _, r := range regexes {
		matches := r.FindStringSubmatch(url)
		if len(matches) == 2 {
			return digest.Digest("sha256:" + matches[1]), nil
		}
	}

	return "", fmt.Errorf("unknown url")
}
