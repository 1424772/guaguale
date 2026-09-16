package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

const (
	leaderboardLimit = 100
	historyLimit     = 100
)

type LeaderboardCache interface {
	Get(context.Context) ([]domain.RankedUser, time.Time, bool, error)
	Set(context.Context, []domain.RankedUser, time.Time, time.Duration) error
}

func (service *Service) Leaderboard(ctx context.Context, user domain.User) (domain.Leaderboard, error) {
	generatedAt := service.now().UTC()
	cached := false
	var entries []domain.RankedUser
	if service.leaderboardCache != nil {
		if snapshot, snapshotAt, found, err := service.leaderboardCache.Get(ctx); err == nil && found {
			entries = snapshot
			generatedAt = snapshotAt
			cached = true
		}
	}
	if entries == nil {
		var err error
		entries, err = service.store.TopPlayers(ctx, leaderboardLimit)
		if err != nil {
			return domain.Leaderboard{}, err
		}
		if service.leaderboardCache != nil {
			_ = service.leaderboardCache.Set(ctx, entries, generatedAt, 8*time.Second)
		}
	}
	current, total, err := service.store.PlayerRank(ctx, user.ID)
	if err != nil {
		return domain.Leaderboard{}, err
	}
	publicEntries := make([]domain.RankedUser, len(entries))
	for index, entry := range entries {
		entry.IsCurrent = entry.UserID == user.ID
		entry.Username = maskUsername(entry.Username)
		publicEntries[index] = entry
	}
	current.IsCurrent = true
	current.Username = maskUsername(current.Username)
	return domain.Leaderboard{
		Entries: publicEntries, CurrentUser: current, TotalUsers: total, GeneratedAt: generatedAt, Cached: cached,
	}, nil
}

func (service *Service) GameHistory(ctx context.Context, userID uint64) ([]domain.HistoryEvent, error) {
	return service.store.GameHistory(ctx, userID, historyLimit)
}

func maskUsername(username string) string {
	characters := []rune(username)
	if len(characters) <= 1 {
		return "*"
	}
	if len(characters) == 2 {
		return string(characters[0]) + "*"
	}
	maskedLength := utf8.RuneCountInString(username) - 2
	return string(characters[0]) + strings.Repeat("*", maskedLength) + string(characters[len(characters)-1])
}
