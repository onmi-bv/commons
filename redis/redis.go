package redis

import (
	"context"
	"strings"

	"github.com/go-redis/redis"
	"github.com/onmi-bv/commons/internal/config"
	log "github.com/sirupsen/logrus"
)

// Config defines connection configurations
type Config struct {
	RedisURL                string `mapstructure:"URL"`
	RedisDB                 int    `mapstructure:"DATABASE"`
	RedisPwd                string `mapstructure:"PASSWORD"`
	RedisAuthEnabled        bool   `mapstructure:"AUTH_ENABLED"`
	RedisSentinelEnabled    bool   `mapstructure:"SENTINEL_ENABLED"`
	RedisSentinelMasterName string `mapstructure:"SENTINEL_MASTER_NAME"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{
		RedisURL:             "redis:6379",
		RedisDB:              0,
		RedisPwd:             "",
		RedisAuthEnabled:     true,
		RedisSentinelEnabled: false,
	}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context) (red *redis.Client, err error) {

	if c.RedisSentinelEnabled {
		log.Debugf("using redis-sentinel with address %v", c.RedisURL)

		redisOpts := &redis.FailoverOptions{
			MasterName:    c.RedisSentinelMasterName,
			SentinelAddrs: strings.Split(c.RedisURL, ","),
			DB:            c.RedisDB,
			MaxRetries:    5,
		}
		if c.RedisAuthEnabled {
			redisOpts.Password = c.RedisPwd
		}
		red = redis.NewFailoverClient(redisOpts)
	} else {
		redisOpts := &redis.Options{
			Addr:       c.RedisURL,
			DB:         c.RedisDB,
			MaxRetries: 5,
		}
		if c.RedisAuthEnabled {
			redisOpts.Password = c.RedisPwd
		}
		red = redis.NewClient(redisOpts)
	}

	_, err = red.Ping().Result()

	return
}

// InitializeUniversalClient creates and initializes a redis universal client.
func (c *Config) InitializeUniversalClient(ctx context.Context) (redis.UniversalClient, error) {

	redisOpts := &redis.UniversalOptions{
		Addrs:      strings.Split(c.RedisURL, ","),
		DB:         c.RedisDB,
		MaxRetries: 5,
		MasterName: c.RedisSentinelMasterName,
	}
	if c.RedisAuthEnabled {
		redisOpts.Password = c.RedisPwd
	}
	client := redis.NewUniversalClient(redisOpts)
	_, err := client.Ping().Result()

	return client, err
}

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string) (mConfig Config, red *redis.Client, err error) {
	mConfig, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	red, err = mConfig.Initialize(ctx)
	return
}

// Load loads redis configuration from file and environment.
func Load(ctx context.Context, cFile string, prefix string) (mConfig Config, err error) {
	mConfig = NewConfig()

	err = config.ReadConfig(cFile, prefix, &mConfig)
	if err != nil {
		return
	}

	log.Tracef("# Connecting to Redis... ")
	log.Tracef("Redis URL: %v", mConfig.RedisURL)
	log.Tracef("Redis database: %v", mConfig.RedisDB)
	log.Tracef("Redis auth enabled: %v", mConfig.RedisAuthEnabled)
	log.Tracef("Redis password: %v", "***")
	log.Tracef("Redis sentinel enabled: %v", mConfig.RedisSentinelEnabled)
	log.Tracef("Redis sentinel master: %v", mConfig.RedisSentinelMasterName)
	log.Traceln("...")

	return
}
