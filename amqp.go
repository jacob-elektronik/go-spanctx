package spanctx

import (
	"github.com/opentracing/opentracing-go"
	"github.com/streadway/amqp"
)

type amqpHeaderCarrier map[string]interface{}

func (c amqpHeaderCarrier) ForeachKey(handler func(key, val string) error) error {
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

func (c amqpHeaderCarrier) Set(key, val string) {
	c[key] = val
}

func AddToAMQPPublishing(spanCtx opentracing.SpanContext, publishing *amqp.Publishing) error {
	if publishing.Headers == nil {
		publishing.Headers = make(amqp.Table)
	}
	c := amqpHeaderCarrier(publishing.Headers)
	return opentracing.GlobalTracer().Inject(spanCtx, opentracing.TextMap, c)
}

func GetFromAMQPDelivery(delivery amqp.Delivery) (opentracing.SpanContext, error) {
	c := amqpHeaderCarrier(delivery.Headers)
	return opentracing.GlobalTracer().Extract(opentracing.TextMap, c)
}
