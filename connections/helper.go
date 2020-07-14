package connections

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// BindEnvs binds the mapstructure to the environment variables
func bindEnvs(iface interface{}, parts ...string) {
	var ifv reflect.Value
	var ift reflect.Type
	if reflect.TypeOf(iface).Kind() == reflect.Struct {
		ifv = reflect.ValueOf(iface)
		ift = reflect.TypeOf(iface)
	} else {
		ifv = reflect.ValueOf(iface).Elem()
		ift = reflect.TypeOf(iface).Elem()
	}
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			bindEnvs(v.Interface(), append(parts, tv)...)
		default:
			viper.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

// ReadConfig loads the application configuration
func ReadConfig(cfgFile string, prefix string, config interface{}) error {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		// Search config in home directory with name ".places" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".conf")
	}

	viper.SetEnvPrefix(prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		return err
	}

	// unmarshal
	bindEnvs(config)
	if err := viper.Unmarshal(config); err != nil {
		return err
	}

	return nil
}

// FatalGet gets env. variable and panics if not set
func FatalGet(env string, fallback string) string {
	s := os.Getenv(env)
	if s == "" {
		if fallback == "" {
			panic(fmt.Sprintf("%s not set", env))
		} else {
			return fallback
		}
	}
	return s
}

// GetEnv gets an env. variable without panicing
func GetEnv(env string, fallback string) string {
	s := os.Getenv(env)
	if s == "" {
		return fallback
	}
	return s
}
