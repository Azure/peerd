// Package urlparser provides interfaces and implementations for parsing information from a URL.
package urlparser

import (
	"github.com/opencontainers/go-digest"
)

// Parser describes an interface for parsing information from a URL.
type Parser interface {
	// ParseDigest parses the digest from the given URL.
	// If none found, implementations should return an error.
	ParseDigest(url string) (digest.Digest, error)
}

type parser struct{}

var _ Parser = &parser{}

// ParseDigest parses the digest from the given URL.
// If none found, returns an error.
func (p *parser) ParseDigest(url string) (digest.Digest, error) {
	return parseDigestFromAzureUrl(url)
}

// New returns a new Parser.
func New() Parser {
	return &parser{}
}
