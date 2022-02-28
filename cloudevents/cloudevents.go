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
	cloudeventsclient "github.com/cloudevents/sdk-go/v2/client"
	"github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// Client defines the cloudevent client
type Client struct {
	Protocol Protocol
	cloudevents.Client
	receiverPort int
}

// Protocol for cloud event
type Protocol string

// CEProtocols to implement
const (
	HTTPProtocol   Protocol = "http"
	PubSubProtocol Protocol = "pubsub"
)

// StartReceiver starts an http receiver able to parse different protocols
func (c *Client) StartReceiver(ctx context.Context, fn interface{}) error {

	// Create a mux for routing incoming requests
	mux := http.NewServeMux()

	// All URLs will be handled by this function
	mux.Handle("/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			if r.Method != "POST" { /* The regular updates are sent using a POST request, deny everything else */
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			ctx, event, err := NewEventFromHTTPRequest(ctx, r, c.Protocol)
			if err != nil {
				log.Errorf("cannot convert request to a valid cloudevent: %v", err)
				http.Error(w, fmt.Sprintf("cannot convert request to a valid cloudevent: %v", err), http.StatusBadRequest)
				return
			}

			switch fn.(type) {
			case func(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result):
				fn.(func(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result))(ctx, *event)
			default:
				log.Error("unsupported receiver fn type")
			}
		},
	))

	// Create a server listening on port 8000
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.receiverPort),
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen:%+s\n", err)
		}
	}()

	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	}
}

// WithPort sets the receiver port for StartReceiver func.
func (c *Client) WithPort(port int) *Client {
	c.receiverPort = port
	return c
}

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
func HTTP(ctx context.Context, port int, opts ...cloudeventsclient.Option) (c Client, err error) {

	protocol, err := cehttp.New(cloudevents.WithPort(port))
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent http protocol, %v", err)
	}

	opts = append(opts, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())

	ce, err := cloudevents.NewClientObserved(protocol, opts...)
	if err != nil {
		return c, fmt.Errorf("failed to create cloudevent client, %v", err)
	}

	return Client{HTTPProtocol, ce, port}, nil
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

	return Client{PubSubProtocol, ce, 0}, nil
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
func NewMessageFromPubSubRequest(ctx context.Context, r *http.Request) (context.Context, *cepubsub.Message, error) {

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ctx, nil, errors.Wrapf(err, "Error while ready request body")
	}
	defer r.Body.Close()

	// PubSubMessage is the payload of a Pub/Sub event.
	pm := struct {
		Message      pubsub.Message
		Subscription string `json:"subscription"`
	}{}

	if err := json.Unmarshal(b, &pm); err != nil {
		return ctx, nil, errors.Wrapf(err, "Error while extracting pubsub message")
	}

	if spanContext, ok := pm.Message.Attributes["spanContext"]; ok {
		var sc = struct {
			TraceID    string
			SpanID     string
			TraceFlags byte
			TraceState string
			Remote     bool
		}{}
		json.Unmarshal([]byte(spanContext), &sc)

		var spanContextConfig = trace.SpanContextConfig{}
		spanContextConfig.TraceID, _ = trace.TraceIDFromHex(sc.TraceID)
		spanContextConfig.SpanID, _ = trace.SpanIDFromHex(sc.SpanID)
		spanContextConfig.TraceFlags = 01
		spanContextConfig.Remote = sc.Remote

		spanContext := trace.NewSpanContext(spanContextConfig)
		ctx = trace.ContextWithRemoteSpanContext(ctx, spanContext)
	}

	return ctx, cepubsub.NewMessage(&pm.Message), nil
}

// NewEventFromHTTPRequest converts http request body to ce-event.
func NewEventFromHTTPRequest(ctx context.Context, r *http.Request, p Protocol) (ectx context.Context, e *event.Event, err error) {

	var m binding.MessageReader

	switch p {
	case HTTPProtocol:
		m = cehttp.NewMessageFromHttpRequest(r)
	case PubSubProtocol:
		if ctx, m, err = NewMessageFromPubSubRequest(ctx, r); err != nil {
			return
		}
	}

	event, err := binding.ToEvent(ctx, m)

	return ctx, event, err
}
