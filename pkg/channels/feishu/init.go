package feishu

import (
	"github.com/HoratiuCode/jameclaw/pkg/bus"
	"github.com/HoratiuCode/jameclaw/pkg/channels"
	"github.com/HoratiuCode/jameclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("feishu", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewFeishuChannel(cfg.Channels.Feishu, b)
	})
}
