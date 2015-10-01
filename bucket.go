package main

import (
	"time"
	"log"
	"errors"
	"sync"
)


type StampyBucket struct {

	keyValueCache     map[string]StampyBucketEntry
	ttlIndex          map[string]bool

	stampyBucketStats *StampyBucketStats

	cacheMutex        sync.RWMutex
	ttlCacheMutex     sync.RWMutex
}


func (s *StampyBucket) putKeyWithValue(key string, value string, validUntil time.Time) {

	now := time.Now()
	var cacheEntry StampyBucketEntry

	cacheEntry.EntryValue = value
	cacheEntry.CreationDate = now
	cacheEntry.LastAccessed = now

	log.Println("Putting/Updating entry for key:", key, "with value:", value)

	if !validUntil.IsZero() {

		log.Println("TTL value has been provided, valid until: ", validUntil)

		s.ttlCacheMutex.Lock()

		cacheEntry.ValidUntil = validUntil
		s.keyValueCache[key] = cacheEntry
		s.ttlIndex[key] = true

		s.ttlCacheMutex.Unlock()
		return
	}

	s.cacheMutex.Lock()

	s.keyValueCache[key] = cacheEntry
	s.stampyBucketStats.KeyPuts++

	s.cacheMutex.Unlock()
}

func (s *StampyBucket) getValueWithKey(key string) (StampyBucketEntry, error) {

	log.Println("Fetching entry with key:", key)

	now := time.Now()

	if s.ttlIndex[key] {
		// key with ttl set
		s.ttlCacheMutex.RLock()
		value, ok := s.keyValueCache[key]

		if ok {

			if value.ValidUntil.Before(now) {
				// key valid but expired
				s.stampyBucketStats.incrementExpiredKeyHits()
				log.Println("Entry with key:", key, "has been expired.")

				defer s.deleteValueWithKeyIfPresent(key)
				return value, errors.New("Expired key")
			}

			// key valid
			s.stampyBucketStats.incrementKeyHits()
			value.LastAccessed = now
			return value, nil
		}

		// missing key
		return value, errors.New("Missing key")

	}

	s.cacheMutex.RLock()

	value, ok := s.keyValueCache[key]

	if ok {
		s.stampyBucketStats.incrementKeyHits()
	} else {
		s.stampyBucketStats.incrementAbsentKeyHits()
	}

	s.cacheMutex.RUnlock()

	if !ok {
		log.Println("Entry with key:", key, "not found.")
		log.Println(s.stampyBucketStats.AbsentKeyHits)
		return value, errors.New("Missing key")
	}

	log.Printf("Entry with key:%s found, %b", key, value)

	return value, nil

}

func (s *StampyBucket) deleteValueWithKeyIfPresent(key string) {

	_, ok := s.keyValueCache[key]

	if ok {
		s.cacheMutex.Lock()

		delete(s.keyValueCache, key)
		s.stampyBucketStats.incrementKeyDeletes()
		log.Println("Deleted key from cache:", key)

		s.cacheMutex.Unlock()
	}


}
type StampyBucketEntry struct {
	EntryValue   string `json:"value"`
	CreationDate time.Time `json:"creationDate"`
	LastAccessed time.Time `json:"lastAccessed"`
	ValidUntil   time.Time `json:"validUntil"`
}

type StampyBucketStats struct {
	KeyPuts        uint64 `json:"keyPuts"`
	KeyDeletes     uint64 `json:"keyDeletes"`
	KeyHits        uint64 `json:"keyHits"`
	AbsentKeyHits  uint64 `json:"absentKeyHits"`
	ExpiredKeys    uint64 `json:"expiredKeys"`
	ExpiredKeyHits uint64 `json:"expiredKeyHits"`
}

func (stats *StampyBucketStats) incrementKeyPuts() {
	stats.KeyPuts += 1
}

func (stats *StampyBucketStats) incrementKeyDeletes() {
	stats.KeyDeletes += 1
}
func (stats *StampyBucketStats) incrementKeyHits() {
	stats.KeyHits += 1
}
func (stats *StampyBucketStats) incrementAbsentKeyHits() {
	stats.AbsentKeyHits += 1
}
func (stats *StampyBucketStats) incrementExpiredKeys() {
	stats.ExpiredKeys += 1
}
func (stats *StampyBucketStats) incrementExpiredKeyHits() {
	stats.ExpiredKeyHits += 1
}

