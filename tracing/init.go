package tracing

import (
	"fmt"
	"io"

	opentracing "github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// Init returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func Init(service string) (opentracing.Tracer, io.Closer, error) {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse Jaeger env vars: %v", err)
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = service
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, nil, err
	}

	opentracing.SetGlobalTracer(tracer)

	return tracer, closer, nil
}
