package cloudevents

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	cepubsub "github.com/cloudevents/sdk-go/protocol/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/pkg/errors"
)

// Client defines the cloudevent client
type Client struct{ cloudevents.Client }

// Protocol for cloud event
type Protocol string

// CEProtocols to implement
const (
	HTTPProtocol   Protocol = "http"
	PubSubProtocol Protocol = "pubsub"
)

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

	ce, err := cloudevents.NewClientObserved(protocol, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent client, %v", err)
	}

	return Client{ce}, nil
}

// PubSub creates and initilizes cloudevent with pubsub protocol.
func PubSub(ctx context.Context, opts ...cepubsub.Option) (c Client, err error) {

	if len(opts) == 0 {
		opts = append(opts, cepubsub.WithTopicIDFromDefaultEnv())
		opts = append(opts, cepubsub.WithProjectIDFromDefaultEnv())
	}

	protocol, err := cepubsub.New(ctx, opts...)
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent pubsub protocol, %v", err)
	}

	ce, err := cloudevents.NewClientObserved(protocol, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent client, %v", err)
	}

	return Client{ce}, nil
}

// EventarcToEvent converts event in Eventarc format to ce-event.
func EventarcToEvent(ctx context.Context, e *event.Event) (*event.Event, error) {

	// PubSubMessage is the payload of a Pub/Sub event.
	pm := struct {
		Message      pubsub.Message
		Subscription string `json:"subscription"`
	}{}

	if err := e.DataAs(&pm); err != nil {
		return nil, errors.Wrapf(err, "Error while extracting pubsub message")
	}

	m := cepubsub.NewMessage(&pm.Message)

	return binding.ToEvent(ctx, m)
}

// PubSubToEvent converts bytes in pubsub format to ce-event.
func PubSubToEvent(ctx context.Context, b []byte) (*event.Event, error) {

	// PubSubMessage is the payload of a Pub/Sub event.
	pm := struct {
		Message      pubsub.Message
		Subscription string `json:"subscription"`
	}{}

	if err := json.Unmarshal(b, &pm); err != nil {
		return nil, errors.Wrapf(err, "Error while extracting pubsub message")
	}

	m := cepubsub.NewMessage(&pm.Message)

	return binding.ToEvent(ctx, m)
}
