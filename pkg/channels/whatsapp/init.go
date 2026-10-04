package whatsapp

import (
	"github.com/HoratiuCode/jameclaw/pkg/bus"
	"github.com/HoratiuCode/jameclaw/pkg/channels"
	"github.com/HoratiuCode/jameclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("whatsapp", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewWhatsAppChannel(cfg.Channels.WhatsApp, b)
	})
}
