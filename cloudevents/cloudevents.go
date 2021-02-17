package cloudevents

import (
	"context"
	"fmt"

	cepubsub "github.com/cloudevents/sdk-go/protocol/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
)

// Client defines the cloudevent client
type Client struct{ cloudevents.Client }

// CloudEvents creates and initilizes cloudevent with http protocol.
func CloudEvents(ctx context.Context, port int) (ce cloudevents.Client, err error) {

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

// HTTP creates and initilizes cloudevent with HTTP protocol.
func HTTP(ctx context.Context, port int) (c Client, err error) {

	protocol, err := cehttp.New(cloudevents.WithPort(port))
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent http protocol, %v", err)
	}
	ce, err := cloudevents.NewClientObserved(protocol,
		cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent client, %v", err)
	}
	return Client{ce}, nil
}

// PubSub creates and initilizes cloudevent with pubsub protocol.
func PubSub(ctx context.Context, opts ...cepubsub.Option) (c Client, err error) {

	protocol, err := cepubsub.New(ctx, opts...)
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent pubsub protocol, %v", err)
	}
	ce, err := cloudevents.NewClientObserved(protocol,
		cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent client, %v", err)
	}
	return Client{ce}, nil
}
