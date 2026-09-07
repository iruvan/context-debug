package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	contextdebug "github.com/iruvan/context-debug"
	"github.com/nsqio/go-nsq"
)

const (
	defaultTopic     = "test"
	defaultChannel   = "consumer"
	republishTopic   = "re-publish-test"
	republishChannel = "consumer"
)

var producer *nsq.Producer

// subscription is one topic/channel worth of NSQ consumption. Add an entry
// here for every topic the service needs to consume — main() takes care of
// starting and stopping all of them the same way. handler receives the same
// base context passed to main, unlike an HTTP handler there's no per-message
// context supplied by the framework, so this is where one would branch on
// the message payload and call contextdebug.New to opt a message into debug
// mode before calling into Dep methods (see ../../dependency.go).
type subscription struct {
	topic   string
	channel string
	handler func(context.Context, *nsq.Message) error
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	lookupdAddr := envOr("NSQ_LOOKUPD_HTTP_ADDR", "localhost:4161")
	nsqdAddr := envOr("NSQ_NSQD_TCP_ADDR", "localhost:4150")

	cfg := nsq.NewConfig()

	var err error
	producer, err = nsq.NewProducer(nsqdAddr, cfg)
	if err != nil {
		log.Fatalf("new producer: %v", err)
	}
	defer producer.Stop()

	subscriptions := []subscription{
		{
			topic:   envOr("NSQ_TOPIC", defaultTopic),
			channel: envOr("NSQ_CHANNEL", defaultChannel),
			handler: handleMessage,
		},
		{
			topic:   republishTopic,
			channel: republishChannel,
			handler: handleRepublishedMessage,
		},
	}

	consumers := make([]*nsq.Consumer, 0, len(subscriptions))
	for _, sub := range subscriptions {
		consumer, err := nsq.NewConsumer(sub.topic, sub.channel, cfg)
		if err != nil {
			log.Fatalf("new consumer for topic=%s channel=%s: %v", sub.topic, sub.channel, err)
		}
		consumer.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
			return sub.handler(ctx, msg)
		}))

		if err := consumer.ConnectToNSQLookupd(lookupdAddr); err != nil {
			log.Fatalf("connect to nsqlookupd %s: %v", lookupdAddr, err)
		}
		log.Printf("consumer started: topic=%s channel=%s lookupd=%s", sub.topic, sub.channel, lookupdAddr)

		consumers = append(consumers, consumer)
	}

	<-ctx.Done()

	log.Println("shutting down")
	for _, consumer := range consumers {
		consumer.Stop()
	}
	for _, consumer := range consumers {
		<-consumer.StopChan
	}
}

// handleMessage is the wiring point for context-debug: branch on the message
// payload (e.g. a "debug" field) and call contextdebug.New(ctx) to opt this
// message into debug mode, call into the same Dep methods the HTTP example
// uses, then do something with contextdebug.Snapshot(ctx) — log it, persist
// it, republish it, etc.
func handleMessage(ctx context.Context, msg *nsq.Message) (err error) {
	start := time.Now()
	var payload map[string]any
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("failed to decode message body %q: %v", msg.Body, err)
		return nil // malformed message; don't requeue
	}

	log.Printf("received message: %+v", payload)

	if payload["debug"] == true {
		ctx = contextdebug.New(ctx)
	}

	contextdebug.Collect(ctx, contextdebug.Entry{
		Name:       "Consumer.handleMessage",
		Request:    payload,
		Response:   nil,
		Error:      err,
		DurationMs: int(time.Since(start).Milliseconds()),
	})

	payload["type"] = "republished"
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to marshal payload: %v", err)
		return nil
	}

	err = producer.Publish(republishTopic, body)
	if err != nil {
		log.Printf("failed to publish message: %v", err)
		return nil
	}

	if payload["debug"] == true {
		debug := contextdebug.Snapshot(ctx)
		log.Printf("debug: %+v", debug)
	}

	log.Printf("published message: %+v", payload)

	return nil
}

// handleRepublishedMessage confirms a message made it around the loop:
// consumed from defaultTopic, republished onto republishTopic by
// handleMessage, and picked up again here.
func handleRepublishedMessage(ctx context.Context, msg *nsq.Message) (err error) {
	start := time.Now()

	var payload map[string]any
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("failed to decode republished message body %q: %v", msg.Body, err)
		return nil // malformed message; don't requeue
	}

	contextdebug.Collect(ctx, contextdebug.Entry{
		Name:       "Consumer.handleRepublishedMessage",
		Request:    payload,
		Response:   nil,
		Error:      err,
		DurationMs: int(time.Since(start).Milliseconds()),
	})

	log.Printf("received republished message: %+v", payload)

	if payload["debug"] == true {
		debug := contextdebug.Snapshot(ctx)
		log.Printf("debug 2: %+v", debug)
	}

	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
