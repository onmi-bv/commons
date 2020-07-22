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
func LoadAndInitialize(ctx context.Context, cFile string, prefix string) (c Config, red *redis.Client, err error) {
	c, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	red, err = c.Initialize(ctx)
	return
}

// Load loads redis configuration from file and environment.
func Load(ctx context.Context, cFile string, prefix string) (c Config, err error) {
	c = NewConfig()

	err = config.ReadConfig(cFile, prefix, &c)
	if err != nil {
		return
	}

	log.Debugf("# Redis config... ")
	log.Debugf("Redis URL: %v", c.RedisURL)
	log.Debugf("Redis database: %v", c.RedisDB)
	log.Debugf("Redis auth enabled: %v", c.RedisAuthEnabled)
	log.Debugf("Redis password: %v", "***")
	log.Debugf("Redis sentinel enabled: %v", c.RedisSentinelEnabled)
	log.Debugf("Redis sentinel master: %v", c.RedisSentinelMasterName)
	log.Debugln("...")

	return
}
