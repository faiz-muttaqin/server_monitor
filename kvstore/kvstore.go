package kvstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

var (
	RDB       *redis.Client
	redisUp   atomic.Bool
	shardMaps [256]*sync.Map
	cachePath = ".cache/kvstore.json"
	cacheMux  sync.RWMutex // Mutex for cache file operations
)

type valueWithTTL struct {
	value string
	ttl   time.Time
}

// CacheEntry represents a key-value pair for JSON serialization
type CacheEntry struct {
	Key   string    `json:"key"`
	Value string    `json:"value"`
	TTL   time.Time `json:"ttl"`
}

// CacheData represents the entire cache for JSON serialization
type CacheData struct {
	Entries []CacheEntry `json:"entries"`
	SavedAt time.Time    `json:"saved_at"`
}

func init() {
	for i := range shardMaps {
		shardMaps[i] = &sync.Map{}
	}
	redisUp.Store(false) // Initialize redisUp to false

	// Load cache from file if exists
	loadCacheFromFile()
}

// InitRedis initializes and returns a Redis client
func InitRedis(addr, password string, db int) {
	// Create a new Redis client
	RDB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Ping the Redis server to check the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RDB.Ping(ctx).Result()
	if err != nil {
		logrus.Error(err)
		redisUp.Store(false)
		fmt.Printf("failed to connect to Redis: %v", err)
	}

	redisUp.Store(true)
	fmt.Println("Connected to the Redis server successfully")
}

func getShard(key string) *sync.Map {
	return shardMaps[uint(fnv32(key))%uint(len(shardMaps))]
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// loadCacheFromFile loads the cache from the JSON file
func loadCacheFromFile() {
	cacheMux.Lock()
	defer cacheMux.Unlock()

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		logrus.Info("Cache file does not exist, starting with empty cache")
		return
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		logrus.Errorf("Failed to read cache file: %v", err)
		return
	}

	// Parse JSON
	var cacheData CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		logrus.Errorf("Failed to parse cache file: %v", err)
		return
	}

	// Load entries into shardMaps, filtering out expired ones
	now := time.Now()
	validEntries := 0
	expiredEntries := 0

	for _, entry := range cacheData.Entries {
		if now.Before(entry.TTL) {
			shard := getShard(entry.Key)
			shard.Store(entry.Key, valueWithTTL{
				value: entry.Value,
				ttl:   entry.TTL,
			})

			// Schedule cleanup for this entry
			remainingTTL := time.Until(entry.TTL)
			time.AfterFunc(remainingTTL, func() {
				shard.Delete(entry.Key)
				saveCacheToFile() // Save after cleanup
			})
			validEntries++
		} else {
			expiredEntries++
		}
	}

	logrus.Infof("Loaded cache from file: %d valid entries, %d expired entries (saved at: %s)",
		validEntries, expiredEntries, cacheData.SavedAt.Format("2006-01-02 15:04:05"))
}

// saveCacheToFile saves the current cache to the JSON file
func saveCacheToFile() {
	// Only save if Redis is down (using local cache)
	if redisUp.Load() {
		return
	}

	cacheMux.Lock()
	defer cacheMux.Unlock()

	// Create cache directory if it doesn't exist
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logrus.Errorf("Failed to create cache directory: %v", err)
		return
	}

	// Collect all entries from shardMaps
	var entries []CacheEntry
	now := time.Now()

	for _, shard := range shardMaps {
		shard.Range(func(key, value interface{}) bool {
			if k, ok := key.(string); ok {
				if v, ok := value.(valueWithTTL); ok {
					// Only save non-expired entries
					if now.Before(v.ttl) {
						entries = append(entries, CacheEntry{
							Key:   k,
							Value: v.value,
							TTL:   v.ttl,
						})
					}
				}
			}
			return true
		})
	}

	// Create cache data structure
	cacheData := CacheData{
		Entries: entries,
		SavedAt: now,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		logrus.Errorf("Failed to marshal cache data: %v", err)
		return
	}

	// Write to temporary file first, then rename (atomic operation)
	tempPath := cachePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		logrus.Errorf("Failed to write cache file: %v", err)
		return
	}

	// Atomic rename
	if err := os.Rename(tempPath, cachePath); err != nil {
		logrus.Errorf("Failed to rename cache file: %v", err)
		os.Remove(tempPath) // Cleanup temp file
		return
	}

	logrus.Debugf("Cache saved to file: %d entries", len(entries))
}

