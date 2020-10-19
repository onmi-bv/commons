package redis

import (
	"context"
	"strings"

	redis "github.com/go-redis/redis/v8"
	"github.com/onmi-bv/commons/confighelper"
	log "github.com/sirupsen/logrus"
)

// Client defines connection configurations
type Client struct {
	URL                string `mapstructure:"URL"`
	Database           int    `mapstructure:"DATABASE"`
	Password           string `mapstructure:"PASSWORD"`
	AuthEnabled        bool   `mapstructure:"AUTH_ENABLED"`
	SentinelEnabled    bool   `mapstructure:"SENTINEL_ENABLED"`
	SentinelMasterName string `mapstructure:"SENTINEL_MASTER_NAME"`
	redis.Cmdable
}

// Configuration used for initialization
type Configuration struct {
	Path   string // Path to config file
	Prefix string // Prefix to environment variables
}

// NewClient creates a config struct with the connection default values
func NewClient() Client {
	return Client{
		URL:             "redis:6379",
		Database:        0,
		AuthEnabled:     true,
		SentinelEnabled: false,
	}
}

// Initialize creates and initializes a redis universal client.
func (c *Client) Initialize(ctx context.Context) error {

	redisOpts := &redis.UniversalOptions{
		Addrs:      strings.Split(c.URL, ","),
		DB:         c.Database,
		MaxRetries: 5,
		MasterName: c.SentinelMasterName,
	}
	if c.AuthEnabled {
		redisOpts.Password = c.Password
	}
	c.Cmdable = redis.NewUniversalClient(redisOpts)
	_, err := c.Ping(ctx).Result()

	return err
}

// Init loads configuration from file or environment and initializes.
func Init(ctx context.Context, config Configuration) (Client, error) {
	c, err := Load(ctx, config)
	if err != nil {
		return c, err
	}
	err = c.Initialize(ctx)
	return c, err
}

// Load loads redis configuration from file and environment.
func Load(ctx context.Context, config Configuration) (c Client, err error) {
	c = NewClient()

	err = confighelper.ReadConfig(config.Path, config.Prefix, &c)
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

// Healthcheck checks if the dgraph server is online using the health endpoint.
func (c *Client) Healthcheck(ctx context.Context) error {
	_, err := c.Ping(ctx).Result()
	return err
}
