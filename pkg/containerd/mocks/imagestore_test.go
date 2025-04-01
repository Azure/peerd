package mocks

import (
	"context"
	"testing"

	"github.com/containerd/containerd/images"
)

func TestMockImageStoreGet(t *testing.T) {
	m := &MockImageStore{
		Data: map[string]images.Image{
			"test": {
				Name: "test",
			},
		},
	}

	got, err := m.Get(context.Background(), "test")
	if err != nil {
		t.Errorf("Get() error = %v", err)
		return
	}

	if got.Name != "test" {
		t.Errorf("Get() got = %v, want %v", got.Name, "test")
	}

	_, err = m.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Errorf("Get() expected error, got nil")
		return
	}
}

func TestMockImageStoreList(t *testing.T) {
	m := &MockImageStore{
		Data: map[string]images.Image{
			"test1": {
				Name: "test1",
			},
			"test2": {
				Name: "test2",
			},
			"hello": {
				Name: "hello",
			},
		},
	}

	got, err := m.List(context.Background(), "name~=\"test\"")
	if err != nil {
		t.Errorf("List() error = %v", err)
		return
	}

	if len(got) != 2 {
		t.Errorf("List() got = %v, want %v", len(got), 2)
	}
	foundTest1 := false
	foundTest2 := false
	for _, img := range got {
		if img.Name == "test1" {
			foundTest1 = true
		}
		if img.Name == "test2" {
			foundTest2 = true
		}
	}
	if !foundTest1 || !foundTest2 {
		t.Errorf("List() did not return expected images, got = %v", got)
	}

	_, err = m.List(context.Background(), "name~=\"nonexistent\"")
	if err != nil {
		t.Errorf("List() expected error, got nil")
		return
	}
}

func TestImageStorePanics(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "TestCreate",
			fn: func() {
				m := &MockImageStore{}
				_, err := m.Create(context.Background(), images.Image{})
				if err != nil {
					t.Errorf("Create() error = %v", err)
				}
			},
		},
		{
			name: "TestUpdate",
			fn: func() {
				m := &MockImageStore{}
				_, err := m.Update(context.Background(), images.Image{}, "fieldpath")
				if err != nil {
					t.Errorf("Update() error = %v", err)
				}
			},
		},
		{
			name: "TestDelete",
			fn: func() {
				m := &MockImageStore{}
				err := m.Delete(context.Background(), "name")
				if err != nil {
					t.Errorf("Delete() error = %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic, but did not panic")
				}
			}()
			tt.fn()
		})
	}
}
