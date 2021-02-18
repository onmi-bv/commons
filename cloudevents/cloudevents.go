package cloudevents

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

// NewMessageFromPubSubRequest converts pubsub request to a ce binding message.
func NewMessageFromPubSubRequest(ctx context.Context, r *http.Request) (*cepubsub.Message, error) {

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while ready request body")
	}
	defer r.Body.Close()

	// PubSubMessage is the payload of a Pub/Sub event.
	pm := struct {
		Message      pubsub.Message
		Subscription string `json:"subscription"`
	}{}

	if err := json.Unmarshal(b, &pm); err != nil {
		return nil, errors.Wrapf(err, "Error while extracting pubsub message")
	}

	return cepubsub.NewMessage(&pm.Message), nil
}

// NewEventFromHTTPRequest converts http request body to ce-event.
func NewEventFromHTTPRequest(ctx context.Context, r *http.Request, p Protocol) (e *event.Event, err error) {

	var m binding.MessageReader

	switch p {
	case HTTPProtocol:
		m = cehttp.NewMessageFromHttpRequest(r)
	case PubSubProtocol:
		if m, err = NewMessageFromPubSubRequest(ctx, r); err != nil {
			return
		}
	}

	return binding.ToEvent(ctx, m)
}
