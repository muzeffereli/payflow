package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"payment-platform/internal/cart/domain"
	"payment-platform/internal/cart/port"
)

var _ port.CartRepository = (*redisRepo)(nil)

const cartTTL = 24 * time.Hour

type redisRepo struct {
	rdb *redis.Client
}

func NewRedisRepo(rdb *redis.Client) port.CartRepository {
	return &redisRepo{rdb: rdb}
}

func (r *redisRepo) key(userID string) string {
	return "cart:" + userID
}

func (r *redisRepo) Get(ctx context.Context, userID string) (*domain.Cart, error) {
	data, err := r.rdb.Get(ctx, r.key(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.New(userID), nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get cart: %w", err)
	}

	var cart domain.Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return nil, fmt.Errorf("unmarshal cart: %w", err)
	}
	return &cart, nil
}

func (r *redisRepo) Save(ctx context.Context, cart *domain.Cart) error {
	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("marshal cart: %w", err)
	}
	return r.rdb.Set(ctx, r.key(cart.UserID), data, cartTTL).Err()
}

func (r *redisRepo) Delete(ctx context.Context, userID string) error {
	return r.rdb.Del(ctx, r.key(userID)).Err()
}
