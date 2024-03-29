package sdformatter

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-stack/stack"
	"github.com/sirupsen/logrus"
)

var skipTimestamp bool

type severity string

const (
	severityDebug    severity = "DEBUG"
	severityInfo     severity = "INFO"
	severityWarning  severity = "WARNING"
	severityError    severity = "ERROR"
	severityCritical severity = "CRITICAL"
	severityAlert    severity = "ALERT"
)

var levelsToSeverity = map[logrus.Level]severity{
	logrus.DebugLevel: severityDebug,
	logrus.InfoLevel:  severityInfo,
	logrus.WarnLevel:  severityWarning,
	logrus.ErrorLevel: severityError,
	logrus.FatalLevel: severityCritical,
	logrus.PanicLevel: severityAlert,
}

type serviceContext struct {
	Service string `json:"service,omitempty"`
	Version string `json:"version,omitempty"`
}

type reportLocation struct {
	FilePath     string `json:"file,omitempty"`
	LineNumber   int    `json:"line,omitempty"`
	FunctionName string `json:"function,omitempty"`
}

type context struct {
}

type entry struct {
	Timestamp      string                 `json:"timestamp,omitempty"`
	ServiceContext *serviceContext        `json:"serviceContext,omitempty"`
	Message        string                 `json:"message,omitempty"`
	Severity       severity               `json:"severity,omitempty"`
	HTTPRequest    map[string]interface{} `json:"httpRequest,omitempty"`
	Trace          interface{}            `json:"logging.googleapis.com/trace,omitempty"`
	SpanID         interface{}            `json:"logging.googleapis.com/spanId,omitempty"`
	TraceSampled   interface{}            `json:"logging.googleapis.com/trace_sampled,omitempty"`
	ReportLocation *reportLocation        `json:"logging.googleapis.com/sourceLocation,omitempty"`
	Operation      interface{}            `json:"logging.googleapis.com/operation,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

// Formatter implements Stackdriver formatting for logrus.
type Formatter struct {
	Service   string
	Version   string
	StackSkip []string
}

// Option lets you configure the Formatter.
type Option func(*Formatter)

// WithService lets you configure the service name used for error reporting.
func WithService(n string) Option {
	return func(f *Formatter) {
		f.Service = n
	}
}

// WithVersion lets you configure the service version used for error reporting.
func WithVersion(v string) Option {
	return func(f *Formatter) {
		f.Version = v
	}
}

// WithStackSkip lets you configure which packages should be skipped for locating the error.
func WithStackSkip(v string) Option {
	return func(f *Formatter) {
		f.StackSkip = append(f.StackSkip, v)
	}
}

// NewFormatter returns a new Formatter.
func NewFormatter(options ...Option) *Formatter {
	fmtr := Formatter{
		StackSkip: []string{
			"github.com/sirupsen/logrus",
		},
	}
	for _, option := range options {
		option(&fmtr)
	}
	return &fmtr
}

func (f *Formatter) errorOrigin() (stack.Call, error) {
	skip := func(pkg string) bool {
		for _, skip := range f.StackSkip {
			if pkg == skip {
				return true
			}
		}
		return false
	}

	// We start at 2 to skip this call and our caller's call.
	for i := 2; ; i++ {
		c := stack.Caller(i)
		// ErrNoFunc indicates we're over traversing the stack.
		if _, err := c.MarshalText(); err != nil {
			return stack.Call{}, nil
		}
		pkg := fmt.Sprintf("%+k", c)
		// Remove vendoring from package path.
		parts := strings.SplitN(pkg, "/vendor/", 2)
		pkg = parts[len(parts)-1]
		if !skip(pkg) {
			return c, nil
		}
	}
}

// Format formats a logrus entry according to the Stackdriver specifications.
// https://cloud.google.com/logging/docs/structured-logging
func (f *Formatter) Format(e *logrus.Entry) ([]byte, error) {
	severity := levelsToSeverity[e.Level]

	ee := entry{
		Message:  e.Message,
		Severity: severity,
		Data:     e.Data,
	}

	if !skipTimestamp {
		ee.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	ee.ServiceContext = &serviceContext{
		Service: f.Service,
		Version: f.Version,
	}

	// When using WithError(), the error is sent separately, but Error
	// Reporting expects it to be a part of the message so we append it
	// instead.
	if err, ok := ee.Data["error"]; ok {
		ee.Message = fmt.Sprintf("%s: %s", e.Message, err)
		delete(ee.Data, "error")
	} else {
		ee.Message = e.Message
	}

	// Extract report location from call stack.
	if c, err := f.errorOrigin(); err == nil {
		lineNumber, _ := strconv.ParseInt(fmt.Sprintf("%d", c), 10, 64)

		ee.ReportLocation = &reportLocation{
			FilePath:     fmt.Sprintf("%+s", c),
			LineNumber:   int(lineNumber),
			FunctionName: fmt.Sprintf("%n", c),
		}
	}

	// As a convenience, when using supplying the httpRequest field, it
	// gets special care.
	if reqData, ok := ee.Data["httpRequest"]; ok {
		if req, ok := reqData.(map[string]interface{}); ok {
			ee.HTTPRequest = req
			delete(ee.Data, "httpRequest")
		}
	}
	if data, ok := ee.Data["trace"]; ok {
		ee.Trace = fmt.Sprintf("projects/%s/traces/%s", os.Getenv("GOOGLE_CLOUD_PROJECT"), data)
		delete(ee.Data, "trace")
	}
	if data, ok := ee.Data["spanId"]; ok {
		ee.SpanID = data
		delete(ee.Data, "spanId")
	}
	if data, ok := ee.Data["traceSampled"]; ok {
		ee.TraceSampled = data
		delete(ee.Data, "traceSampled")
	}
	if data, ok := ee.Data["operation"]; ok {
		ee.Operation = data
		delete(ee.Data, "operation")
	}

	b, err := json.Marshal(ee)
	if err != nil {
		return nil, err
	}

	return append(b, '\n'), nil
}
