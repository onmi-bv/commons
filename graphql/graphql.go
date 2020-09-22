package graphql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/onmi-bv/commons/confighelper"
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

// Config defines graphql host parameters.
type Config struct {
	Host        string `mapstructure:"HOST"`
	AuthEnabled bool   `mapstructure:"AUTH_ENABLED"`
	AuthSecret  string `mapstructure:"SECRET"`
	HealthPath  string `mapstructure:"HEALTH_PATH"` //TODO: add healthcheck
}

// LoadConfig loads the graphql host parameters from environment
func LoadConfig(ctx context.Context, cFile string, prefix string) (Config, error) {
	c := Config{}

	if err := confighelper.ReadConfig(cFile, prefix, &c); err != nil {
		return c, err
	}

	log.Debugf("# GraphQL config... ")
	log.Debugf("GraphQL URI: %v", c.Host)
	log.Debugf("GraphQL auth enabled: %v", c.AuthEnabled)

	if c.AuthSecret != "" {
		log.Debugf("GraphQL secret: %v", "***")
	} else {
		log.Debugf("GraphQL secret: %v", "<empty>")
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

// UpsertNode adds or updates a node.
func (c *Config) UpsertNode(ctx context.Context, node Node) (uid string, err error) {
	log.Tracef("saving.. node: %v %v", node.DType(), node.GetID())

	// update record
	res, err := c.UpdateNode(ctx, node)
	if err != nil {
		log.Errorf("could not update node: %v", err)
		return node.GetID(), err
	}

	// if no record was updated, add it
	if res.NumUids == 0 {
		res, err = c.AddNode(ctx, []Node{node})
		if err != nil {
			log.Errorf("could not add node: %v", err)
			return node.GetID(), err
		}
		if res.NumUids != 1 {
			log.Warningf("Could not add new node: %v %v", node.DType(), node.GetID())
		}
	}

	if res.NumUids > 1 {
		log.Warningf("Inconsistent db state, multiple entries for node after add mu: %v %v, numUids=%v", node.DType(), node.GetID(), res.NumUids)
	}

	return node.GetID(), err
}

// UpdateNode uses the update<type> GraphQL API to update a node.
func (c *Config) UpdateNode(ctx context.Context, node Node) (*MutationResult, error) {
	log.Debugf("updating node: %v %v", node.DType(), node.GetID())

	if node.GetID() == "" {
		return nil, fmt.Errorf("updateNode requires XID in node")
	}

	dtype := node.DType()

	// save node
	query := `
	mutation update` + dtype + `Mutation ($set: ` + dtype + `Patch) {
		update` + dtype + `(input: {
			filter: {` + node.Key() + `: {eq: "` + node.GetID() + `"}},
			set: $set
		}){
			numUids
		}
	}`

	log.Tracef("graphql query: %v", query)

	b, _ := json.MarshalIndent(node.Patch(), "  ", "  ")
	log.Trace("graphql node: %v", string(b))

	// create a client (safe to share across requests)
	client := graphql.NewClient(c.Host)

	// make a request
	req := graphql.NewRequest(query)

	// set any variables
	req.Var("set", node.Patch())

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}
	if err := client.Run(ctx, req, &respData); err != nil {
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
func (c *Config) AddNode(ctx context.Context, node []Node) (*MutationResult, error) {

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

	log.Tracef("graphql query: %v", query)

	b, _ := json.MarshalIndent(node, "  ", "  ")
	log.Tracef("graphql node: %v", string(b))

	// create a client (safe to share across requests)
	client := graphql.NewClient(c.Host)

	// make a request
	req := graphql.NewRequest(query)

	// set any variables
	req.Var("set", node)

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}

	if err := client.Run(ctx, req, &respData); err != nil {
		return nil, err
	}

	res := MutationResult{
		NumUids: respData[`add`+dtype].NumUids,
	}

	return &res, nil
}

// DeleteNodeByID uses the Delete<type> API to delete a node.
// Action is unreversable and should be used with care.
func (c *Config) DeleteNodeByID(ctx context.Context, _type string, key string, id string) (*MutationResult, error) {

	log.Debugf("deleting.. %v nodes: %v", _type, id)

	if _type == "" || id == "" || key == "" {
		return nil, fmt.Errorf("DeleteNode requires _type, key and id")
	}

	// delete node
	query := `
	mutation delete` + _type + `Mutation ($id: String!) {
		delete` + _type + `(filter: { ` + key + `: { eq: $id }}){
			numUids
		}
	}`

	log.Tracef("graphql query: %v", query)

	// create a client (safe to share across requests)
	client := graphql.NewClient(c.Host)

	// make a request
	req := graphql.NewRequest(query)

	// set any variables
	req.Var("id", id)

	// run it and capture the response
	var respData map[string]struct {
		NumUids int
	}

	if err := client.Run(ctx, req, &respData); err != nil {
		return nil, err
	}

	res := MutationResult{
		NumUids: respData[`delete`+_type].NumUids,
	}

	return &res, nil
}
