package mongo

import (
	"context"
	"fmt"

	"github.com/onmi-bv/commons/internal/config"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config defines connection configurations
type Config struct {
	URI            string `mapstructure:"URI"`
	Username       string `mapstructure:"USERNAME"`
	Database       string `mapstructure:"DATABASE"`
	Source         string `mapstructure:"SOURCE"`
	DataCollection string `mapstructure:"COLLECTION"`
	Password       string `mapstructure:"PASSWORD"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context, appName string) (*mongo.Client, error) {

	mongoOpts := options.Client().ApplyURI(c.URI)
	mongoOpts.AppName = &appName

	if c.Username != "" {
		mongoOpts.Auth.Username = c.Username
	}
	if c.Password != "" {
		mongoOpts.Auth.Password = c.Password
	}
	if c.Source != "" {
		mongoOpts.Auth.AuthSource = c.Source
	}

	m, err := mongo.Connect(ctx, mongoOpts)

	if err != nil {
		return nil, fmt.Errorf("cannot create mongo client: %v", err)
	}

	err = m.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot ping Mongo server: %v", err.Error())
	}
	return m, nil
}

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string, appName string) (c Config, m *mongo.Client, err error) {
	c, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	m, err = c.Initialize(ctx, appName)

	return
}

// Load loads redis configuration from file and environment.
func Load(ctx context.Context, cFile string, prefix string) (c Config, err error) {
	c = NewConfig()

	err = config.ReadConfig(cFile, prefix, &c)
	if err != nil {
		return
	}

	// MongoDataCollection string `mapstructure:"COLLECTION"`

	log.Debugf("# Mongo config... ")
	log.Debugf("Mongo URL: %v", c.URI)
	log.Debugf("Mongo database: %v", c.Database)
	log.Debugf("Mongo collection: %v", c.DataCollection)
	log.Debugf("Mongo source: %v", c.Source)
	log.Debugf("Mongo username: %v", c.Username)
	log.Debugf("Mongo password: %v", "***")
	log.Debugln("...")

	return
}
