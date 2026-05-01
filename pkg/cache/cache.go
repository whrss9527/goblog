package cache

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/allegro/bigcache"

	"goblog/pkg/utils"
)

var BigCache *Cache

type Cache struct {
	BigCache *bigcache.BigCache
}

func (c Cache) Get(key string) (value any, err error) {
	valueBytes, err := c.BigCache.Get(key)
	if err != nil {
		return nil, err
	}
	value = utils.Deserialize(valueBytes)
	return value, nil
}

func (c Cache) Set(key string, value any) {
	valueBytes := utils.Serialize(value)
	err := c.BigCache.Set(key, valueBytes)
	if err != nil {
		slog.Error("bigcache set failed", "key", key, "err", err)
	}
}

func InitBigCacheConfig() {
	config := bigcache.Config{
		Shards:           1024,
		LifeWindow:       math.MaxInt16 * time.Hour,
		CleanWindow:      2 * time.Minute,
		MaxEntrySize:     500,
		HardMaxCacheSize: 0,
	}
	bigCache, err := bigcache.NewBigCache(config)
	if err != nil {
		panic(fmt.Errorf("init BigCache: %w", err))
	}
	BigCache = &Cache{
		BigCache: bigCache,
	}
	slog.Info("BigCache initialized")
}
