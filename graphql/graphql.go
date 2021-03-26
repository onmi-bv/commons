package graphql

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/onmi-bv/commons/confighelper"
	graphqlapi "github.com/onmi-bv/commons/graphql/api"
	log "github.com/sirupsen/logrus"
)

// Node defines a base interface for nodes
type Node interface {
	Key() string         // Gets the node ID key.
	GetID() string       // Gets the node ID value.
	DType() string       // Gets the node GraphQL type.
	Create() interface{} // Create an empty type for parsing new values.
	Patch() interface{}  // Get the patchable fields for updating node.
}

// Client defines graphql host parameters.
type Client struct {
	Host        string `mapstructure:"HOST"`
	HealthURL   string `mapstructure:"HEALTH_URL"`
	AuthEnabled bool   `mapstructure:"AUTH_ENABLED"`
	AuthSecret  string `mapstructure:"SECRET"`
	Proxy       string `mapstructure:"PROXY"`
	*graphqlapi.Client
}

// Configuration used for initialization
type Configuration struct {
	Path          string // Path to config file.
	Prefix        string // Prefix to environment variables.
	RequestOption RequestOption
}

// Init client
func Init(ctx context.Context, conf Configuration) (Client, error) {
	c, err := LoadConfig(ctx, conf.Path, conf.Prefix)
	if err != nil {
		return c, fmt.Errorf("Load: %v", err)
	}
	c.RequestOption = conf.RequestOption
	c.Log = func(s string) { log.Trace(s) }
	return c, err
}

// LoadConfig loads the graphql host parameters from environment
func LoadConfig(ctx context.Context, cFile string, prefix string) (Client, error) {
	c := Client{}

	if err := confighelper.ReadConfig(cFile, prefix, &c); err != nil {
		return c, err
	}

	log.Debugf("# GraphQL config... ")
	log.Debugf("GraphQL Host: %v", c.Host)
	log.Debugf("GraphQL auth enabled: %v", c.AuthEnabled)

	if c.AuthSecret != "" {
		log.Debugf("GraphQL secret: %v", "***")
	} else {
		log.Debugf("GraphQL secret: %v", "<empty>")
	}

	log.Debugf("GraphQL health URL: %v", c.HealthURL)

	// setup client with auth proxy
	if proxy, err := url.Parse(c.Proxy); err == nil && proxy.String() != "" {

		log.Debugf("GraphQL proxy: %s", proxy.String())

		// use custom client with proxy
		host, _ := url.Parse(c.Host)
		host.Host = proxy.Host
		c.Client = graphqlapi.NewClient(host.String())
	} else {
		c.Client = graphqlapi.NewClient(c.Host)
	}

	log.Debugln("...")

	return c, nil
}

// MutationResult defines GraphQL mutation result
type MutationResult struct {
	NumUids int
	Query   interface{}
	Errors  []struct {
		Message string
	}
}

// Request is a GraphQL request.
type Request = graphqlapi.Request

// RequestOption are functions that are passed to
// modify the graphql requests. Use function to modify headers, Vars.
type RequestOption = func(*Request)

// RetryRun makes request with retries
func (c *Client) RetryRun(ctx context.Context, req *graphqlapi.Request, resp interface{}, retry int) error {
	var err error
	for i := 0; i < retry; i++ {
		err = c.Run(ctx, req, resp)

		if err != nil && strings.Contains(err.Error(), "i/o timeout") {
			log.Warnf("graphql: retrying.. %d", i)
			time.Sleep(1 * time.Second)
			continue
		}
		if err != nil {
			return err
		}
		break
	}
	return err
}

// UpsertNode adds or updates a node.
func (c *Client) UpsertNode(ctx context.Context, node Node, opts ...RequestOption) (uid string, err error) {
	log.Tracef("saving.. node: %v %v", node.DType(), node.GetID())

	// update record
	res, err := c.UpdateNode(ctx, node, opts...)
	if err != nil {
		log.Warningf("could not update node: %v", err)
	}

	// if no record was updated, add it
	if res == nil || res.NumUids == 0 {
		res, err = c.AddNode(ctx, []Node{node}, opts...)
		if err != nil {
			return node.GetID(), fmt.Errorf("could not add node: %v", err)
		}
	}

	if res != nil && res.NumUids == 0 {
		log.Warningf("record was not upserted: %v %v, numUids=%v", node.DType(), node.GetID(), res.NumUids)
	}

	return node.GetID(), err
}

