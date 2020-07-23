package cloudevents

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
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
