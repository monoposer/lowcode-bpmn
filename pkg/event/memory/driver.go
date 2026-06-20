package memory

import (
	"github.com/monoposer/lowcode-bpmn/pkg/event"
	"github.com/monoposer/lowcode-bpmn/pkg/event/transport"
)

func init() {
	transport.Register(driver{})
}

type driver struct{}

func (driver) Name() string { return transport.DriverMemory }

func (driver) NewConsumer(stream event.Stream, cfg transport.Config) (event.Consumer, error) {
	buf := cfg.BufferSize
	if buf <= 0 {
		buf = 512
	}
	return New(stream, buf), nil
}
