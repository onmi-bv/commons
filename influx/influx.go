package influx

import (
	"context"
	"fmt"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/onmi-bv/commons/confighelper"
	log "github.com/sirupsen/logrus"
)

// Config defines connection configurations
type Config struct {
	URL         string `mapstructure:"URL"`
	Database    string `mapstructure:"DATABASE"`
	Measurement string `mapstructure:"MEASUREMENT"`
	AuthEnabled bool   `mapstructure:"AUTH_ENABLED"`
	Username    string `mapstructure:"USERNAME"`
	Password    string `mapstructure:"PASSWORD"`
}

// NewConfig creates a config struct with the connection default values
func NewConfig() Config {
	return Config{}
}

// Initialize creates and initializes a redis client.
func (c *Config) Initialize(ctx context.Context) (client.Client, error) {

	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     c.URL,
		Username: c.Username,
		Password: c.Password,
	})

	if err != nil {
		return nil, fmt.Errorf("cannot connect to influx database: %v", err)
	}

	// ping the influx server
	_, _, err = cli.Ping(0)
	if err != nil {
		return nil, fmt.Errorf("cannot ping InfluxLocation server: %v", err.Error())
	}

	return cli, nil
}

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string) (c Config, cli client.Client, err error) {
	c, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	cli, err = c.Initialize(ctx)

	return
}

// Load loads redis configuration from file and environment.
func Load(ctx context.Context, cFile string, prefix string) (c Config, err error) {
	c = NewConfig()

	err = confighelper.ReadConfig(cFile, prefix, &c)
	if err != nil {
		return
	}

	log.Debugf("# Influx config... ")
	log.Debugf("Influx URL: %v", c.URL)
	log.Debugf("Influx database: %v", c.Database)
	log.Debugf("Influx measurement: %v", c.Measurement)
	log.Debugf("Influx auth enabled: %v", c.AuthEnabled)
	log.Debugf("Influx username: %v", c.Username)
	if c.Password != "" {
		log.Debugf("Influx password: %v", "***")
	} else {
		log.Debugf("Influx password: %v", "<empty>")
	}
	log.Debugln("...")

	return
}
