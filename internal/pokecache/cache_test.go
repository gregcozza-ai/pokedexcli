package pokecache

import (
	"testing"
	"time"
)

func TestAddGet(t *testing.T) {
	const interval = 5 * time.Second
	cache := NewCache(interval)

	testKey := "https://example.com"
	testVal := []byte("testdata")

	cache.Add(testKey, testVal)
	val, ok := cache.Get(testKey)
	if !ok {
		t.Errorf("expected to find key")
	}
	if string(val) != string(testVal) {
		t.Errorf("expected to find value")
	}
}

func TestReapLoop(t *testing.T) {
	const baseTime = 5 * time.Millisecond
	const waitTime = baseTime + 5*time.Millisecond
	cache := NewCache(baseTime)

	testKey := "https://example.com"
	testVal := []byte("testdata")
	cache.Add(testKey, testVal)

	_, ok := cache.Get(testKey)
	if !ok {
		t.Errorf("expected to find key")
	}

	time.Sleep(waitTime)

	_, ok = cache.Get(testKey)
	if ok {
		t.Errorf("expected to not find key")
	}
}
