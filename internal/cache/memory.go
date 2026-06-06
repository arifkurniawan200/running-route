package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/arifkurniawan200/running-route/config"
)

type entry struct {
	value  interface{}
	expiry time.Time
}

type Memory struct {
	mu     sync.RWMutex
	data   map[string]*entry
	stopCh chan struct{}
	stats  Stats
}

type Stats struct {
	Entries    int
	Hits       int64
	Misses     int64
	DefaultTTL time.Duration
}

func NewMemory(cfg *config.Config) *Memory {
	m := &Memory{
		data:   make(map[string]*entry),
		stopCh: make(chan struct{}),
	}
	m.stats.DefaultTTL = cfg.CacheTTL
	go m.evictionLoop()
	return m
}

func (m *Memory) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	e, ok := m.data[key]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		m.stats.Misses++
		m.mu.Unlock()
		return nil, false
	}

	if time.Now().After(e.expiry) {
		m.mu.Lock()
		delete(m.data, key)
		m.stats.Misses++
		m.mu.Unlock()
		return nil, false
	}

	m.mu.Lock()
	m.stats.Hits++
	m.mu.Unlock()
	return e.value, true
}

func (m *Memory) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = &entry{
		value:  value,
		expiry: time.Now().Add(ttl),
	}
}

func (m *Memory) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *Memory) Stats() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return fmt.Sprintf("memory cache: %d entries, %d hits, %d misses, default TTL=%s",
		len(m.data), m.stats.Hits, m.stats.Misses, m.stats.DefaultTTL)
}

func (m *Memory) Stop() {
	close(m.stopCh)
}

func (m *Memory) evictionLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.evict()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Memory) evict() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, e := range m.data {
		if now.After(e.expiry) {
			delete(m.data, k)
		}
	}
}
