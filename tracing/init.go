package tracing

import (
	"context"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/onmi-bv/commons/confighelper"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/api/global"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ExporterType defines the supported exporters
type ExporterType string

// Exporter types.
const (
	JeagerExporter      ExporterType = "jaeger"
	StackdriverExporter ExporterType = "stackdriver"
)

// Configuration ...
type Configuration struct {
	// Exporter type supported by commons
	Exporter ExporterType

	// ProjectID is the identifier of the Stackdriver
	// project the user is uploading the stats data to.
	// If not set, this will default to your "Application Default Credentials".
	// For details see: https://developers.google.com/accounts/docs/application-default-credentials.
	//
	// It will be used in the project_id label of a Stackdriver monitored
	// resource if the resource does not inherently belong to a specific
	// project, e.g. on-premise resource like k8s_container or generic_task.
	ProjectID string

	// Location is the identifier of the GCP or AWS cloud region/zone in which
	// the data for a resource is stored.
	// If not set, it will default to the location provided by the metadata server.
	//
	// It will be used in the location label of a Stackdriver monitored resource
	// if the resource does not inherently belong to a specific project, e.g.
	// on-premise resource like k8s_container or generic_task.
	Location string

	// MaxNumberOfWorkers sets the maximum number of go rountines that send requests
	// to Cloud Trace. The minimum number of workers is 1.
	MaxNumberOfWorkers int
}

// Init initializes opentelemetry. The returned Tracer is ready to use.
// The returned Exporter will be useful for flushing spans before exiting the process.
func Init(ctx context.Context, name string) (*apitrace.Tracer, error) {

	// init config params
	config := Configuration{}
	err := confighelper.ReadConfig("app.conf", "", &config)
	if err != nil {
		return nil, err
	}

	// create exporter
	var exporter trace.SpanExporter

	switch config.Exporter {
	// Create exporter for stackdriver
	case StackdriverExporter:
		exporter, err = texporter.NewExporter(
			texporter.WithContext(ctx),
			texporter.WithProjectID(string(config.ProjectID)),
			texporter.WithMaxNumberOfWorkers(config.MaxNumberOfWorkers),
		)
		if err != nil {
			return nil, errors.Wrap(err, "cannot init stackdriver exporter")
		}

	default:
		return nil, errors.New("unsupported exporter")
	}

	// Create trace provider with the exporter.
	//
	// By default it uses AlwaysSample() which samples all traces.
	// In a production environment or high QPS setup please use
	// ProbabilitySampler set at the desired probability.
	// Example:
	//   config := sdktrace.Config{DefaultSampler:sdktrace.ProbabilitySampler(0.0001)}
	//   tp, err := sdktrace.NewProvider(sdktrace.WithConfig(config), ...)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	global.SetTracerProvider(tp)

	tracer := global.TracerProvider().Tracer(name)

	return &tracer, err
}
