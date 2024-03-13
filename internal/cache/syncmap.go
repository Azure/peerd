// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package cache

import (
	"sync"
)

const defaultEvictionPercentage int = 5 //The default eviction percentage used when map reaches its capacity at insertion

// SyncMap is a map with synchronized access support
type SyncMap struct {
	mapObj             *map[string]interface{}
	lock               *sync.RWMutex
	capacity           int
	evictionPercentage int
}

// Get retrieves the value associated with the given key from the SyncMap.
// It returns the value and a boolean indicating whether the key was found.
func (sm *SyncMap) Get(key string) (entry interface{}, ok bool) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	entry, ok = (*sm.mapObj)[key]
	return
}

// Set sets a new entry or updates an existing one.
// Set adds or updates an entry in the SyncMap with the specified key.
// If the key already exists in the map, the entry will be updated.
// If the key does not exist and the map is at capacity, some entries will be evicted first.
func (sm *SyncMap) Set(key string, entry interface{}) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if _, ok := (*sm.mapObj)[key]; !ok { //We will need to add an entry
		if numEntries := len(*sm.mapObj); numEntries >= sm.capacity { //exceeding capacity, remove evictionPercentage of the entries
			numToEvict := numEntries * sm.evictionPercentage / 100
			if numToEvict <= 1 { //We will evict one as the minimum
				numToEvict = 1
			}
			numEvicted := 0
			for k := range *sm.mapObj { // GO map iterator will randomize the order. We just delete the first in the iterator
				delete(*sm.mapObj, k)
				numEvicted++
				if numEvicted >= numToEvict {
					break
				}
			}
		}
	}

	(*sm.mapObj)[key] = entry
}

// Delete removes the entry with the specified key from the SyncMap.
// If the key does not exist, this method does nothing.
func (sm *SyncMap) Delete(key string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	delete(*sm.mapObj, key)
}

// MakeSyncMap creates a new SyncMap with the specified maximum number of entries.
// If the maximum number of entries is less than or equal to 0, it will be set to 1.
func MakeSyncMap(maxEntries int) *SyncMap {
	if maxEntries <= 0 {
		maxEntries = 1
	}
	return &SyncMap{mapObj: &map[string]interface{}{},
		lock:               &sync.RWMutex{},
		capacity:           maxEntries,
		evictionPercentage: defaultEvictionPercentage}
}
