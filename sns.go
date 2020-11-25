package spanctx

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/opentracing/opentracing-go"
)

type snsAttributeInjectCarrier map[string]*sns.MessageAttributeValue

func (c snsAttributeInjectCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, val := range c {
		if *val.DataType != "String" {
			continue
		}
		if err := handler(k, *val.StringValue); err != nil {
			return err
		}
	}
	return nil
}

func (c snsAttributeInjectCarrier) Set(key, val string) {
	c[key].DataType = aws.String("String")
	c[key].StringValue = aws.String(val)
}

func AddToSNSPublishInput(spanCtx opentracing.SpanContext, pubInput *sns.PublishInput) error {
	if spanCtx == nil {
		return nil
	}
	c := snsAttributeInjectCarrier(pubInput.MessageAttributes)
	return opentracing.GlobalTracer().Inject(spanCtx, opentracing.TextMap, c)
}

type snsAttributeExtractCarrier map[string]interface{}

func (c snsAttributeExtractCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, val := range c {
		v, ok := val.(string)
		if !ok {
			continue
		}
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}

func GetFromSNSEvent(event events.SNSEvent) (opentracing.SpanContext, error) {
	c := snsAttributeExtractCarrier(event.Records[0].SNS.MessageAttributes)
	return opentracing.GlobalTracer().Extract(opentracing.TextMap, c)
}
