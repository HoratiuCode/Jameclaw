package jame

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/channels"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/media"
)

func TestJameChannelSendMediaBroadcastsMediaCreate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.JameConfig{ReadTimeout: 10, PingInterval: 60}
	cfg.SetToken("test-token")
	ch, err := NewJameChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	if err = ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	srv := httptest.NewServer(ch)
	defer srv.Close()

	ws, _, err := websocket.DefaultDialer.Dial(
		wsURL(srv.URL)+"/jame/ws?session_id=sess-media",
		http.Header{"Authorization": {"Bearer test-token"}},
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	imgPath := filepath.Join(t.TempDir(), "photo.png")
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	if err = os.WriteFile(imgPath, png, 0o600); err != nil {
		t.Fatalf("write png: %v", err)
	}
	ref, err := store.Store(imgPath, media.MediaMeta{
		Filename:    "photo.png",
		ContentType: "image/png",
		Source:      "test",
	}, "test")
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	err = ch.SendMedia(ctx, bus.OutboundMediaMessage{
		ChatID: "jame:sess-media",
		Parts: []bus.MediaPart{{
			Type:        "image",
			Ref:         ref,
			Caption:     "caption",
			Filename:    "sent.png",
			ContentType: "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("SendMedia: %v", err)
	}

	var got JameMessage
	if err = ws.ReadJSON(&got); err != nil {
		t.Fatalf("read message: %v", err)
	}

	if got.Type != TypeMediaCreate {
		t.Fatalf("type = %q, want %q", got.Type, TypeMediaCreate)
	}
	if got.SessionID != "sess-media" {
		t.Fatalf("session_id = %q, want sess-media", got.SessionID)
	}
	if got.Payload["kind"] != "image" {
		t.Fatalf("kind = %v, want image", got.Payload["kind"])
	}
	if got.Payload["filename"] != "sent.png" {
		t.Fatalf("filename = %v, want sent.png", got.Payload["filename"])
	}
	if got.Payload["content_type"] != "image/png" {
		t.Fatalf("content_type = %v, want image/png", got.Payload["content_type"])
	}
	if got.Payload["caption"] != "caption" {
		t.Fatalf("caption = %v, want caption", got.Payload["caption"])
	}
	if got.Payload["data"] != base64.StdEncoding.EncodeToString(png) {
		t.Fatal("media payload data did not match stored file")
	}
}

func TestJameChannelSendMediaRequiresMediaStore(t *testing.T) {
	cfg := config.JameConfig{}
	cfg.SetToken("test-token")
	ch, err := NewJameChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	ch.SetRunning(true)

	err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "jame:sess-media",
		Parts:  []bus.MediaPart{{Ref: "media://missing"}},
	})
	if !errors.Is(err, channels.ErrSendFailed) {
		t.Fatalf("SendMedia() error = %v, want ErrSendFailed", err)
	}
}

func TestPublishMemoryChangeBroadcastsDedicatedEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.JameConfig{ReadTimeout: 10, PingInterval: 60}
	cfg.SetToken("test-token")
	ch, err := NewJameChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatal(err)
	}
	if err = ch.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer ch.Stop(ctx)

	srv := httptest.NewServer(ch)
	defer srv.Close()
	ws, _, err := websocket.DefaultDialer.Dial(
		wsURL(srv.URL)+"/jame/ws?session_id=sess-memory",
		http.Header{"Authorization": {"Bearer test-token"}},
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	want := "Jame updated long-term memory for future conversations."
	if err = ch.PublishMemoryChange(ctx, "jame:sess-memory", want); err != nil {
		t.Fatalf("PublishMemoryChange: %v", err)
	}
	var got JameMessage
	if err = ws.ReadJSON(&got); err != nil {
		t.Fatalf("read message: %v", err)
	}
	if got.Type != TypeMemoryChanged {
		t.Fatalf("type = %q, want %q", got.Type, TypeMemoryChanged)
	}
	if got.SessionID != "sess-memory" {
		t.Fatalf("session_id = %q, want sess-memory", got.SessionID)
	}
	if got.Payload["summary"] != want {
		t.Fatalf("summary = %q, want %q", got.Payload["summary"], want)
	}
}
