package jame

import (
	"github.com/HoratiuCode/jameclaw/pkg/bus"
	"github.com/HoratiuCode/jameclaw/pkg/channels"
	"github.com/HoratiuCode/jameclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("jame", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewJameChannel(cfg.Channels.Jame, b)
	})
	channels.RegisterFactory("jame_client", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewJameClientChannel(cfg.Channels.JameClient, b)
	})
}
