package mongo

import (
	"context"
	"fmt"

	"github.com/onmi-bv/commons/confighelper"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config defines connection configurations
type Config struct {
	URI         string `mapstructure:"URI"`
	Database    string `mapstructure:"DATABASE"`
	Collection  string `mapstructure:"COLLECTION"`
	AuthEnabled bool   `mapstructure:"AUTH_ENABLED"`
	Username    string `mapstructure:"USERNAME"`
	Source      string `mapstructure:"SOURCE"`
	Password    string `mapstructure:"PASSWORD"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context, appName string) (*mongo.Client, error) {

	mongoOpts := options.Client().ApplyURI(c.URI)
	mongoOpts.AppName = &appName

	if c.AuthEnabled {
		cred := options.Credential{
			AuthSource: c.Source,
			Username:   c.Username,
			Password:   c.Password,
		}
		mongoOpts.SetAuth(cred)
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

	log.Debugf("# Mongo config... ")
	log.Debugf("Mongo URI: %v", c.URI)
	log.Debugf("Mongo database: %v", c.Database)
	log.Debugf("Mongo collection: %v", c.Collection)
	log.Debugf("Mongo auth enabled: %v", c.AuthEnabled)
	log.Debugf("Mongo source: %v", c.Source)
	log.Debugf("Mongo username: %v", c.Username)
	if c.Password != "" {
		log.Debugf("Mongo password: %v", "***")
	} else {
		log.Debugf("Mongo password: %v", "<empty>")
	}
	log.Debugln("...")

	err = confighelper.ReadConfig(cFile, prefix, &c)
	if err != nil {
		return
	}

	return
}
