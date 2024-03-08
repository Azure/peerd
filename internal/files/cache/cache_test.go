package cache

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/azure/peerd/pkg/math"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

func TestGetKey(t *testing.T) {
	name := newRandomStringN(10)
	offset := int64(100)
	c := New(context.Background())
	got := c.(*fileCache).getKey(name, offset)
	want := fmt.Sprintf("%v/%v/%v", Path, name, offset)
	if got != want {
		t.Errorf("expected: %v, got: %v", want, got)
	}
}

func TestExists(t *testing.T) {
	c := New(context.Background())

	filesThatExist := []string{}
	for i := 0; i < 5; i++ {
		filename := newRandomStringN(10)
		off := int64(100*i + 1) // 1, 101, 201, 301, 401
		filesThatExist = append(filesThatExist, fmt.Sprintf("%v_%v", filename, off))
		//nolint:errcheck
		c.GetOrCreate(filename, off, 1024, func() ([]byte, error) {
			return []byte(newRandomString()), nil
		})
	}

	filesThatExistAndNotFilled := []string{}
	for i := 0; i < 5; i++ {
		fileName := strings.Split(filesThatExist[i], "_")[0]
		off := int64(10*(i+1) + 1) // 11, 21, 31, 41, 51
		filesThatExistAndNotFilled = append(filesThatExistAndNotFilled, fmt.Sprintf("%v_%v", fileName, off))
		key := c.(*fileCache).getKey(fileName, off)
		val, err := newItem(key, c.(*fileCache).log)
		if err != nil {
			t.Fatal(err)
		}
		c.(*fileCache).fileCache.Set(key, val, 0)
	}

	filesThatDoNotExist := []string{}
	for i := 0; i < 5; i++ {
		filename := newRandomStringN(10)
		off := int64(100*i + 1) // 1, 101, 201, 301, 401
		filesThatDoNotExist = append(filesThatDoNotExist, fmt.Sprintf("%v_%v", filename, off))
	}

	type tc struct {
		name     string
		filename string
		offset   int64
		want     bool
	}

	tcs := []tc{}

	for i := 0; i < 15; i++ {
		var filename, name, offset string
		var want bool

		if i < 5 {
			name = "exists"
			filename = strings.Split(filesThatExist[i], "_")[0]
			offset = strings.Split(filesThatExist[i], "_")[1]
			want = true
		} else if i < 10 {
			name = "exists-not-filled"
			filename = strings.Split(filesThatExistAndNotFilled[i-5], "_")[0]
			offset = strings.Split(filesThatExistAndNotFilled[i-5], "_")[1]
			want = false
		} else {
			name = "does-not-exist"
			filename = strings.Split(filesThatDoNotExist[i-10], "_")[0]
			offset = strings.Split(filesThatDoNotExist[i-10], "_")[1]
			want = false
		}

		o, _ := strconv.Atoi(offset)
		tcs = append(tcs, tc{
			name:     name,
			filename: filename,
			offset:   int64(o),
			want:     want,
		})
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := c.Exists(tc.filename, tc.offset)
			if got != tc.want {
				t.Errorf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestPutAndGetSize(t *testing.T) {
	c := New(context.Background())
	var eg errgroup.Group

	for i := 0; i < 1000; i++ {
		eg.Go(func() error {
			filename := newRandomStringN(10)
			size := rand.Int63n(1024)

			_, ok := c.Size(filename)
			if ok == true {
				return fmt.Errorf("expected false, got %v", ok)
			}

			ok = c.PutSize(filename, size)
			if ok != true {
				return fmt.Errorf("expected true, got %v", ok)
			}

			val, ok := c.Size(filename)
			if !ok {
				return fmt.Errorf("file size: expected true, got %v", ok)
			}
			if val != size {
				return fmt.Errorf("expected %v, got %v", size, val)
			}
			return nil
		})
	}
	err := eg.Wait()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetOrCreate(t *testing.T) {
	zerolog.TimeFieldFormat = time.RFC3339
	//c := New(zerolog.New(os.Stdout).With().Timestamp().Logger().WithContext(context.Background()))
	c := New(context.Background())
	var eg errgroup.Group

	fileNames := new(sync.Map)
	fileContents := new(sync.Map)
	fileSizes := new(sync.Map)

	for i := 0; i < 100; i++ {

		fileNames.Store(i, newRandomStringN(10))
		b := []byte(newRandomString())
		size := int64(len(b))
		fileContents.Store(i, b)
		fileSizes.Store(i, size)

		for j := 0; j < 10; j++ {
			segs, err := math.NewSegments(0, 1024*1024, size, size)
			if err != nil {
				t.Fatal(err)
			}
			for seg := range segs.All() {

				s := seg
				fileIndex := i

				eg.Go(func() error {
					offset := s.Index
					count := s.Count
					fc, ok := fileContents.Load(fileIndex)
					if !ok {
						t.Fatalf("could not load fileContent from sync map: %v", fileIndex)
					}
					fcBytes, _ := fc.([]byte)
					expected := make([]byte, count)
					copy(expected, fcBytes[offset:offset+int64(count)])

					fn, ok := fileNames.Load(fileIndex)
					if !ok {
						t.Fatalf("could not load fileName from sync map: %v", fileIndex)
					}
					name, _ := fn.(string)

					got, err := c.GetOrCreate(name, offset, count, func() ([]byte, error) {
						return expected, nil
					})
					if err != nil {
						return fmt.Errorf("failed to get or create: %v -- %v", name, err)
					}

					l := len(got)
					if count != l {
						return fmt.Errorf("size mismatch, expected %v, got %v, offset: %v, fileName: %v", count, len(got), offset, name)
					}

					validationLen := math.Min(100, count)
					if !bytes.Equal(expected[:validationLen], got[:validationLen]) {
						return fmt.Errorf("leading bytes mismatch, expected: %v, got: %v", expected[:validationLen], got[:validationLen])
					}

					if !bytes.Equal(expected[l-validationLen:], got[l-validationLen:]) {
						return fmt.Errorf("ending bytes mismatch, expected: %v, got: %v", expected[l-validationLen:], got[l-validationLen:])
					}

					return nil
				})
			}
		}
	}

	err := eg.Wait()
	if err != nil {
		t.Fatal(err)
	}
}
