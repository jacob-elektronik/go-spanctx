package spanctx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

const (
	maxClientContextBytes = 3583
	traceIDKey            = "uber-trace-id"
	baggagePrefix         = "uberctx-"
)

var (
	ErrorTooMuchBaggage            = errors.New("span baggage data too large")
	ErrorUnsupportedInvocationType = errors.New("unsupported invocation type")
	ErrorJaegerSpanContextExpected = errors.New("span context implementation not supported")
)

func hasReqRespInvocationType(input *lambda.InvokeInput) bool {
	if input.InvocationType == nil {
		return true // since RequestResponse is the default
	}
	t := *input.InvocationType
	return t == "" || t == lambda.InvocationTypeRequestResponse
}

func AddToLambdaInvokeInput(spanCtx opentracing.SpanContext, input *lambda.InvokeInput) error {
	if spanCtx == nil || input == nil {
		return nil
	}
	if !hasReqRespInvocationType(input) {
		return ErrorUnsupportedInvocationType
	}

	jaegerSpanCtx, ok := spanCtx.(jaeger.SpanContext)
	if !ok {
		return ErrorJaegerSpanContextExpected
	}

	clientContext := struct {
		Custom map[string]string `json:"custom"`
	}{Custom: make(map[string]string)}

	clientContext.Custom[traceIDKey] = jaegerSpanCtx.String()
	jaegerSpanCtx.ForeachBaggageItem(func(k, v string) bool {
		clientContext.Custom[baggagePrefix+k] = v
		return true
	})

	ccJSON, err := json.Marshal(clientContext)
	if err != nil {
		return err
	}

	enc := base64.StdEncoding
	if enc.EncodedLen(len(ccJSON)) > maxClientContextBytes {
		return ErrorTooMuchBaggage
	}
	input.ClientContext = aws.String(enc.EncodeToString(ccJSON))
	return nil
}

func GetFromLambdaContext(ctx context.Context) (opentracing.SpanContext, error) {
	if ctx == nil {
		return nil, nil
	}
	lambdaContext, ok := lambdacontext.FromContext(ctx)
	if !ok || len(lambdaContext.ClientContext.Custom) == 0 {
		return nil, nil
	}

	tmpCtx, err := jaeger.ContextFromString(lambdaContext.ClientContext.Custom[traceIDKey])
	if err != nil {
		return nil, err
	}

	baggage := make(map[string]string, len(lambdaContext.ClientContext.Custom)-1)
	for k, v := range lambdaContext.ClientContext.Custom {
		if strings.HasPrefix(k, baggagePrefix) {
			baggage[strings.TrimPrefix(k, baggagePrefix)] = v
		}
	}

	return jaeger.NewSpanContext(
		tmpCtx.TraceID(),
		tmpCtx.SpanID(),
		tmpCtx.ParentID(),
		tmpCtx.IsSampled(),
		baggage,
	), nil
}
