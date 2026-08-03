//go:build redis

package redisdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/redis/go-redis/v9"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// Config for Redis VectorDB provider
type Config struct {
	// Addr like "localhost:6379"; if empty tries REDIS_URL env via go-redis
	Addr string
	// Password optional
	Password string
	// DB index
	DB int
	// CollectionName required logical collection
	CollectionName string
	// EmbeddingFunction optional for text query
	EmbeddingFunction vectordb.EmbeddingFunction
	// DistanceFunction default cosine
	DistanceFunction vectordb.DistanceFunction
	// KeyPrefix optional custom prefix
	KeyPrefix string
}

type RedisDB struct {
	client   *redis.Client
	prefix   string
	coll     string
	embedder vectordb.EmbeddingFunction
	distance vectordb.DistanceFunction
}

func New(cfg Config) (*RedisDB, error) {
	if cfg.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	if cfg.DistanceFunction == "" {
		cfg.DistanceFunction = vectordb.Cosine
	}
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB})
	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "vectordb"
	}
	return &RedisDB{client: rdb, coll: cfg.CollectionName, prefix: prefix, embedder: cfg.EmbeddingFunction, distance: cfg.DistanceFunction}, nil
}

func (r *RedisDB) keyDoc(id string) string { return fmt.Sprintf("%s:%s:doc:%s", r.prefix, r.coll, id) }
func (r *RedisDB) keyIdx() string          { return fmt.Sprintf("%s:%s:index", r.prefix, r.coll) }

func (r *RedisDB) CreateCollection(ctx context.Context, name string, _ map[string]interface{}) error {
	if name != "" {
		r.coll = name
	}
	// Mark index key
	return r.client.HSetNX(ctx, r.keyIdx(), "_created", "1").Err()
}

func (r *RedisDB) DeleteCollection(ctx context.Context, name string) error {
	if name != "" {
		r.coll = name
	}
	// Scan-delete all docs under prefix
	cursor := uint64(0)
	pattern := fmt.Sprintf("%s:%s:doc:*", r.prefix, r.coll)
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return r.client.Del(ctx, r.keyIdx()).Err()
}

func (r *RedisDB) Add(ctx context.Context, documents []vectordb.Document) error {
	if len(documents) == 0 {
		return nil
	}
	for _, d := range documents {
		b, _ := json.Marshal(d)
		if err := r.client.Set(ctx, r.keyDoc(d.ID), b, 0).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (r *RedisDB) Update(ctx context.Context, documents []vectordb.Document) error {
	return r.Add(ctx, documents)
}

func (r *RedisDB) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.keyDoc(id)
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisDB) Query(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	if r.embedder == nil {
		return nil, fmt.Errorf("embedding function required for text query")
	}
	emb, err := r.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, err
	}
	return r.QueryWithEmbedding(ctx, emb, limit, filter)
}

func (r *RedisDB) QueryWithEmbedding(ctx context.Context, embedding []float32, limit int, _ map[string]interface{}) ([]vectordb.SearchResult, error) {
	if len(embedding) == 0 {
		return nil, errors.New("embedding required")
	}
	// Fetch all docs for naive scoring
	cursor := uint64(0)
	pattern := fmt.Sprintf("%s:%s:doc:*", r.prefix, r.coll)
	results := make([]vectordb.SearchResult, 0)
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			raw, err := r.client.Get(ctx, k).Bytes()
			if err != nil {
				continue
			}
			var doc vectordb.Document
			if err := json.Unmarshal(raw, &doc); err != nil {
				continue
			}
			if len(doc.Embedding) == 0 {
				continue
			}
			score, dist := scoreVectors(embedding, doc.Embedding, r.distance)
			results = append(results, vectordb.SearchResult{ID: doc.ID, Content: doc.Content, Metadata: doc.Metadata, Score: float32(score), Distance: float32(dist)})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	// sort by distance asc (or score desc for cosine)
	sort.Slice(results, func(i, j int) bool {
		if r.distance == vectordb.Cosine || r.distance == vectordb.InnerProduct {
			return results[i].Score > results[j].Score
		}
		return results[i].Distance < results[j].Distance
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (r *RedisDB) Get(ctx context.Context, ids []string) ([]vectordb.Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = r.keyDoc(id)
	}
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	out := make([]vectordb.Document, 0, len(ids))
	for _, v := range vals {
		if v == nil {
			continue
		}
		var doc vectordb.Document
		if err := json.Unmarshal([]byte(v.(string)), &doc); err == nil {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (r *RedisDB) Count(ctx context.Context) (int, error) {
	cursor := uint64(0)
	pattern := fmt.Sprintf("%s:%s:doc:*", r.prefix, r.coll)
	total := 0
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return 0, err
		}
		total += len(keys)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return total, nil
}

func (r *RedisDB) Close() error { return r.client.Close() }

func scoreVectors(a, b []float32, dist vectordb.DistanceFunction) (score float64, distance float64) {
	// simple cosine / l2 / inner product
	n := min(len(a), len(b))
	if n == 0 {
		return 0, 0
	}
	var dot, na, nb float64
	var l2 float64
	for i := 0; i < n; i++ {
		va := float64(a[i])
		vb := float64(b[i])
		dot += va * vb
		na += va * va
		nb += vb * vb
		d := va - vb
		l2 += d * d
	}
	switch dist {
	case vectordb.InnerProduct:
		return dot, -dot
	case vectordb.L2:
		return -l2, l2
	default: // Cosine
		if na == 0 || nb == 0 {
			return 0, 1
		}
		c := dot / (sqrt(na) * sqrt(nb))
		return c, 1 - c
	}
}

func sqrt(x float64) float64 {
	// Newton's method for a few iterations is enough here
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 8; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
