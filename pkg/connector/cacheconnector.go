package connector

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"docupdatesexecutor/pkg/structures"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var ErrCacheMiss = errors.New("cache miss")

type CacheConnector interface {
	Get(ctx context.Context, key string) (*structures.TDocument, error)
	Set(ctx context.Context, key string, doc *structures.TDocument, ttl time.Duration) error
	StartBackgroundSync(ctx context.Context, dbConnector DBConnector, syncInterval time.Duration)
}

type RedisCacheConnector struct {
	client *redis.Client
}

func NewCacheConnector(config *structures.RedisConfig) CacheConnector {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Username: config.User,
		Password: config.Password,
		DB:       config.DB,
	})
	ctx := context.Background()
	client.ConfigSet(ctx, "maxmemory", config.MaxMemory)
	client.ConfigSet(ctx, "maxmemory-policy", config.MaxMemoryPolicy)
	client.ConfigSet(ctx, "lazyfree-lazy-eviction", config.LazyfreeLazyEviction)

	return &RedisCacheConnector{client: client}
}

func (r *RedisCacheConnector) Get(ctx context.Context, key string) (*structures.TDocument, error) {
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrCacheMiss
	}

	doc := &structures.TDocument{
		Url:  data["Url"],
		Text: data["Text"],
	}
	doc.PubDate, _ = strconv.ParseUint(data["PubDate"], 10, 64)
	doc.FetchTime, _ = strconv.ParseUint(data["FetchTime"], 10, 64)
	doc.FirstFetchTime, _ = strconv.ParseUint(data["FirstFetchTime"], 10, 64)

	return doc, nil
}

func (r *RedisCacheConnector) Set(ctx context.Context, key string, doc *structures.TDocument, ttl time.Duration) error {
	data := map[string]interface{}{
		"Url":            doc.Url,
		"Text":           doc.Text,
		"PubDate":        strconv.FormatUint(doc.PubDate, 10),
		"FetchTime":      strconv.FormatUint(doc.FetchTime, 10),
		"FirstFetchTime": strconv.FormatUint(doc.FirstFetchTime, 10),
	}

	_, err := r.client.HSet(ctx, key, data).Result()
	if err != nil {
		return err
	}

	return r.client.Expire(ctx, key, ttl).Err()
}

func (r *RedisCacheConnector) StartBackgroundSync(ctx context.Context, dbConnector DBConnector, syncInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.syncCacheWithDB(ctx, dbConnector)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (r *RedisCacheConnector) syncCacheWithDB(ctx context.Context, dbConnector DBConnector) {
	keys, err := r.client.Keys(ctx, "doc:*").Result()
	if err != nil {
		log.Printf("Failed to get keys from Redis: %v", err)
		return
	}

	for _, key := range keys {
		doc, err := r.Get(ctx, key)
		if err != nil {
			log.Printf("Failed to get document from Redis for key %s: %v", key, err)
			continue
		}

		existingDoc, err := dbConnector.GetDocument(doc.Url)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := dbConnector.AddDocument(doc); err != nil {
					log.Printf("Failed to add document to DB: %v", err)
				}
			} else {
				log.Printf("Failed to get document from DB: %v", err)
			}
		} else {
			updateDocumentFields(existingDoc, doc)
			if err := dbConnector.UpdateDocument(existingDoc); err != nil {
				log.Printf("Failed to update document in DB: %v", err)
			}
		}
	}
}

func updateDocumentFields(existingDoc, newDoc *structures.TDocument) {
	if newDoc.FetchTime > existingDoc.FetchTime {
		existingDoc.Text = newDoc.Text
		existingDoc.FetchTime = newDoc.FetchTime
	}
	if newDoc.FetchTime < existingDoc.FetchTime {
		existingDoc.PubDate = newDoc.PubDate
		existingDoc.FirstFetchTime = newDoc.FetchTime
	}
}
