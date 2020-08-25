package mongo

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/onmi-bv/commons/testutils"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// * flags
	var enableLogging = flag.Bool("log", false, "enable application logging")
	var logLevel = flag.String("log-level", "info", "enable application logging")
	flag.Parse()

	os.Setenv("LOG_LEVEL", *logLevel)
	os.Setenv("LOG_FORMATTER", "text")
	os.Setenv("LOG_SET_REPORTER_CALLER", "false")
	if !*enableLogging {
		os.Setenv("LOG_OUTPUT", "discard")
	}

	os.Setenv("MONGO_URI", "mongodb://localhost:27000")
	os.Setenv("MONGO_DATABASE", "test")
	os.Setenv("MONGO_COLLECTION", "data")
	os.Setenv("MONGO_SOURCE", "admin")
	os.Setenv("MONGO_USERNAME", "root")
	os.Setenv("MONGO_PASSWORD", "secret")

	// * setup mongo using docker
	_, mongoID, _ := testutils.CreateNewContainer("mongo", "27000", "27017", []string{"MONGO_INITDB_ROOT_USERNAME=root", "MONGO_INITDB_ROOT_PASSWORD=secret"})
	defer testutils.RemoveContainer(mongoID) // make sure to stop container in case of fatal error

	//* run test
	res := m.Run()

	// * close mongo running in docker
	testutils.RemoveContainer(mongoID) // stop container before exiting

	os.Exit(res)
}

func TestLoadAndInit(t *testing.T) {
	_, m, err := LoadAndInitialize(context.Background(), "", "mongo", "testmongo")
	assert.NoErrorf(t, err, "cannot load and initialize mongo client: %v", err)
	assert.NotEmpty(t, m, "Expected a mongo client.")
}

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		args    Config
		wantErr bool
	}{
		// Add test cases.
		{
			name:    "empty config",
			args:    Config{},
			wantErr: true,
		},
		{
			name: "invalid uri",
			args: Config{
				URI: "invalid uri",
			},
			wantErr: true,
		},
		{
			name: "valid uri",
			args: Config{
				URI: "mongodb://localhost:27000",
			},
			wantErr: false,
		},
		{
			name: "auth enabled without credentials",
			args: Config{
				URI:         "mongodb://localhost:27000",
				AuthEnabled: true,
			},
			wantErr: true,
		},
		{
			name: "auth enabled with only source",
			args: Config{
				URI:         "mongodb://localhost:27000",
				AuthEnabled: true,
				Source:      "admin",
			},
			wantErr: true,
		},
		{
			name: "auth enabled with only source, and username",
			args: Config{
				URI:         "mongodb://localhost:27000",
				AuthEnabled: true,
				Source:      "admin",
				Username:    "root",
			},
			wantErr: true,
		},
		{
			name: "auth enabled",
			args: Config{
				URI:         "mongodb://localhost:27000",
				AuthEnabled: true,
				Source:      "admin",
				Username:    "root",
				Password:    "secret",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.args.Initialize(context.Background(), "testmongo")

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.TestInit() error = %+v, wantErr %+v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			assert.NotEmpty(t, m, "Expected a mongo client.")
		})
	}

	// cleanup
	// service.mongo.Database("test").Collection(service.mongoConfig.Collection).Drop(service.ctx)
}
