package spanctx

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/opentracing/opentracing-go"
)

type sqsInjectAttributeCarrier map[string]*sqs.MessageAttributeValue

func (c sqsInjectAttributeCarrier) ForeachKey(handler func(key, val string) error) error {
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

func (c sqsInjectAttributeCarrier) Set(key, val string) {
	c[key].DataType = aws.String("String")
	c[key].StringValue = aws.String(val)
}

func AddToSQSMessageInput(spanCtx opentracing.SpanContext, pubInput *sqs.SendMessageInput) error {
	if spanCtx == nil {
		return nil
	}
	c := sqsInjectAttributeCarrier(pubInput.MessageAttributes)
	return opentracing.GlobalTracer().Inject(spanCtx, opentracing.TextMap, c)
}

type sqsExtractAttributeCarrier map[string]events.SQSMessageAttribute

func (c sqsExtractAttributeCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, val := range c {
		if val.DataType != "String" {
			continue
		}
		if err := handler(k, *val.StringValue); err != nil {
			return err
		}
	}
	return nil
}

func GetFromSQSEvent(event events.SQSEvent) (opentracing.SpanContext, error) {
	c := sqsExtractAttributeCarrier(event.Records[0].MessageAttributes)
	return opentracing.GlobalTracer().Extract(opentracing.TextMap, c)
}
