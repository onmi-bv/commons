package graphql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/onmi-bv/commons/internal/config"
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
}

// LoadConfig loads the graphql host parameters from environment
func LoadConfig(ctx context.Context, cFile string, prefix string) (Config, error) {
	c := Config{}

	log.Debugf("# GraphQL config... ")
	log.Debugf("GraphQL URI: %v", c.Host)
	log.Debugf("GraphQL auth enabled: %v", c.AuthEnabled)

	if c.AuthSecret != "" {
		log.Debugf("GraphQL secret: %v", "***")
	} else {
		log.Debugf("GraphQL secret: %v", "<empty>")
	}

	log.Debugln("...")

	if err := config.ReadConfig(cFile, prefix, &c); err != nil {
		return c, err
	}

	return c, err
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
func UpsertNode(ctx context.Context, config Config, node Node) (uid string, err error) {
	log.Tracef("saving.. node: %v %v", node.DType(), node.GetID())

	// update record
	res, err := UpdateNode(ctx, config, node)
	if err != nil {
		log.Errorf("could not update node: %v", err)
		return node.GetID(), err
	}

	// if no record was updated, add it
	if res.NumUids == 0 {
		res, err = AddNode(ctx, config, node)
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
func UpdateNode(ctx context.Context, config Config, node Node) (*MutationResult, error) {
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

	log.Tracef("graphQL query: %v", query)

	b, _ := json.MarshalIndent(node.Patch(), "  ", "  ")
	log.Trace(string(b))

	// create a client (safe to share across requests)
	client := graphql.NewClient(config.Host)

	// make a request
	req := graphql.NewRequest(query)

	// set any variables
	// req.Var("id", node.xid())
	// req.Var("remove", "{}")
	req.Var("set", node.Patch())

	// set header fields
	req.Header.Set("Cache-Control", "no-cache")

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

// AddNode uses the Add<type> API to add a new node.
func AddNode(ctx context.Context, config Config, node Node) (*MutationResult, error) {
	log.Debugf("adding.. node: %v %v", node.DType(), node.GetID())

	if node.GetID() == "" {
		return nil, fmt.Errorf("addNode requires XID in node")
	}

	dtype := node.DType()

	// save node
	query := `
	mutation add` + dtype + `Mutation ($set: Add` + dtype + `Input!) {
		add` + dtype + `(input: [$set]){
			numUids
		}
	}`

	log.Tracef("graphQL query: %v", query)

	b, _ := json.MarshalIndent(node, "  ", "  ")
	log.Trace(string(b))

	// create a client (safe to share across requests)
	client := graphql.NewClient(config.Host)

	// make a request
	req := graphql.NewRequest(query)

	// set any variables
	// req.Var("id", node.xid())
	// req.Var("remove", "{}")
	req.Var("set", node)

	// set header fields
	req.Header.Set("Cache-Control", "no-cache")

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

	// log.Infof("Result: %v, res: %+v", respData, res)
	return &res, nil
}
