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
	MongoURI            string `mapstructure:"URI"`
	MongoUsername       string `mapstructure:"USERNAME"`
	MongoDatabase       string `mapstructure:"DATABASE"`
	MongoSource         string `mapstructure:"SOURCE"`
	MongoDataCollection string `mapstructure:"COLLECTION"`
	MongoPassword       string `mapstructure:"PASSWORD"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context, appName string) (*mongo.Client, error) {

	mongoOpts := options.Client().ApplyURI(c.MongoURI)
	mongoOpts.AppName = &appName

	if c.MongoUsername != "" {
		mongoOpts.Auth.Username = c.MongoUsername
	}
	if c.MongoPassword != "" {
		mongoOpts.Auth.Password = c.MongoPassword
	}
	if c.MongoSource != "" {
		mongoOpts.Auth.AuthSource = c.MongoSource
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
	log.Debugf("Mongo URL: %v", c.MongoURI)
	log.Debugf("Mongo database: %v", c.MongoDatabase)
	log.Debugf("Mongo collection: %v", c.MongoDataCollection)
	log.Debugf("Mongo source: %v", c.MongoSource)
	log.Debugf("Mongo username: %v", c.MongoUsername)
	log.Debugf("Mongo password: %v", "***")
	log.Debugln("...")

	return
}
