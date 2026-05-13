package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"be-lonceng_unman/internal/model"
)

// CacheService menyediakan caching untuk KRS data
// Menggunakan in-memory cache dengan TTL
// TODO: Ganti dengan Redis jika diperlukan scaling

type CacheService struct {
	cache      map[string]cacheItem
	mu         sync.RWMutex
	defaultTTL time.Duration
	ctx        context.Context
}

type cacheItem struct {
	Value      []byte
	Expiration time.Time
}

// NewCacheService membuat instance CacheService baru dengan context untuk graceful shutdown
func NewCacheService(ctx context.Context) *CacheService {
	cache := &CacheService{
		cache:      make(map[string]cacheItem),
		defaultTTL: 24 * time.Hour, // Default TTL 24 jam
		ctx:        ctx,
	}

	// Jalankan cleanup goroutine
	go cache.cleanup()

	return cache
}

// Set menyimpan data ke cache dengan TTL
func (c *CacheService) Set(ctx context.Context, key string, value *model.KRSResponse, ttl time.Duration) error {
	if ttl < 0 {
		return fmt.Errorf("cache TTL cannot be negative")
	}
	if ttl == 0 {
		return fmt.Errorf("cache TTL cannot be zero")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = cacheItem{
		Value:      data,
		Expiration: time.Now().Add(ttl),
	}

	return nil
}

// Get mengambil data dari cache
func (c *CacheService) Get(ctx context.Context, key string) (*model.KRSResponse, error) {
	c.mu.RLock()
	item, exists := c.cache[key]
	c.mu.RUnlock()

	if !exists {
		return nil, errors.New("cache miss")
	}

	if time.Now().After(item.Expiration) {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
		return nil, errors.New("cache expired")
	}

	var result model.KRSResponse
	if err := json.Unmarshal(item.Value, &result); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}

	return &result, nil
}

// Delete menghapus data dari cache
func (c *CacheService) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)
	return nil
}

// cleanup menghapus item yang sudah expired secara berkala
func (c *CacheService) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Context cancelled, exit cleanup goroutine
			return
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for key, item := range c.cache {
				if now.After(item.Expiration) {
					delete(c.cache, key)
				}
			}
			c.mu.Unlock()
		}
	}
}
