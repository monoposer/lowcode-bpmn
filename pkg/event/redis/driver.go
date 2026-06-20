package redis

import (
	"fmt"

	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/event/transport"
)

func init() {
	transport.Register(driver{})
}

type driver struct{}

func (driver) Name() string { return transport.DriverRedis }

func (driver) NewConsumer(stream event.Stream, cfg transport.Config) (event.Consumer, error) {
	url := cfg.BrokerURL
	if url == "" {
		return nil, fmt.Errorf("EVENT_BROKER_URL or EVENT_REDIS_URL required when EVENT_CONSUMER=redis")
	}
	return New(url, stream, cfg.Destination(stream))
}
