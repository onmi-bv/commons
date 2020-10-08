package dgraph

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"github.com/dgraph-io/dgo/v2"
	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/onmi-bv/commons/confighelper"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client defines graphql host parameters and client.
type Client struct {
	Host        string `mapstructure:"GRPC_HOST"`
	AuthEnabled bool   `mapstructure:"AUTH_ENABLED"`
	AuthSecret  string `mapstructure:"SECRET"`
	HealthURL   string `mapstructure:"URL"`
	*dgo.Dgraph
}

type authorizationCredentials struct {
	token string
}

func (a *authorizationCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"Authorization": a.token}, nil
}

func (a *authorizationCredentials) RequireTransportSecurity() bool {
	return true
}

// Load loads the graphql host parameters from environment
func Load(ctx context.Context, cFile string, prefix string) (Client, error) {
	c := Client{}

	if err := confighelper.ReadConfig(cFile, prefix, &c); err != nil {
		return c, err
	}

	log.Debugf("# Dgraph config... ")
	log.Debugf("Dgraph GRPC HOST: %v", c.Host)
	log.Debugf("Dgraph auth enabled: %v", c.AuthEnabled)

	if c.AuthSecret != "" {
		log.Debugf("Dgraph secret: %v", "***")
	} else {
		log.Debugf("Dgraph secret: %v", "<empty>")
	}

	log.Debugf("Dgraph health URL: %v", c.HealthURL)
	log.Debugln("...")

	return c, nil
}

// Initialize creates and initializes a redis client.
func (c *Client) Initialize(ctx context.Context) (err error) {

	var conn *grpc.ClientConn

	if !c.AuthEnabled {
		conn, err = grpc.Dial(c.Host, grpc.WithInsecure())

	} else {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return err
		}

		tls := credentials.NewTLS(&tls.Config{
			RootCAs:    pool,
			ServerName: strings.Split(c.Host, ":")[0],
		})

		auth := authorizationCredentials{token: c.AuthSecret}

		conn, err = grpc.Dial(c.Host, grpc.WithTransportCredentials(tls), grpc.WithPerRPCCredentials(&auth))

		if err != nil {
			return err
		}
	}

	dgraphClient := api.NewDgraphClient(conn)
	c.Dgraph = dgo.NewDgraphClient(dgraphClient)

	return
}

// LoadAndInitialize loads configuration from file or environment and initializes.
func LoadAndInitialize(ctx context.Context, cFile string, prefix string) (cli Client, err error) {
	cli, err = Load(ctx, cFile, prefix)
	if err != nil {
		return
	}
	err = cli.Initialize(ctx)

	return
}

// Healthcheck checks if the dgraph server is online using the health endpoint.
func (c *Client) Healthcheck() error {
	resp, err := http.Get(c.HealthURL)

	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("got error code %d", resp.StatusCode)
		log.Error(err)
		return err
	}

	return nil
}
