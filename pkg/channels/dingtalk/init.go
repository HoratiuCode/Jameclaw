package dingtalk

import (
	"github.com/HoratiuCode/jameclaw/pkg/bus"
	"github.com/HoratiuCode/jameclaw/pkg/channels"
	"github.com/HoratiuCode/jameclaw/pkg/config"
)

func init() {
	channels.RegisterFactory("dingtalk", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewDingTalkChannel(cfg.Channels.DingTalk, b)
	})
}
