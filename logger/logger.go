package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/onmi-bv/commons/internal/slackrus"

	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/evalphobia/logrus_fluent"
	"github.com/onmi-bv/commons/internal/config"
	logger "github.com/sirupsen/logrus"
)

// Config defines application configuration
type Config struct {
	// Output sets the output destination for the logger. I.e., stderr, stdout, discard
	Output string `mapstructure:"OUTPUT"`

	// Level sets the log level. I.e., trace, debug, info, warning, error
	Level string `mapstructure:"LEVEL"`

	// Formatter sets the output formatter type. I.e., opts: text (plain text), json, sd (stackdriver format)
	Formatter string `mapstructure:"FORMATTER"`

	// External sets external logging. Enable to use fluentd
	External bool `mapstructure:"EXTERNAL"`

	// FluentdHost sets the fluentd host
	FluentdHost string `mapstructure:"FLUENTD_HOST"`

	// FluentdPort sets the fluentd port
	FluentdPort int `mapstructure:"FLUENTD_PORT"`

	// FieldMap (json) allows users to customize the names of keys for default fields.
	// FieldKeyTime:  "@timestamp"
	// FieldKeyLevel: "@level"
	// FieldKeyMsg:   "@message"
	FieldMap string `mapstructure:"FIELD_MAP"`

	// PrettyPrint will indent all json logs
	PrettyPrint bool `mapstructure:"PRETTY_PRINT"`

	// SetReporterCaller enables logging the report caller
	SetReporterCaller bool `mapstructure:"SET_REPORTER_CALLER"`

	// Slack configures slack integration
	Slack slackrus.Hook
}

// NewConfig creates a config struct with log default values
func NewConfig() Config {
	return Config{
		Level:             "info",
		External:          false,
		FluentdHost:       "127.0.0.1",
		FluentdPort:       24224,
		FieldMap:          "",
		PrettyPrint:       false,
		SetReporterCaller: false,
		Formatter:         "text",
		Slack:             slackrus.NewHook(),
	}
}

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string, appName string, version string) (mConfig Config, mLogger *logger.Logger, err error) {
	mConfig = NewConfig()

	err = config.ReadConfig(cFile, prefix, &mConfig)
	if err != nil {
		return
	}

	mLogger, err = mConfig.Initialize(ctx, appName, version)
	return
}

// Initialize implements logic for application log configuration
func (config *Config) Initialize(ctx context.Context, appName string, appVersion string) (*logger.Logger, error) {

	// * set log level
	logLevel, err := logger.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	logger.SetLevel(logLevel)

	// * set output
	switch config.Output {
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "discard":
		logger.SetOutput(ioutil.Discard)
	default:
	}

	// * parse field map
	fieldMap := logger.FieldMap{}
	if config.FieldMap != "" {
		err = json.Unmarshal([]byte(config.FieldMap), &fieldMap)
		if err != nil {
			return nil, fmt.Errorf("cannot parse logFieldMap %v, error: %v", config.FieldMap, err)
		}
	}

	// * set log formatter
	switch config.Formatter {
	case "text":
		logger.SetFormatter(&logger.TextFormatter{FieldMap: fieldMap, DisableColors: false, ForceColors: true})
	case "json":
		logger.SetFormatter(&logger.JSONFormatter{FieldMap: fieldMap, PrettyPrint: config.PrettyPrint})
	case "sd":
		logger.Debug("set stackdriver log formatter")
		logger.SetFormatter(stackdriver.NewFormatter(
			stackdriver.WithService(appName),
			stackdriver.WithVersion(appVersion),
		))
	}

	// set log report caller
	logger.SetReportCaller(config.SetReporterCaller)

	// log external
	if config.External {
		hook, err := logrus_fluent.NewWithConfig(logrus_fluent.Config{
			Host: config.FluentdHost,
			Port: config.FluentdPort,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot configure new logrus webhook to FluentD: %v", err)
		}
		logger.AddHook(hook)
	}

	//  add slack
	if config.Slack.Username == "" {
		config.Slack.Username = appName
	}
	logger.AddHook(&config.Slack)

	logger.Debugf("log level: %v", config.Level)

	return logger.StandardLogger(), nil
}
