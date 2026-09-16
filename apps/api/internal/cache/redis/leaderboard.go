package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	redisclient "github.com/redis/go-redis/v9"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

const leaderboardKey = "guaguale:leaderboard:top100:v1"

type LeaderboardCache struct {
	client *redisclient.Client
}

type snapshot struct {
	Entries     []snapshotEntry `json:"entries"`
	GeneratedAt time.Time       `json:"generatedAt"`
}

type snapshotEntry struct {
	Rank     int    `json:"rank"`
	UserID   uint64 `json:"userId"`
	Username string `json:"username"`
	Balance  int64  `json:"balance"`
}

func New(address, password string) *LeaderboardCache {
	return &LeaderboardCache{client: redisclient.NewClient(&redisclient.Options{
		Addr: address, Password: password, DB: 0,
		DialTimeout: 2 * time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second,
	})}
}

func (cache *LeaderboardCache) Ping(ctx context.Context) error {
	return cache.client.Ping(ctx).Err()
}

func (cache *LeaderboardCache) Close() error {
	return cache.client.Close()
}

func (cache *LeaderboardCache) Get(ctx context.Context) ([]domain.RankedUser, time.Time, bool, error) {
	value, err := cache.client.Get(ctx, leaderboardKey).Bytes()
	if errors.Is(err, redisclient.Nil) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, err
	}
	var cached snapshot
	if err := json.Unmarshal(value, &cached); err != nil {
		return nil, time.Time{}, false, err
	}
	entries := make([]domain.RankedUser, len(cached.Entries))
	for index, entry := range cached.Entries {
		entries[index] = domain.RankedUser{Rank: entry.Rank, UserID: entry.UserID, Username: entry.Username, Balance: entry.Balance}
	}
	return entries, cached.GeneratedAt, true, nil
}

func (cache *LeaderboardCache) Set(ctx context.Context, entries []domain.RankedUser, generatedAt time.Time, ttl time.Duration) error {
	cachedEntries := make([]snapshotEntry, len(entries))
	for index, entry := range entries {
		cachedEntries[index] = snapshotEntry{Rank: entry.Rank, UserID: entry.UserID, Username: entry.Username, Balance: entry.Balance}
	}
	value, err := json.Marshal(snapshot{Entries: cachedEntries, GeneratedAt: generatedAt})
	if err != nil {
		return err
	}
	return cache.client.Set(ctx, leaderboardKey, value, ttl).Err()
}
