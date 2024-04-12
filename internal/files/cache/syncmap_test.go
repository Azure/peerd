// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package cache

import (
	"fmt"
	"sync"
	"testing"
)

func TestSyncMapAddEvict(t *testing.T) {
	sm := NewSyncMap(100)
	sm.evictionPercentage = 10
	var wg sync.WaitGroup
	addEntry := func(key string, value int) {
		sm.Set(key, value)
		wg.Done()
	}

	wg.Add(100)
	for i := 0; i < 100; i++ {
		go addEntry(fmt.Sprintf("%d", i), i)
	}
	wg.Wait()

	if mapLen := len(*sm.mapObj); mapLen != 100 {
		t.Fatalf("unexpected length of map after adding to capacity: %d", mapLen)
	}

	sm.Set("200", 200) //Now it's beyond the map capacity. 10% of entries will be evicted
	if mapLen := len(*sm.mapObj); mapLen != 91 {
		t.Fatalf("unexpected length of map after adding beyond capacity: %d", mapLen)
	}
}

func TestSyncMapAddDelete(t *testing.T) {
	sm := NewSyncMap(10)
	var wg sync.WaitGroup

	addEntry := func(key string, value int) {
		sm.Set(key, value)
		wg.Done()
	}

	deleteEntry := func(key string) {
		sm.Delete(key)
		wg.Done()
	}

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go addEntry(fmt.Sprintf("%d", i), i)
	}

	wg.Wait()

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go deleteEntry(fmt.Sprintf("%d", i))
	}

	wg.Wait()
	mapLen := len(*sm.mapObj)
	if mapLen != 0 {
		t.Fatalf("unexpected length of map: %d", mapLen)
	}
}

func TestSyncMapUpdate(t *testing.T) {
	sm := NewSyncMap(10)
	var wg sync.WaitGroup
	addEntry := func(key string, value int) {
		sm.Set(key, value)
		wg.Done()
	}

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go addEntry(fmt.Sprintf("%d", i), i)
	}
	wg.Wait()
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go addEntry(fmt.Sprintf("%d", i), i*2)
	}
	wg.Wait()

	entry0, ok0 := sm.Get("0")
	entry9, ok1 := sm.Get("9")
	if !ok0 || !ok1 {
		t.Fatalf("no matching items in map")
	}
	if entry0.(int) != 0 || entry9.(int) != 18 {
		t.Fatalf("value is not correct")
	}
}
