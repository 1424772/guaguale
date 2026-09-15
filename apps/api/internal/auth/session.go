package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/1424772/guaguale/apps/api/internal/domain"
)

const SessionDuration = 30 * 24 * time.Hour

func NewSession(userID uint64, now time.Time) (string, domain.Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", domain.Session{}, ErrTokenGeneration
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, domain.Session{
		TokenHash: HashToken(token),
		UserID:    userID,
		ExpiresAt: now.Add(SessionDuration),
	}, nil
}

func HashToken(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}
