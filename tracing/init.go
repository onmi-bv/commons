package tracing

import (
	"context"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/go-logr/logr"
	"github.com/onmi-bv/commons/confighelper"
	"github.com/pkg/errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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
	Exporter ExporterType `mapstructure:"EXPORTER"`

	// ProjectID is the identifier of the Stackdriver
	// project the user is uploading the stats data to.
	// If not set, this will default to your "Application Default Credentials".
	// For details see: https://developers.google.com/accounts/docs/application-default-credentials.
	//
	// It will be used in the project_id label of a Stackdriver monitored
	// resource if the resource does not inherently belong to a specific
	// project, e.g. on-premise resource like k8s_container or generic_task.
	ProjectID string `mapstructure:"PROJECT_ID"`

	// Location is the identifier of the GCP or AWS cloud region/zone in which
	// the data for a resource is stored.
	// If not set, it will default to the location provided by the metadata server.
	//
	// It will be used in the location label of a Stackdriver monitored resource
	// if the resource does not inherently belong to a specific project, e.g.
	// on-premise resource like k8s_container or generic_task.
	Location string `mapstructure:"LOCATION"`

	// MaxNumberOfWorkers sets the maximum number of go rountines that send requests
	// to Cloud Trace. The minimum number of workers is 1.
	MaxNumberOfWorkers int `mapstructure:"MAX_NUMBER_OF_WORKERS"`

	// Tracer logger
	Logger logr.Logger
}

// Tracer type
type (
	Tracer        = trace.Tracer
	TraceProvider = *sdktrace.TracerProvider
)

var TP *sdktrace.TracerProvider

// Init initializes opentelemetry. The returned Tracer is ready to use.
// The returned Exporter will be useful for flushing spans before exiting the process.
func Init(ctx context.Context, name string, config Configuration) (Tracer, TraceProvider, error) {

	tracer := otel.Tracer(name)

	// init config params
	err := confighelper.ReadConfig("app.conf", "tracing", &config)
	if err != nil {
		return tracer, nil, err
	}

	// create exporter
	var exporter sdktrace.SpanExporter

	switch config.Exporter {
	case StackdriverExporter: // Create exporter for stackdriver
		exporter, err = texporter.New(
			texporter.WithContext(ctx),
			texporter.WithProjectID(config.ProjectID),
		)
		if err != nil {
			return tracer, nil, errors.Wrap(err, "cannot init stackdriver exporter")
		}

	default:
		return tracer, nil, errors.New("unsupported exporter")
	}

	// Create trace provider with the exporter.
	//
	// By default it uses AlwaysSample() which samples all traces.
	// In a production environment or high QPS setup please use
	// ProbabilitySampler set at the desired probability.
	// Example:
	//   config := sdktrace.Config{DefaultSampler:sdktrace.ProbabilitySampler(0.0001)}
	//   tp, err := sdktrace.NewProvider(sdktrace.WithConfig(config), ...)
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetLogger(config.Logger)

	tracer = TP.Tracer(name)

	return tracer, tp, err
}
