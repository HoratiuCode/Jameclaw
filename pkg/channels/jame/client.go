package jame

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/channels"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/identity"
	"github.com/sipeed/jameclaw/pkg/logger"
)

// JameClientChannel connects to a remote Jame Protocol WebSocket server.
type JameClientChannel struct {
	*channels.BaseChannel
	config config.JameClientConfig
	conn   *jameConn
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewJameClientChannel creates a new Jame Protocol client channel.
func NewJameClientChannel(
	cfg config.JameClientConfig,
	messageBus *bus.MessageBus,
) (*JameClientChannel, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("jame_client url is required")
	}

	base := channels.NewBaseChannel("jame_client", cfg, messageBus, cfg.AllowFrom)

	return &JameClientChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Start dials the remote server and begins reading.
func (c *JameClientChannel) Start(ctx context.Context) error {
	logger.InfoC("jame_client", "Starting Jame Client channel")
	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.dial(); err != nil {
		c.cancel()
		return fmt.Errorf("jame_client initial connect: %w", err)
	}

	c.SetRunning(true)
	go c.reconnectLoop()

	logger.InfoCF("jame_client", "Connected", map[string]any{"url": c.config.URL})
	return nil
}

// Stop closes the connection.
func (c *JameClientChannel) Stop(ctx context.Context) error {
	logger.InfoC("jame_client", "Stopping Jame Client channel")
	c.SetRunning(false)
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Lock()
	if c.conn != nil {
		c.conn.close()
	}
	c.mu.Unlock()
	logger.InfoC("jame_client", "Jame Client channel stopped")
	return nil
}

func (c *JameClientChannel) dial() error {
	header := http.Header{}
	if c.config.Token != "" {
		header.Set("Authorization", "Bearer "+c.config.Token)
	}

	ws, resp, err := websocket.DefaultDialer.DialContext(c.ctx, c.config.URL, header)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return err
	}

	connCtx, connCancel := context.WithCancel(c.ctx)

	pc := &jameConn{
		id:        uuid.New().String(),
		conn:      ws,
		sessionID: c.config.SessionID,
		cancel:    connCancel,
	}
	if pc.sessionID == "" {
		pc.sessionID = uuid.New().String()
	}

	c.mu.Lock()
	c.conn = pc
	c.mu.Unlock()

	go c.readLoop(connCtx, pc)
	return nil
}

// reconnectLoop re-dials when the connection drops.
func (c *JameClientChannel) reconnectLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.Lock()
		pc := c.conn
		c.mu.Unlock()

		if pc == nil || pc.closed.Load() {
			backoff := 5 * time.Second
			logger.InfoC("jame_client", "Reconnecting...")
			if err := c.dial(); err != nil {
				logger.WarnCF("jame_client", "Reconnect failed", map[string]any{
					"error": err.Error(),
				})
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(backoff):
				}
				continue
			}
			logger.InfoC("jame_client", "Reconnected")
		}

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(1 * time.Second):
		}
	}
}

func (c *JameClientChannel) readLoop(connCtx context.Context, pc *jameConn) {
	defer pc.close()

	readTimeout := time.Duration(c.config.ReadTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}

	_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	pc.conn.SetPongHandler(func(string) error {
		return pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	})

	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	go c.pingLoop(connCtx, pc, pingInterval)

	for {
		select {
		case <-connCtx.Done():
			return
		default:
		}

		_, raw, err := pc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
			) {
				logger.DebugCF("jame_client", "Read error", map[string]any{
					"error": err.Error(),
				})
			}
			return
		}

		_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg JameMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		c.handleInbound(pc, msg)
	}
}

func (c *JameClientChannel) pingLoop(connCtx context.Context, pc *jameConn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-connCtx.Done():
			return
		case <-ticker.C:
			if pc.closed.Load() {
				return
			}
			pc.writeMu.Lock()
			err := pc.conn.WriteMessage(websocket.PingMessage, nil)
			pc.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// handleInbound processes messages from the remote server.
// In client mode the server sends message.create (responses) and the client
// sends message.send (user input). We treat message.create from the server
// as inbound user messages to feed into the agent loop.
func (c *JameClientChannel) handleInbound(pc *jameConn, msg JameMessage) {
	switch msg.Type {
	case TypePong:
		// response to our ping, ignore
	case TypeMessageCreate:
		// Server sent us a message — treat as inbound
		c.handleServerMessage(pc, msg)
	default:
		logger.DebugCF("jame_client", "Ignoring message type", map[string]any{
			"type": msg.Type,
		})
	}
}

func (c *JameClientChannel) handleServerMessage(pc *jameConn, msg JameMessage) {
	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		return
	}

	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = pc.sessionID
	}

	chatID := "jame_client:" + sessionID
	senderID := "jame-remote"
	peer := bus.Peer{Kind: "direct", ID: chatID}

	sender := bus.SenderInfo{
		Platform:    "jame_client",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("jame_client", senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	c.HandleMessage(c.ctx, peer, msg.ID, senderID, chatID, content, nil, map[string]string{
		"platform":   "jame_client",
		"session_id": sessionID,
	}, sender)
}

// Send sends a message to the remote server.
func (c *JameClientChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	c.mu.Lock()
	pc := c.conn
	c.mu.Unlock()
	if pc == nil || pc.closed.Load() {
		return channels.ErrSendFailed
	}

	outMsg := newMessage(TypeMessageSend, map[string]any{
		"content": msg.Content,
	})
	outMsg.SessionID = strings.TrimPrefix(msg.ChatID, "jame_client:")
	return pc.writeJSON(outMsg)
}

// StartTyping implements channels.TypingCapable.
func (c *JameClientChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	c.mu.Lock()
	pc := c.conn
	c.mu.Unlock()
	if pc == nil || pc.closed.Load() {
		return func() {}, nil
	}

	startMsg := newMessage(TypeTypingStart, nil)
	startMsg.SessionID = strings.TrimPrefix(chatID, "jame_client:")
	if err := pc.writeJSON(startMsg); err != nil {
		return func() {}, err
	}
	return func() {
		c.mu.Lock()
		currentPC := c.conn
		c.mu.Unlock()
		if currentPC == nil {
			return
		}
		stopMsg := newMessage(TypeTypingStop, nil)
		stopMsg.SessionID = strings.TrimPrefix(chatID, "jame_client:")
		currentPC.writeJSON(stopMsg)
	}, nil
}
