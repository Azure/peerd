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
				err := got.Close()
				if err != nil {
					t.Errorf("ReaderAt() error closing reader = %v", err)
				}
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
				err := m.Delete(context.Background(), digest.Digest("sha256:1234567890abcdef"))
				if err != nil {
					t.Errorf("Delete() error = %v", err)
				}
			},
		},
		{
			name: "TestWalk",
			fn: func() {
				m := &MockContentStore{}
				err := m.Walk(context.Background(), nil)
				if err != nil {
					t.Errorf("Walk() error = %v", err)
				}
			},
		},
		{
			name: "TestStatus",
			fn: func() {
				m := &MockContentStore{}
				_, err := m.Status(context.Background(), "test")
				if err != nil {
					t.Errorf("Status() error = %v", err)
				}
			},
		},
		{
			name: "TestUpdate",
			fn: func() {
				m := &MockContentStore{}
				_, err := m.Update(context.Background(), content.Info{}, "")
				if err != nil {
					t.Errorf("Update() error = %v", err)
				}
			},
		},
		{
			name: "TestListStatuses",
			fn: func() {
				m := &MockContentStore{}
				_, err := m.ListStatuses(context.Background(), "")
				if err != nil {
					t.Errorf("ListStatuses() error = %v", err)
				}
			},
		},
		{
			name: "TestWriter",
			fn: func() {
				m := &MockContentStore{}
				_, err := m.Writer(context.Background())
				if err != nil {
					t.Errorf("Writer() error = %v", err)
				}
			},
		},
		{
			name: "TestAbort",
			fn: func() {
				m := &MockContentStore{}
				err := m.Abort(context.Background(), "test")
				if err != nil {
					t.Errorf("Abort() error = %v", err)
				}
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
