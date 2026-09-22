//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestForkUpdateCacheIsolation(t *testing.T) {
	cache := &updateServiceCacheStub{}
	s := NewUpdateService(cache, nil, "0.2.7-fix1", "release")
	s.saveToCache(context.Background(), &UpdateInfo{LatestVersion: "0.2.7-fix2"})
	cached, err := s.getFromCache(context.Background())
	require.NoError(t, err)
	require.True(t, cached.HasUpdate)

	raw, err := json.Marshal(map[string]any{
		"repository": "unrelated/repository",
		"latest":     "99.0.0",
		"timestamp":  time.Now().Unix(),
	})
	require.NoError(t, err)
	cache.data = string(raw)
	_, err = s.getFromCache(context.Background())
	require.ErrorContains(t, err, "different repository")

	raw, err = json.Marshal(map[string]any{"latest": "99.0.0", "timestamp": time.Now().Unix()})
	require.NoError(t, err)
	cache.data = string(raw)
	_, err = s.getFromCache(context.Background())
	if githubRepo == "Wei-Shaw/sub2api" {
		require.NoError(t, err, "legacy official caches remain valid for upstream builds")
	} else {
		require.ErrorContains(t, err, "different repository")
	}
}
