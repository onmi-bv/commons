package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/onmi-bv/commons/internal/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2"
)

// Config defines connection configurations
type Config struct {
	MongoUsername       string `mapstructure:"USERNAME"`
	MongoDatabase       string `mapstructure:"DATABASE"`
	MongoSource         string `mapstructure:"SOURCE"`
	MongoDataCollection string `mapstructure:"COLLECTION"`
	MongoPassword       string `mapstructure:"PASSWORD"`
	MongoURL            string `mapstructure:"URL"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context) (*mgo.Session, error) {

	info, err := mgo.ParseURL(c.MongoURL)
	if err != nil {
		return nil, fmt.Errorf("cannot parse mongo url '%v': %v", c.MongoURL, err)
	}

	info.Timeout = 60 * time.Second
	info.Database = c.MongoDatabase

	if c.MongoUsername != "" {
		info.Username = c.MongoUsername
	}
	if c.MongoPassword != "" {
		info.Password = c.MongoPassword
	}
	if c.MongoSource != "" {
		info.Source = c.MongoSource
	}

	session, err := mgo.DialWithInfo(info)
	if err != nil {
		info.Password = "<opaque>"
		return nil, fmt.Errorf("cannot initialize mongo session: %v, dialInfo: %+v", err, info)
	}

	err = session.Ping()
	if err != nil {
		info.Password = "<opaque>"
		return nil, fmt.Errorf("cannot connect to mongo database: %v, dialInfo: %+v", err, info)
	}

	return session, nil
}

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string) (c Config, m *mgo.Session, err error) {
	c, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	m, err = c.Initialize(ctx)

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
	log.Debugf("Mongo URL: %v", c.MongoURL)
	log.Debugf("Mongo database: %v", c.MongoDatabase)
	log.Debugf("Mongo collection: %v", c.MongoDataCollection)
	log.Debugf("Mongo source: %v", c.MongoSource)
	log.Debugf("Mongo username: %v", c.MongoUsername)
	log.Debugf("Mongo password: %v", "***")
	log.Debugln("...")

	return
}
