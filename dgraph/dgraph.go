package dgraph

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	HealthURL   string `mapstructure:"HEALTH_URL"`
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

// Configuration used for initialization
type Configuration struct {
	Path      string // Path to config file.
	Prefix    string // Prefix to environment variables.
	RetryDial int    // Retry grpc dial in case server requires a cold start
}

// Init client
func Init(ctx context.Context, conf Configuration) (Client, error) {

	client, err := Load(ctx, conf.Path, conf.Prefix)
	if err != nil {
		return client, fmt.Errorf("Load: %v", err)
	}

	if conf.RetryDial == 0 {
		conf.RetryDial = 3
	}

	for ; conf.RetryDial > 0; conf.RetryDial-- {

		if err = client.Initialize(ctx); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	return client, err
}

// Healthcheck checks if the dgraph server is online using the health endpoint.
func (c *Client) Healthcheck() error {
	req, _ := http.NewRequest("OPTIONS", c.HealthURL, nil)

	resp, err := http.DefaultClient.Do(req)

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

// // RetryDo makes request with retries
// func (c *Client) RetryDo(ctx context.Context, req *api.Request, retry int) (*api.Response, error) {
// 	var err error
// 	var res *api.Response
// 	for i := 0; i < retry; i++ {
// 		res, err = c.NewTxn().Do(ctx, req)

// 		if err != nil && strings.Contains(err.Error(), "i/o timeout") {
// 			continue
// 		}
// 		if err != nil {
// 			return res, err
// 		}
// 		break
// 	}
// 	return res, err
// }
