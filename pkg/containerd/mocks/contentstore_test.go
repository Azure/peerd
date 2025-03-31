package mocks

import (
	"context"
	"testing"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestInfo(t *testing.T) {
	mcs := &MockContentStore{
		Data: map[string]string{
			"sha256:1234567890abcdefg": `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":"sha256:1234567890abcdef","size":123},"layers":[{"digest":"sha256:1234567890abcdef","size":123}]}`,
		},
	}

	tests := []struct {
		dgst    digest.Digest
		wantErr bool
	}{
		{
			dgst:    digest.Digest("sha256:1234567890abcdefg"),
			wantErr: false,
		},
		{
			dgst:    digest.Digest("sha256:nonexistent"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.dgst.String(), func(t *testing.T) {
			got, err := mcs.Info(context.Background(), test.dgst)
			if (err != nil) != test.wantErr {
				t.Errorf("Info() error = %v, wantErr %v", err, test.wantErr)
			}

			if !test.wantErr {
				if got.Digest != test.dgst {
					t.Errorf("Info() got = %v, want %v", got.Digest, test.dgst)
				}
			}
		})
	}
}

func TestReaderAt(t *testing.T) {
	mcs := &MockContentStore{
		Data: map[string]string{
			"sha256:1234567890abcdefg": `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"digest":"sha256:1234567890abcdef","size":123},"layers":[{"digest":"sha256:1234567890abcdef","size":123}]}`,
		},
	}

	tests := []struct {
		desc    v1.Descriptor
		wantErr bool
	}{
		{
			desc: v1.Descriptor{
				Digest: digest.Digest("sha256:1234567890abcdefg"),
				Size:   192,
			},
			wantErr: false,
		},
		{
			desc: v1.Descriptor{
				Digest: digest.Digest("sha256:nonexistent"),
				Size:   123,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc.Digest.String(), func(t *testing.T) {
			got, err := mcs.ReaderAt(context.Background(), test.desc)
			if (err != nil) != test.wantErr {
				t.Errorf("ReaderAt() error = %v, wantErr %v", err, test.wantErr)
			}

			if !test.wantErr {
				if got.Size() != test.desc.Size {
					t.Errorf("ReaderAt() got size = %d, want %d", got.Size(), test.desc.Size)
				}
				got.Close()
			}
		})
	}
}

func TestContentStorePanics(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "TestDelete",
			fn: func() {
				m := &MockContentStore{}
				m.Delete(context.Background(), digest.Digest("sha256:1234567890abcdef"))
			},
		},
		{
			name: "TestWalk",
			fn: func() {
				m := &MockContentStore{}
				m.Walk(context.Background(), nil)
			},
		},
		{
			name: "TestStatus",
			fn: func() {
				m := &MockContentStore{}
				m.Status(context.Background(), "test")
			},
		},
		{
			name: "TestUpdate",
			fn: func() {
				m := &MockContentStore{}
				m.Update(context.Background(), content.Info{}, "")
			},
		},
		{
			name: "TestListStatuses",
			fn: func() {
				m := &MockContentStore{}
				m.ListStatuses(context.Background(), "")
			},
		},
		{
			name: "TestWriter",
			fn: func() {
				m := &MockContentStore{}
				m.Writer(context.Background())
			},
		},
		{
			name: "TestAbort",
			fn: func() {
				m := &MockContentStore{}
				m.Abort(context.Background(), "test")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic, but did not get one")
				}
			}()
			test.fn()
		})
	}
}
