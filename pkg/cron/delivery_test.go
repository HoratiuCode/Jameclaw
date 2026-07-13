package cron

import (
	"context"
	"testing"
)

func TestDeliverJobResultRequiresApproval(t *testing.T) {
	job := &CronJob{
		Payload: CronPayload{
			Message: "hello",
			Deliver: true,
			Channel: "telegram",
			To:      "chat-1",
		},
	}

	called := false
	err := DeliverJobResult(context.Background(), job, "", func(context.Context, DeliveryTarget, string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("DeliverJobResult() error = %v", err)
	}
	if called {
		t.Fatal("delivery should not run without approval")
	}
}

func TestDeliverJobResultWithApproval(t *testing.T) {
	job := &CronJob{
		Payload: CronPayload{
			Message:          "fallback",
			Deliver:          true,
			DeliveryApproved: true,
			Channel:          "telegram",
			To:               "chat-1",
		},
	}

	var gotTarget DeliveryTarget
	var gotContent string
	err := DeliverJobResult(context.Background(), job, "done", func(_ context.Context, target DeliveryTarget, content string) error {
		gotTarget = target
		gotContent = content
		return nil
	})
	if err != nil {
		t.Fatalf("DeliverJobResult() error = %v", err)
	}
	if gotTarget.Channel != "telegram" || gotTarget.ChatID != "chat-1" {
		t.Fatalf("target = %+v, want telegram/chat-1", gotTarget)
	}
	if gotContent != "done" {
		t.Fatalf("content = %q, want done", gotContent)
	}
}
