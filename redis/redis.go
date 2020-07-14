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

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string) (mConfig Config, red *redis.Client, err error) {
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

	red, err = mConfig.Initialize(ctx)
	return
}
