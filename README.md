[jaeger]: https://github.com/jaegertracing/jaeger-client-go
[baggage]: https://github.com/jaegertracing/jaeger-client-go#baggage-injection
[amqp]: https://pkg.go.dev/github.com/streadway/amqp#Publishing
[sns]: https://docs.aws.amazon.com/sns/latest/dg/SNSMessageAttributes.html
[sqs]: https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-message-metadata.html

# Span Context Lib
 
This module helps with the propagation of [Jaeger][jaeger] `SpanContext`s. It provides functions to add `SpanContext`s to AWS Lambda calls, as well as to AMQP, AWS SNS and AWS SQS messages.

## AWS Lambda
Passing `SpanContext`s to Lambda functions only works for synchronous calls, i.e. if the Lambda's invocation type is `RequestResponse` (which is the default). The reason for this limitation is that the `SpanContext` is embedded in the `ClientContext` which not send along for asynchronous calls, i.e. when the invocation type is `Event`.

Hint: Avoid excessive amounts of [baggage][baggage] since AWS limits the size of `ClientContext`.

Here's how to pass a `SpanContext` to a Lambda function:
```go
    params := &lambda.InvokeInput{
        FunctionName: aws.String(fnName),
        Payload:      []byte(payload),
    }
    err := spanctx.AddToLambdaInvokeInput(span.Context(), params)
    if err != nil { /* handle error */ }
    resp, err := lambdaClient.Invoke(params)
```

Here's how to retrieve the `SpanContext` from the Lambda handler's context argument:
```go
func main() {
    lambda.Start(func(ctx context.Context) error {
        spanCtx, err := spanctx.GetFromLambdaContext(ctx)
        if err != nil { /* handle error */ }
        ...
    })
}
```

## AMQP
`SpanContext`s are added to the [headers of AMQP message][amqp].

Here's how to add a `SpanContext` to an AMQP message:
```go
message := amqp.Publishing{
    ContentType: msgMimeType,
    Body:        msgBody,
}
err := spanctx.AddToAMQPPublishing(span.Context(), &message)
if err != nil { /* handle error */ }
err = channel.Publish(
    exchangeName,
    routingKey,
    isMandatory,
    isImmediate,
    message,
)
```

Here's how to retrieve a `SpanContext` from an AMQP message:
```go
delivery, more := <- msgChan
if !more { /* deal with closed channel */ }
spanCtx, err := spanctx.GetFromAMQPDelivery(delivery)
if err != nil { /* handle error */ }
```


## AWS SNS
`SpanContext`s are added to the [SNS message attributes][sns].

Note that these message attributes are not send, if `PublishInput.MessageStructure` is set to `json` (i.e. if you want to send different strings to different subscription types).

Here's how to add a `SpanContext` to an SNS message:
```go
params := &sns.PublishInput{
    Message: aws.String(message),
    QueueUrl:    aws.String(queueURL),
}
err := spanctx.AddToSNSPublishInput(span.Context(), params)
if err != nil { /* handle error */ }
resp, err := sqsClient.SendMessage(params)
```

Here's how to retrieve a `SpanContext` from an SNS message:
```go
func main() {
    lambda.Start(func(ctx context.Context, event events.SNSEvent) error {
        spanCtx, err := spanctx.GetFromSNSEvent(event)
        if err != nil { /* handle error */ }
        ...
    })
}
```


## AWS SQS
`SpanContext`s are added to the [SQS message attributes][sqs].

Here's how to add a `SpanContext` to an SQS message:
```go
params := &sqs.SendMessageInput{
    MessageBody: aws.String(message),
    QueueUrl:    aws.String(queueURL),
}
err := spanctx.AddToSQSMessageInput(span.Context(), params)
if err != nil { /* handle error */ }
resp, err := sqsClient.SendMessage(params)
```

Here's how to retrieve a `SpanContext` from an SQS message:
```go
func main() {
    lambda.Start(func(ctx context.Context, event events.SQSEvent) error {
        spanCtx, err := spanctx.GetFromSQSEvent(event)
        ...
    })
}
```
