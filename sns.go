package spanctx

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/opentracing/opentracing-go"
)

type snsInjectAttributeCarrier map[string]*sns.MessageAttributeValue

func (c snsInjectAttributeCarrier) ForeachKey(handler func(key, val string) error) error {
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

func (c snsInjectAttributeCarrier) Set(key, val string) {
	c[key] = &sns.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(val),
	}
}

func AddToSNSPublishInput(spanCtx opentracing.SpanContext, pubInput *sns.PublishInput) error {
	if spanCtx == nil || pubInput == nil {
		return nil
	}
	if pubInput.MessageAttributes == nil {
		pubInput.MessageAttributes = make(snsInjectAttributeCarrier)
	}
	c := snsInjectAttributeCarrier(pubInput.MessageAttributes)
	return opentracing.GlobalTracer().Inject(spanCtx, opentracing.TextMap, c)
}

type snsExtractAttributeCarrier map[string]interface{}

func (c snsExtractAttributeCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, raw := range c {
		attrValueMap, ok := raw.(map[string]interface{})
		if !ok || attrValueMap["Type"] != "String" {
			continue
		}
		v := attrValueMap["Value"].(string)
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}

func GetFromSNSEvent(event events.SNSEvent) (opentracing.SpanContext, error) {
	c := snsExtractAttributeCarrier(event.Records[0].SNS.MessageAttributes)
	return opentracing.GlobalTracer().Extract(opentracing.TextMap, c)
}
