package connections

import (
	"context"
	"fmt"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectionsConfig defines all connections for luka services
type ConnectionsConfig struct {
	InfluxLocationURL         string `mapstructure:"INFLUX_LOCATION_URL"`
	InfluxLocationDB          string `mapstructure:"INFLUX_LOCATION_DB"`
	InfluxLocationUser        string `mapstructure:"INFLUX_LOCATION_USER"`
	InfluxLocationPwd         string `mapstructure:"INFLUX_LOCATION_PWD"`
	InfluxLocationMeasurement string `mapstructure:"INFLUX_LOCATION_MEASUREMENT"`

	InfluxStopsURL         string `mapstructure:"INFLUX_STOPS_URL"`
	InfluxStopsDB          string `mapstructure:"INFLUX_STOPS_DB"`
	InfluxStopsUser        string `mapstructure:"INFLUX_STOPS_USER"`
	InfluxStopsPwd         string `mapstructure:"INFLUX_STOPS_PWD"`
	InfluxStopsMeasurement string `mapstructure:"INFLUX_STOPS_MEASUREMENT"`

	RedisURL                string `mapstructure:"REDIS_URL"`
	RedisDB                 int    `mapstructure:"REDIS_DB"`
	RedisPwd                string `mapstructure:"REDIS_PWD"`
	RedisAuthEnabled        bool   `mapstructure:"REDIS_AUTH_ENABLED"`
	RedisSentinelEnabled    bool   `mapstructure:"REDIS_SENTINEL_ENABLED"`
	RedisSentinelMasterName string `mapstructure:"REDIS_SENTINEL_MASTER_NAME"`

	MongoEnabled    bool   `mapstructure:"MONGO_ENABLED"`
	MongoURI        string `mapstructure:"MONGO_URI"`
	MongoDB         string `mapstructure:"MONGO_DB"`
	MongoCollection string `mapstructure:"MONGO_COLLECTION"`
}

// NewConnectionsConfig creates a config struct with connections default values
func NewConnectionsConfig() ConnectionsConfig {
	return ConnectionsConfig{
		InfluxLocationURL:         "http://host.docker.internal:8086",
		InfluxLocationDB:          "lukadb",
		InfluxLocationUser:        "admin",
		InfluxLocationPwd:         "cledgeggauscerackeereare",
		InfluxLocationMeasurement: "location",

		InfluxStopsURL:         "http://influxdb:8086",
		InfluxStopsDB:          "lukadb",
		InfluxStopsUser:        "admin",
		InfluxStopsPwd:         "ewyelococlaytooddethigni",
		InfluxStopsMeasurement: "stops",

		RedisURL:                "redis:6379",
		RedisDB:                 0,
		RedisPwd:                "thisisjustaplaceholder",
		RedisAuthEnabled:        true,
		RedisSentinelEnabled:    false,
		RedisSentinelMasterName: "master",

		MongoEnabled:    false,
		MongoURI:        "mongodb://mongodb:27017",
		MongoDB:         "luka",
		MongoCollection: "days",
	}
}

// Redis creates and initializes a redis client.
func (c *ConnectionsConfig) Redis(ctx context.Context) (red *redis.Client, err error) {

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

// Mongo creates and initilizes mongo client
func (c *ConnectionsConfig) Mongo(ctx context.Context) (m *mongo.Client, err error) {
	mongoOpts := mongoOptions.Client().ApplyURI(c.MongoURI)
	m, err = mongo.Connect(ctx, mongoOpts)

	if err != nil {
		return nil, fmt.Errorf("cannot create mongo client: %v", err)
	}

	err = m.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot ping Mongo server: %v", err.Error())
	}
	return
}

// CloudEvent creates and initilizes cloudevent with http protocol.
func CloudEvent(ctx context.Context, port int) (ce cloudevents.Client, err error) {

	protocol, err := cehttp.New(cloudevents.WithPort(port))
	if err != nil {
		return ce, fmt.Errorf("failed to create cloudevent http protocol, %v", err)
	}
	ce, err = cloudevents.NewClientObserved(protocol,
		cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return ce, fmt.Errorf("failed to create cloudevent client, %v", err)
	}
	return
}
