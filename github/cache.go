package github

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/joshmedeski/sesh/v2/home"
	"github.com/joshmedeski/sesh/v2/model"
)

type Cache interface {
	Get(org string) ([]model.GitHubRepo, bool)
	Set(org string, repos []model.GitHubRepo, timeout int)
	GetCachePath() string
}

type RealCache struct {
	home home.Home
}

func NewCache(home home.Home) Cache {
	return &RealCache{
		home: home,
	}
}

func (c *RealCache) Get(org string) ([]model.GitHubRepo, bool) {
	cachePath := c.getCacheFilePath(org)
	
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, false
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		slog.Warn("Failed to read cache file", "path", cachePath, "error", err)
		return nil, false
	}

	var cache model.GitHubCache
	if err := json.Unmarshal(data, &cache); err != nil {
		slog.Warn("Failed to unmarshal cache", "error", err)
		return nil, false
	}

	if time.Now().After(cache.ExpiresAt) {
		slog.Debug("Cache expired", "expired_at", cache.ExpiresAt)
		return nil, false
	}

	slog.Debug("Cache hit", "org", org, "repos_count", len(cache.Repos))
	return cache.Repos, true
}

func (c *RealCache) Set(org string, repos []model.GitHubRepo, timeout int) {
	cachePath := c.getCacheFilePath(org)
	
	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		slog.Error("Failed to create cache directory", "error", err)
		return
	}

	now := time.Now()
	cache := model.GitHubCache{
		Repos:     repos,
		CachedAt:  now,
		ExpiresAt: now.Add(time.Duration(timeout) * time.Minute),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal cache", "error", err)
		return
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		slog.Error("Failed to write cache file", "path", cachePath, "error", err)
		return
	}

	slog.Debug("Cache updated", "org", org, "repos_count", len(repos), "expires_at", cache.ExpiresAt)
}

func (c *RealCache) GetCachePath() string {
	return c.getCacheFilePath("")
}

func (c *RealCache) getCacheFilePath(org string) string {
	// Get home directory using os package since Home interface doesn't expose HomeDir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Failed to get home directory", "error", err)
		return ""
	}
	if org == "" {
		return filepath.Join(homeDir, ".cache", "sesh", "github")
	}
	return filepath.Join(homeDir, ".cache", "sesh", "github", org+".json")
}
