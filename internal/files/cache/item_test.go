// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package cache

import (
	"crypto/rand"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestWriteAll(t *testing.T) {
	// Setup
	l := zerolog.Nop()
	name := newRandomStringN(10)
	filePath := path.Join(Path, name)

	i, err := newItem(filePath, l)
	if err != nil {
		t.Fatal(err)
	}
	data, err := randomBytesN(20)
	if err != nil {
		t.Fatal(err)
	}

	// Test
	got, err := writeAll(i.file, data)

	// Assert
	if err != nil {
		t.Fatal(err)
	} else if got != 20 {
		t.Fatalf("got %v, expected %v", got, 20)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	} else if string(fileContent) != string(data) {
		t.Fatalf("writeAll corrupted data: got %v, expected %v", fileContent, data)
	}

}

func TestReadFromStart(t *testing.T) {
	// Setup
	l := zerolog.Nop()
	name := newRandomStringN(10)
	filePath := path.Join(Path, name)

	i, err := newItem(filePath, l)
	if err != nil {
		t.Fatal(err)
	}
	data, err := randomBytesN(20)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filePath, data, fs.FileMode(os.O_APPEND))
	if err != nil {
		t.Fatal(err)
	}

	// Test
	got, err := readFromStart(i.file)
	if err != nil {
		t.Fatal(err)
	} else if string(got) != string(data) {
		t.Fatalf("got %v, expected %v", got, data)
	}
}

func TestFill(t *testing.T) {
	// Setup
	l := zerolog.Nop()
	name := newRandomStringN(10)
	filePath := path.Join(Path, name)

	i, err := newItem(filePath, l)
	if err != nil {
		t.Fatal(err)
	}
	data, err := randomBytesN(20)
	if err != nil {
		t.Fatal(err)
	}
	dataFunc := func() ([]byte, error) {
		return data, nil
	}

	// Test
	got, err := i.fill(l, dataFunc)

	// Assert
	if err != nil {
		t.Fatal(err)
	} else if got != 20 {
		t.Fatalf("got %v, expected %v", got, 20)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	} else if string(fileContent) != string(data) {
		t.Fatalf("fill corrupted data: got %v, expected %v", fileContent, data)
	}
}

func TestBytes(t *testing.T) {
	// Setup
	l := zerolog.Nop()
	name := newRandomStringN(10)
	filePath := path.Join(Path, name)

	i, err := newItem(filePath, l)
	if err != nil {
		t.Fatal(err)
	}

	data, err := randomBytesN(20)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filePath, data, fs.FileMode(os.O_APPEND))
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	} else if string(got) != string(data) {
		t.Fatalf("got %v, expected %v", got, data)
	}

	// Test
	got = i.bytes(l)

	// Assert
	if string(got) != string(data) {
		t.Fatalf("got %v, expected %v", got, data)
	}
}

func TestDrop(t *testing.T) {
	// Setup
	l := zerolog.Nop()
	name := newRandomStringN(10)
	filePath := path.Join(Path, name)

	i, err := newItem(filePath, l)
	if err != nil {
		t.Fatal(err)
	}

	data, err := randomBytesN(20)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filePath, data, fs.FileMode(os.O_APPEND))
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	} else if string(got) != string(data) {
		t.Fatalf("got %v, expected %v", got, data)
	}

	// Test
	i.drop(l)

	// Assert
	got, err = os.ReadFile(filePath)
	if err == nil {
		t.Fatalf("got %v, expected error", got)
	} else if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatal(err)
	}
}

func randomBytesN(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
