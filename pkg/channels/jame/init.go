package jame

import (
	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/channels"
	"github.com/sipeed/jameclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("jame", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewJameChannel(cfg.Channels.Jame, b)
	})
	channels.RegisterFactory("jame_client", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewJameClientChannel(cfg.Channels.JameClient, b)
	})
}
