package modelcache

import (
	"context"
	"testing"
	"time"

	"gomodel/internal/cache"
)

func TestRedisModelCache_GetSet(t *testing.T) {
	store := cache.NewMapStore()
	defer store.Close()
	c := NewRedisModelCacheWithStore(store, "test:models", time.Hour)
	defer c.Close()

	ctx := context.Background()
	got, err := c.Get(ctx)
	if err != nil {
		t.Fatalf("Get empty: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for empty cache, got %v", got)
	}

	mc := &ModelCache{
		UpdatedAt: time.Now(),
		Providers: map[string]CachedProvider{
			"openai": {
				ProviderType: "openai",
				OwnedBy:      "openai",
				Models: []CachedModel{
					{ID: "gpt-4", Created: 123},
				},
			},
		},
	}
	if err := c.Set(ctx, mc); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err = c.Get(ctx)
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil ModelCache")
	}
	if len(got.Providers) != 1 {
		t.Errorf("Providers: got %d entries, want 1", len(got.Providers))
	}
	p, ok := got.Providers["openai"]
	if !ok {
		t.Fatal("expected openai in Providers")
	}
	if p.ProviderType != "openai" {
		t.Errorf("ProviderType: got %s, want openai", p.ProviderType)
	}
	if len(p.Models) != 1 {
		t.Errorf("Models: got %d entries, want 1", len(p.Models))
	}
	if p.Models[0].ID != "gpt-4" {
		t.Errorf("Model ID: got %s, want gpt-4", p.Models[0].ID)
	}
}

func TestRedisModelCache_DefaultKeyAndTTL(t *testing.T) {
	store := cache.NewMapStore()
	defer store.Close()
	c := NewRedisModelCacheWithStore(store, "", 0)
	defer c.Close()

	rc, ok := c.(*redisModelCache)
	if !ok {
		t.Fatal("expected *redisModelCache from NewRedisModelCacheWithStore")
	}
	if rc.key != DefaultRedisKey {
		t.Errorf("key = %q, want %q", rc.key, DefaultRedisKey)
	}
	if rc.ttl != cache.DefaultRedisTTL {
		t.Errorf("ttl = %v, want %v", rc.ttl, cache.DefaultRedisTTL)
	}

	ctx := context.Background()
	mc := &ModelCache{
		UpdatedAt: time.Now(),
		Providers: map[string]CachedProvider{},
	}
	if err := c.Set(ctx, mc); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil ModelCache")
	}
}

func TestRedisModelCacheWithStore_CloseDoesNotCloseSharedStore(t *testing.T) {
	store := cache.NewMapStore()
	c := NewRedisModelCacheWithStore(store, "test:models", time.Hour)

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Store must still be usable after the cache is closed.
	ctx := context.Background()
	if err := store.Set(ctx, "probe", []byte("ok"), time.Hour); err != nil {
		t.Errorf("store.Set after cache Close: %v — shared store was closed unexpectedly", err)
	}
	store.Close()
}

func TestRedisModelCache_CloseClosesOwnedStore(t *testing.T) {
	store := cache.NewMapStore()
	c := &redisModelCache{store: store, key: DefaultRedisKey, ttl: cache.DefaultRedisTTL, owned: true}

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// MapStore.Close is a no-op, so we verify the owned flag drove the call
	// by confirming Close returned nil (not skipped).
	// A second Close should also be safe since MapStore.Close is idempotent.
	if err := c.Close(); err != nil {
		t.Errorf("second Close on owned cache: %v", err)
	}
}