// SetKey sets a key with a value and an expiration time
func SetKey(key string, value string, ttl time.Duration) error {
	if redisUp.Load() {
		err := RDB.Set(context.Background(), key, value, ttl).Err()
		if err == nil {
			return nil
		}
		redisUp.Store(false)
	}

	shard := getShard(key)
	expiration := time.Now().Add(ttl)
	shard.Store(key, valueWithTTL{value: value, ttl: expiration})

	// Schedule cleanup
	time.AfterFunc(ttl, func() {
		shard.Delete(key)
		saveCacheToFile() // Save after cleanup
	})

	// Save to cache file
	go saveCacheToFile()

	return nil
}

// GetKey retrieves a value by key
func GetKey(key string) (string, error) {
	if redisUp.Load() {
		val, err := RDB.Get(context.Background(), key).Result()
		if err == nil {
			return val, nil
		}
		redisUp.Store(false)
	}

	shard := getShard(key)
	if val, ok := shard.Load(key); ok {
		v := val.(valueWithTTL)
		if time.Now().Before(v.ttl) {
			return v.value, nil
		}
		shard.Delete(key)
	}
	return "", fmt.Errorf("key not found")
}

// ExistsIn checks if a key exists
func ExistsIn(key string) (bool, error) {
	if redisUp.Load() {
		count, err := RDB.Exists(context.Background(), key).Result()
		if err == nil {
			return count > 0, nil
		}
		redisUp.Store(false)
	}

	shard := getShard(key)
	if val, ok := shard.Load(key); ok {
		v := val.(valueWithTTL)
		if time.Now().Before(v.ttl) {
			return true, nil
		}
		shard.Delete(key)
	}
	return false, nil
}

// ExtendKeyTTL extends the expiration of a key
func ExtendKeyTTL(key string, ttl time.Duration) error {
	if redisUp.Load() {
		err := RDB.Expire(context.Background(), key, ttl).Err()
		if err == nil {
			return nil
		}
		redisUp.Store(false)
	}

	shard := getShard(key)
	if val, ok := shard.Load(key); ok {
		v := val.(valueWithTTL)
		v.ttl = time.Now().Add(ttl)
		shard.Store(key, v)

		// Schedule cleanup
		time.AfterFunc(ttl, func() {
			shard.Delete(key)
			saveCacheToFile() // Save after cleanup
		})

		// Save to cache file
		go saveCacheToFile()

		return nil
	}
	return fmt.Errorf("key not found")
}

// DeleteKey removes a key
func DeleteKey(key string) error {
	if redisUp.Load() {
		err := RDB.Del(context.Background(), key).Err()
		if err == nil {
			return nil
		}
		redisUp.Store(false)
	}

	shard := getShard(key)
	shard.Delete(key)

	// Save to cache file
	go saveCacheToFile()

	return nil
}

// DeleteKeysWithPrefix removes all keys that start with the given prefix
func DeleteKeysWithPrefix(prefix string) error {
	if redisUp.Load() {
		ctx := context.Background()
		iter := RDB.Scan(ctx, 0, prefix+"*", 0).Iterator()
		for iter.Next(ctx) {
			err := RDB.Del(ctx, iter.Val()).Err()
			if err != nil {
				logrus.Error(err)
				redisUp.Store(false)
				break
			}
		}
		if err := iter.Err(); err == nil {
			return nil
		}
	}

	// If Redis is down, delete from local cache
	deletedCount := 0
	for _, shard := range shardMaps {
		shard.Range(func(key, value interface{}) bool {
			if k, ok := key.(string); ok && len(k) >= len(prefix) && k[:len(prefix)] == prefix {
				shard.Delete(key)
				deletedCount++
			}
			return true
		})
	}

	// Save to cache file if any keys were deleted
	if deletedCount > 0 {
		go saveCacheToFile()
		logrus.Debugf("Deleted %d keys with prefix '%s' from local cache", deletedCount, prefix)
	}

	return nil
}

// GetKeyTTL retrieves the remaining expiration time of a key
func GetKeyTTL(key string) (time.Duration, error) {
	if redisUp.Load() {
		ttl, err := RDB.TTL(context.Background(), key).Result()
		if err == nil {
			return ttl, nil
		}
		redisUp.Store(false)
	}

	shard := getShard(key)
	if val, ok := shard.Load(key); ok {
		v := val.(valueWithTTL)
		if time.Now().Before(v.ttl) {
			return time.Until(v.ttl), nil
		}
		shard.Delete(key)
	}
	return 0, fmt.Errorf("key not found")
}
