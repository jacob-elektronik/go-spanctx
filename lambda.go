package spanctx

import (
	"context"
	"encoding/base64"
	"errors"

	"encoding/json"

	"strings"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

const (
	maxClientContextBytes = 3583
	traceIdKey            = "uber-trace-id"
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
	if spanCtx == nil {
		return nil
	}
	if !hasReqRespInvocationType(input) {
		return ErrorUnsupportedInvocationType
	}
	jaegerSpanCtx, ok := spanCtx.(jaeger.SpanContext)
	if !ok {
		return ErrorJaegerSpanContextExpected
	}

	clientContext := make(map[string]string)
	clientContext[traceIdKey] = jaegerSpanCtx.String()
	jaegerSpanCtx.ForeachBaggageItem(func(k, v string) bool {
		clientContext[baggagePrefix+k] = v
		return true
	})

	ccJson, err := json.Marshal(clientContext)
	if err != nil {
		return err
	}
	var ccJsonBase64 []byte
	base64.StdEncoding.Encode(ccJsonBase64, ccJson)
	if len(ccJsonBase64) > maxClientContextBytes {
		return ErrorTooMuchBaggage
	}

	input.ClientContext = aws.String(string(ccJsonBase64))
	return nil
}

func GetFromLambdaContext(ctx context.Context) (opentracing.SpanContext, error) {
	if ctx == nil {
		return nil, nil
	}
	lambdaContext, ok := lambdacontext.FromContext(ctx)
	if !ok {
		return nil, nil
	}
	if len(lambdaContext.ClientContext.Custom) == 0 {
		return nil, nil
	}
	tmpCtx, err := jaeger.ContextFromString(lambdaContext.ClientContext.Custom[traceIdKey])
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
