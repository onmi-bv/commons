package redis

import (
	"context"
	"log"
	"strings"

	"github.com/go-redis/redis"
	"github.com/onmi-bv/commons/internal/config"
)

// Config defines connection configurations
type Config struct {
	RedisURL                string `mapstructure:"URL"`
	RedisDB                 int    `mapstructure:"DB"`
	RedisPwd                string `mapstructure:"PWD"`
	RedisAuthEnabled        bool   `mapstructure:"AUTH_ENABLED"`
	RedisSentinelEnabled    bool   `mapstructure:"SENTINEL_ENABLED"`
	RedisSentinelMasterName string `mapstructure:"SENTINEL_MASTER_NAME"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{
		RedisURL:                "redis:6379",
		RedisDB:                 0,
		RedisPwd:                "",
		RedisAuthEnabled:        true,
		RedisSentinelEnabled:    false,
		RedisSentinelMasterName: "master",
	}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context) (red *redis.Client, err error) {

	if c.RedisSentinelEnabled {
		log.Printf("using redis-sentinel with address %v\n", c.RedisURL)

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

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string) (cfg Config, red *redis.Client, err error) {
	cfg = NewConfig()

	err = config.ReadConfig(cFile, prefix, &cfg)
	if err != nil {
		return
	}

	red, err = cfg.Initialize(ctx)
	return
}
