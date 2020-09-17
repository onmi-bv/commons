package redis

import (
	"context"
	"strings"

	"github.com/go-redis/redis"
	redisv8 "github.com/go-redis/redis/v8"
	"github.com/onmi-bv/commons/confighelper"
	log "github.com/sirupsen/logrus"
)

// Config defines connection configurations
type Config struct {
	URL                string `mapstructure:"URL"`
	Database           int    `mapstructure:"DATABASE"`
	Password           string `mapstructure:"PASSWORD"`
	AuthEnabled        bool   `mapstructure:"AUTH_ENABLED"`
	SentinelEnabled    bool   `mapstructure:"SENTINEL_ENABLED"`
	SentinelMasterName string `mapstructure:"SENTINEL_MASTER_NAME"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{
		URL:             "redis:6379",
		Database:        0,
		AuthEnabled:     true,
		SentinelEnabled: false,
	}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context) (red *redis.Client, err error) {

	if c.SentinelEnabled {
		log.Debugf("using redis-sentinel with address %v", c.URL)

		redisOpts := &redis.FailoverOptions{
			MasterName:    c.SentinelMasterName,
			SentinelAddrs: strings.Split(c.URL, ","),
			DB:            c.Database,
			MaxRetries:    5,
		}
		if c.AuthEnabled {
			redisOpts.Password = c.Password
		}
		red = redis.NewFailoverClient(redisOpts)
	} else {
		redisOpts := &redis.Options{
			Addr:       c.URL,
			DB:         c.Database,
			MaxRetries: 5,
		}
		if c.AuthEnabled {
			redisOpts.Password = c.Password
		}
		red = redis.NewClient(redisOpts)
	}

	_, err = red.Ping().Result()

	return
}

// InitializeUniversalClient creates and initializes a redis universal client.
func (c *Config) InitializeUniversalClient(ctx context.Context) (redis.UniversalClient, error) {

	redisOpts := &redis.UniversalOptions{
		Addrs:      strings.Split(c.URL, ","),
		DB:         c.Database,
		MaxRetries: 5,
		MasterName: c.SentinelMasterName,
	}
	if c.AuthEnabled {
		redisOpts.Password = c.Password
	}
	client := redis.NewUniversalClient(redisOpts)
	_, err := client.Ping().Result()

	return client, err
}

// InitializeUniversalClientV8 creates and initializes a redis universal client for v8.
func (c *Config) InitializeUniversalClientV8(ctx context.Context) (redisv8.UniversalClient, error) {

	redisOpts := &redisv8.UniversalOptions{
		Addrs:      strings.Split(c.URL, ","),
		DB:         c.Database,
		MaxRetries: 5,
		MasterName: c.SentinelMasterName,
	}
	if c.AuthEnabled {
		redisOpts.Password = c.Password
	}
	client := redisv8.NewUniversalClient(redisOpts)
	_, err := client.Ping(ctx).Result()

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

// LoadAndInitializeUniversalClient loads configuration from file or environment and initializes.
func LoadAndInitializeUniversalClient(ctx context.Context, cFile string, prefix string) (c Config, red redis.UniversalClient, err error) {
	c, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	red, err = c.InitializeUniversalClient(ctx)
	return
}

// LoadAndInitializeUniversalClientV8 loads configuration from file or environment and initializes.
func LoadAndInitializeUniversalClientV8(ctx context.Context, cFile string, prefix string) (c Config, red redisv8.UniversalClient, err error) {
	c, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	red, err = c.InitializeUniversalClientV8(ctx)
	return
}

// Load loads redis configuration from file and environment.
func Load(ctx context.Context, cFile string, prefix string) (c Config, err error) {
	c = NewConfig()

	err = confighelper.ReadConfig(cFile, prefix, &c)
	if err != nil {
		return
	}

	log.Debugf("# Redis config... ")
	log.Debugf("Redis URL: %v", c.URL)
	log.Debugf("Redis database: %v", c.Database)
	log.Debugf("Redis auth enabled: %v", c.AuthEnabled)
	log.Debugf("Redis password: %v", "***")
	log.Debugf("Redis sentinel enabled: %v", c.SentinelEnabled)
	log.Debugf("Redis sentinel master: %v", c.SentinelMasterName)
	log.Debugln("...")

	return
}
