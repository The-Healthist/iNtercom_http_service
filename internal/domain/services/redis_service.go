package services

import (
	"context"
	"encoding/json"
	"fmt"
	"intercom_http_service/internal/domain/models"
	"intercom_http_service/internal/infrastructure/config"
	"time"

	"github.com/go-redis/redis/v8"
)

// InterfaceRedisService defines the Redis service interface
type InterfaceRedisService interface {
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string, dest interface{}) error
	Delete(key string) error
	CacheRTCToken(userID, channelID, token string, expiration time.Duration) error
	GetRTCToken(userID, channelID string) (string, error)
	GetCallRecordByID(id string) (*models.CallRecord, error)
	CacheCallRecord(record *models.CallRecord, expiration time.Duration) error
}

// RedisService handles Redis operations
type RedisService struct {
	Client *redis.Client
	Ctx    context.Context
}

// NewRedisService creates a new Redis service
func NewRedisService(cfg *config.Config) InterfaceRedisService {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: "", // No password set
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()

	return &RedisService{
		Client: client,
		Ctx:    ctx,
	}
}

// 1 Set sets a key-value pair in Redis with expiration
func (s *RedisService) Set(key string, value interface{}, expiration time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.Client.Set(s.Ctx, key, jsonValue, expiration).Err()
}

// 2 Get gets a value from Redis by key
func (s *RedisService) Get(key string, dest interface{}) error {
	val, err := s.Client.Get(s.Ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// 3 Delete deletes a key from Redis
func (s *RedisService) Delete(key string) error {
	return s.Client.Del(s.Ctx, key).Err()
}

// 4 CacheRTCToken caches an RTC token with expiration
func (s *RedisService) CacheRTCToken(userID, channelID, token string, expiration time.Duration) error {
	key := "rtc_token:" + userID + ":" + channelID
	return s.Client.Set(s.Ctx, key, token, expiration).Err()
}

// 5 GetRTCToken gets an RTC token from cache
func (s *RedisService) GetRTCToken(userID, channelID string) (string, error) {
	key := "rtc_token:" + userID + ":" + channelID
	return s.Client.Get(s.Ctx, key).Result()
}

// 6 GetCallRecordByID gets a call record by ID from cache
func (s *RedisService) GetCallRecordByID(id string) (*models.CallRecord, error) {
	var record models.CallRecord
	key := "call_record:" + id
	err := s.Get(key, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// 7 CacheCallRecord caches a call record with expiration
func (s *RedisService) CacheCallRecord(record *models.CallRecord, expiration time.Duration) error {
	key := fmt.Sprintf("call_record:%d", record.ID)
	return s.Set(key, record, expiration)
}