// UpdateNode uses the update<type> GraphQL API to update a node.
func (c *Client) UpdateNode(ctx context.Context, node Node, opts ...RequestOption) (*MutationResult, error) {
	log.Debugf("updating node: %v %v", node.DType(), node.GetID())

	if node.GetID() == "" {
		return nil, fmt.Errorf("updateNode requires XID in node")
	}

	dtype := node.DType()

	// save node
	query := `
	mutation update` + dtype + `Mutation ($set: ` + dtype + `Patch) {
		update` + dtype + `(input: {
			filter: {` + node.Key() + `: {eq: "` + node.GetID() + `"}}
			set: $set
		}){
			numUids
		}
	}`

	// log.Tracef("graphql query: %v", query)

	// b, _ := json.MarshalIndent(node.Patch(), "  ", "  ")
	// log.Tracef("graphql node: %v", string(b))

	// make a request
	req := graphqlapi.NewRequest(query)

	// set any variables
	req.Var("set", node.Patch())

	// run request functions
	for _, optionFunc := range opts {
		optionFunc(req)
	}

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}

	if err := c.RetryRun(ctx, req, &respData, 3); err != nil {
		return nil, err
	}

	res := MutationResult{
		NumUids: respData[`update`+dtype].NumUids,
	}

	// log.Infof("Result: %v, res: %+v", respData, res)
	return &res, nil
}

// AddNode uses the Add<type> API to add new nodes.
// If more than 1 node is added, make sure they are of same types and their IDs doesn't exist,
// otherwise, use Upsert tp add them individually.
func (c *Client) AddNode(ctx context.Context, node []Node, opts ...RequestOption) (*MutationResult, error) {

	if len(node) == 0 {
		return nil, fmt.Errorf("addNodes requires nodes to add, but received none")
	}

	log.Debugf("adding.. %v nodes: %v %v", len(node), node[0].DType(), node[0].GetID())

	if node[0].GetID() == "" {
		return nil, fmt.Errorf("addNode requires XID in node")
	}

	dtype := node[0].DType()

	// save node
	query := `
	mutation add` + dtype + `Mutation ($set: [Add` + dtype + `Input!]!) {
		add` + dtype + `(input: $set){
			numUids
		}
	}`

	// log.Tracef("graphql query: %v", query)

	// b, _ := json.MarshalIndent(node, "  ", "  ")
	// log.Tracef("graphql node: %v", string(b))

	// make a request
	req := graphqlapi.NewRequest(query)

	// set any variables
	req.Var("set", node)

	// run request functions
	for _, optionFunc := range opts {
		optionFunc(req)
	}

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}

	if err := c.RetryRun(ctx, req, &respData, 3); err != nil {
		return nil, err
	}

	res := MutationResult{
		NumUids: respData[`add`+dtype].NumUids,
	}

	return &res, nil
}

// DeleteNodeByID uses the Delete<type> API to delete a node.
// Action is unreversable and should be used with care.
func (c *Client) DeleteNodeByID(ctx context.Context, _type string, ids []string, opts ...RequestOption) (*MutationResult, error) {

	log.Debugf("deleting.. %v nodes: %v", _type, ids)

	// delete node
	query := `
	mutation delete` + _type + `Mutation ($id: [String]) {
		delete` + _type + `(filter: { id: { in: $id }}){
			numUids
		}
	}`

	// log.Tracef("graphql query: %v", query)

	// make a request
	req := graphqlapi.NewRequest(query)

	// set any variables
	req.Var("id", ids)

	// run request functions
	for _, optionFunc := range opts {
		optionFunc(req)
	}

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}

	if err := c.RetryRun(ctx, req, &respData, 3); err != nil {
		return nil, err
	}

	res := MutationResult{
		NumUids: respData[`delete`+_type].NumUids,
	}

	return &res, nil
}

// CustomNodeMutation uses the custom API
func (c *Client) CustomNodeMutation(ctx context.Context, customFn string, inputType string, node interface{}, opts ...RequestOption) (*MutationResult, error) {

	log.Debugf("custom.. %s", customFn)

	// b, _ := json.MarshalIndent(node, "  ", "  ")
	// log.Tracef("graphql node: %v", string(b))

	// delete node
	query := `
	mutation ` + customFn + `Mutation ($input: ` + inputType + `) {
		` + customFn + `(input: $input){
			numUids
		}
	}`

	// make a request
	req := graphqlapi.NewRequest(query)

	req.Var("input", node)

	// log.Tracef("graphql query: %s %s", query, string(b))

	// run request functions
	for _, optionFunc := range opts {
		optionFunc(req)
	}

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}

	if err := c.RetryRun(ctx, req, &respData, 3); err != nil {
		return nil, err
	}

	res := MutationResult{
		NumUids: respData[customFn].NumUids,
	}

	return &res, nil
}

// CustomNodeMutationWithoutInput uses the custom API
func (c *Client) CustomNodeMutationWithoutInput(ctx context.Context, customFn string, opts ...RequestOption) (*MutationResult, error) {

	log.Debugf("custom.. %s", customFn)

	// delete node
	query := `
	mutation ` + customFn + `Mutation {
		` + customFn + `{
			numUids
		}
	}`

	// make a request
	req := graphqlapi.NewRequest(query)

	// log.Tracef("graphql query: %s", query)

	// run request functions
	for _, optionFunc := range opts {
		optionFunc(req)
	}

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}

	if err := c.RetryRun(ctx, req, &respData, 3); err != nil {
		return nil, err
	}

	res := MutationResult{
		NumUids: respData[customFn].NumUids,
	}

	return &res, nil
}

// Healthcheck checks if the graphql server is online using the health endpoint.
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
