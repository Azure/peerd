package mocks

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd/images"
)

// MockImageStore is a mock implementation of containerd's image store.
type MockImageStore struct {
	Data map[string]images.Image
}

var _ images.Store = &MockImageStore{}

// Get gets an image by name if it exists in the mocked data keyed by name.
func (m *MockImageStore) Get(ctx context.Context, name string) (images.Image, error) {
	img, ok := m.Data[name]
	if !ok {
		return images.Image{}, fmt.Errorf("image with name %s does not exist", name)
	}
	return img, nil
}

// List lists the images in the image store filtered by the given filters.
// Note that only some filters are recognized by this mock implementation.
func (m *MockImageStore) List(ctx context.Context, filters ...string) ([]images.Image, error) {
	result := []images.Image{}
	for _, filter := range filters {
		if strings.HasPrefix(filter, "name~=") {
			n := strings.TrimLeft(filter, "name~=")
			names := strings.Split(n, "|")
			for _, name := range names {
				name = strings.TrimLeft(name, "\"")
				name = strings.TrimRight(name, "\"")
				for k, v := range m.Data {
					if strings.HasPrefix(k, name) {
						result = append(result, v)
					}
				}
			}
		}
	}
	return result, nil
}

func (*MockImageStore) Create(ctx context.Context, image images.Image) (images.Image, error) {
	return images.Image{}, nil
}

func (*MockImageStore) Update(ctx context.Context, image images.Image, fieldpaths ...string) (images.Image, error) {
	return images.Image{}, nil
}

func (*MockImageStore) Delete(ctx context.Context, name string, opts ...images.DeleteOpt) error {
	return nil
}
