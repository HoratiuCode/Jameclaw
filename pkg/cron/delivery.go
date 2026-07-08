package cron

import "context"

type DeliveryTarget struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
}

type DeliveryFunc func(ctx context.Context, target DeliveryTarget, content string) error

func DeliverJobResult(ctx context.Context, job *CronJob, result string, deliver DeliveryFunc) error {
	if job == nil || deliver == nil || !job.Payload.Deliver {
		return nil
	}
	content := result
	if content == "" {
		content = job.Payload.Message
	}
	return deliver(ctx, DeliveryTarget{
		Channel: job.Payload.Channel,
		ChatID:  job.Payload.To,
	}, content)
}
