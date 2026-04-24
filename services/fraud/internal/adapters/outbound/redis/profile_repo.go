package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/DeSouzaRafael/go-fintech-microservices/pkg/errors"
	"github.com/DeSouzaRafael/go-fintech-microservices/services/fraud/internal/domain"
)

const profileTTL = 25 * time.Hour

type ProfileRepository struct {
	client *redis.Client
}

func NewProfileRepository(client *redis.Client) *ProfileRepository {
	return &ProfileRepository{client: client}
}

func (r *ProfileRepository) GetProfile(ctx context.Context, userID uuid.UUID) (domain.UserProfile, error) {
	key := profileKey(userID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return domain.UserProfile{}, errors.New(errors.CodeNotFound, "profile not found")
	}
	if err != nil {
		return domain.UserProfile{}, errors.Wrap(errors.CodeInternal, "get profile", err)
	}

	var profile domain.UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return domain.UserProfile{}, errors.Wrap(errors.CodeInternal, "unmarshal profile", err)
	}
	return profile, nil
}

func (r *ProfileRepository) UpdateProfile(ctx context.Context, profile domain.UserProfile) error {
	data, err := json.Marshal(profile)
	if err != nil {
		return errors.Wrap(errors.CodeInternal, "marshal profile", err)
	}

	if err := r.client.Set(ctx, profileKey(profile.UserID), data, profileTTL).Err(); err != nil {
		return errors.Wrap(errors.CodeInternal, "set profile", err)
	}
	return nil
}

func profileKey(userID uuid.UUID) string {
	return fmt.Sprintf("fraud:profile:%s", userID.String())
}
