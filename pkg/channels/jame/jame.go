package jame

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/channels"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/identity"
	"github.com/sipeed/jameclaw/pkg/logger"
	"github.com/sipeed/jameclaw/pkg/media"
	"github.com/sipeed/jameclaw/pkg/utils"
)

const maxJameInboundMediaBytes = 25 << 20
const maxJameOutboundMediaBytes = 25 << 20

// jameConn represents a single WebSocket connection.
type jameConn struct {
	id        string
	conn      *websocket.Conn
	sessionID string
	writeMu   sync.Mutex
	closed    atomic.Bool
	cancel    context.CancelFunc // cancels per-connection goroutines (e.g. pingLoop)
}

// writeJSON sends a JSON message to the connection with write locking.
func (pc *jameConn) writeJSON(v any) error {
	if pc.closed.Load() {
		return fmt.Errorf("connection closed")
	}
	pc.writeMu.Lock()
	defer pc.writeMu.Unlock()
	return pc.conn.WriteJSON(v)
}

// close closes the connection.
func (pc *jameConn) close() {
	if pc.closed.CompareAndSwap(false, true) {
		if pc.cancel != nil {
			pc.cancel()
		}
		pc.conn.Close()
	}
}

// JameChannel implements the native Jame Protocol WebSocket channel.
// It serves as the reference implementation for all optional capability interfaces.
type JameChannel struct {
	*channels.BaseChannel
	config             config.JameConfig
	upgrader           websocket.Upgrader
	connections        sync.Map // connID → *jameConn
	connCount          atomic.Int32
	ctx                context.Context
	cancel             context.CancelFunc
	activePlaceholders sync.Map // chatID → messageID (string)
}

// NewJameChannel creates a new Jame Protocol channel.
func NewJameChannel(cfg config.JameConfig, messageBus *bus.MessageBus) (*JameChannel, error) {
	if cfg.Token() == "" {
		return nil, fmt.Errorf("jame token is required")
	}

	base := channels.NewBaseChannel("jame", cfg, messageBus, cfg.AllowFrom)

	allowOrigins := cfg.AllowOrigins
	checkOrigin := func(r *http.Request) bool {
		if len(allowOrigins) == 0 {
			return true // allow all if not configured
		}
		origin := r.Header.Get("Origin")
		for _, allowed := range allowOrigins {
			if allowed == "*" || allowed == origin {
				return true
			}
		}
		return false
	}

	return &JameChannel{
		BaseChannel: base,
		config:      cfg,
		upgrader: websocket.Upgrader{
			CheckOrigin:     checkOrigin,
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}, nil
}

// Start implements Channel.
func (c *JameChannel) Start(ctx context.Context) error {
	logger.InfoC("jame", "Starting Jame Protocol channel")
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.SetRunning(true)
	logger.InfoC("jame", "Jame Protocol channel started")
	return nil
}

// Stop implements Channel.
func (c *JameChannel) Stop(ctx context.Context) error {
	logger.InfoC("jame", "Stopping Jame Protocol channel")
	c.SetRunning(false)

	// Close all connections
	c.connections.Range(func(key, value any) bool {
		if pc, ok := value.(*jameConn); ok {
			pc.close()
		}
		c.connections.Delete(key)
		return true
	})

	if c.cancel != nil {
		c.cancel()
	}

	logger.InfoC("jame", "Jame Protocol channel stopped")
	return nil
}

// WebhookPath implements channels.WebhookHandler.
func (c *JameChannel) WebhookPath() string { return "/jame/" }

// ServeHTTP implements http.Handler for the shared HTTP server.
func (c *JameChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/jame")

	switch {
	case path == "/ws" || path == "/ws/":
		c.handleWebSocket(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Send implements Channel — sends a message to the appropriate WebSocket connection.
func (c *JameChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	outMsg := newMessage(TypeMessageCreate, map[string]any{
		"content": msg.Content,
	})

	return c.broadcastToSession(msg.ChatID, outMsg)
}

// SendMedia implements channels.MediaSender for the web console protocol.
func (c *JameChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("jame media store is not configured: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		localPath, meta, err := store.ResolveWithMeta(part.Ref)
		if err != nil {
			return fmt.Errorf("resolve jame media: %w", channels.ErrSendFailed)
		}

		info, err := os.Stat(localPath)
		if err != nil || info.IsDir() {
			return fmt.Errorf("stat jame media: %w", channels.ErrSendFailed)
		}
		if info.Size() > maxJameOutboundMediaBytes {
			return fmt.Errorf("jame media too large: %d bytes: %w", info.Size(), channels.ErrSendFailed)
		}

		data, err := os.ReadFile(localPath)
		if err != nil {
			return fmt.Errorf("read jame media: %w", channels.ErrSendFailed)
		}

		filename := part.Filename
		if filename == "" {
			filename = meta.Filename
		}
		if filename == "" {
			filename = filepath.Base(localPath)
		}

		contentType := part.ContentType
		if contentType == "" {
			contentType = meta.ContentType
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		outMsg := newMessage(TypeMediaCreate, map[string]any{
			"caption":      part.Caption,
			"content_type": contentType,
			"data":         base64.StdEncoding.EncodeToString(data),
			"filename":     filename,
			"kind":         jameMediaKind(part.Type, filename, contentType),
		})
		if err := c.broadcastToSession(msg.ChatID, outMsg); err != nil {
			return err
		}
	}
	return nil
}

// EditMessage implements channels.MessageEditor.
func (c *JameChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	outMsg := newMessage(TypeMessageUpdate, map[string]any{
		"message_id": messageID,
		"content":    content,
	})
	return c.broadcastToSession(chatID, outMsg)
}

// StartTyping implements channels.TypingCapable.
func (c *JameChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	startMsg := newMessage(TypeTypingStart, nil)
	if err := c.broadcastToSession(chatID, startMsg); err != nil {
		return func() {}, err
	}
	return func() {
		stopMsg := newMessage(TypeTypingStop, nil)
		c.broadcastToSession(chatID, stopMsg)
	}, nil
}

// PublishActivity sends a high-level, non-sensitive work milestone directly to
// Web Console clients. Unlike an editable placeholder, each update is its own
// event, so rapid agent progress is never collapsed into a single message edit.
func (c *JameChannel) PublishActivity(ctx context.Context, chatID, label string) error {
	outMsg := newMessage(TypeActivityUpdate, map[string]any{
		"label": label,
	})
	return c.broadcastToSession(chatID, outMsg)
}

// SendPlaceholder implements channels.PlaceholderCapable.
// It sends a placeholder message via the Jame Protocol that will later be
// edited to the actual response via EditMessage (channels.MessageEditor).
func (c *JameChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}
	return c.sendPlaceholderForced(ctx, chatID)
}

func (c *JameChannel) sendPlaceholderForced(ctx context.Context, chatID string) (string, error) {
	text := c.config.Placeholder.Text
	if text == "" {
		text = "Thinking... 💭"
	}

	msgID := uuid.New().String()
	outMsg := newMessage(TypeMessageCreate, map[string]any{
		"content":    text,
		"message_id": msgID,
	})

	if err := c.broadcastToSession(chatID, outMsg); err != nil {
		return "", err
	}

	c.activePlaceholders.Store(chatID, msgID)
	return msgID, nil
}

// BeginStream implements channels.StreamingCapable.
func (c *JameChannel) BeginStream(ctx context.Context, chatID string) (bus.Streamer, error) {
	msgIDVal, ok := c.activePlaceholders.Load(chatID)
	var msgID string
	if ok {
		msgID = msgIDVal.(string)
	} else {
		var err error
		msgID, err = c.sendPlaceholderForced(ctx, chatID)
		if err != nil {
			return nil, err
		}
	}

	return &jameStreamer{
		channel:   c,
		chatID:    chatID,
		messageID: msgID,
	}, nil
}

type jameStreamer struct {
	channel   *JameChannel
	chatID    string
	messageID string
}

func (s *jameStreamer) Update(ctx context.Context, content string) error {
	return s.channel.EditMessage(ctx, s.chatID, s.messageID, content)
}

func (s *jameStreamer) Finalize(ctx context.Context, content string) error {
	s.channel.activePlaceholders.Delete(s.chatID)
	return s.channel.EditMessage(ctx, s.chatID, s.messageID, content)
}

func (s *jameStreamer) Cancel(ctx context.Context) {
	s.channel.activePlaceholders.Delete(s.chatID)
	_ = s.channel.EditMessage(ctx, s.chatID, s.messageID, "*Request canceled.*")
}

// broadcastToSession sends a message to all connections with a matching session.
func (c *JameChannel) broadcastToSession(chatID string, msg JameMessage) error {
	// chatID format: "jame:<sessionID>"
	sessionID := strings.TrimPrefix(chatID, "jame:")
	msg.SessionID = sessionID

	var sent bool
	c.connections.Range(func(key, value any) bool {
		pc, ok := value.(*jameConn)
		if !ok {
			return true
		}
		if pc.sessionID == sessionID {
			if err := pc.writeJSON(msg); err != nil {
				logger.DebugCF("jame", "Write to connection failed", map[string]any{
					"conn_id": pc.id,
					"error":   err.Error(),
				})
			} else {
				sent = true
			}
		}
		return true
	})

	if !sent {
		return fmt.Errorf("no active connections for session %s: %w", sessionID, channels.ErrSendFailed)
	}
	return nil
}

// handleWebSocket upgrades the HTTP connection and manages the WebSocket lifecycle.
func (c *JameChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !c.IsRunning() {
		http.Error(w, "channel not running", http.StatusServiceUnavailable)
		return
	}

	// Authenticate
	if !c.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check connection limit
	maxConns := c.config.MaxConnections
	if maxConns <= 0 {
		maxConns = 100
	}
	if int(c.connCount.Load()) >= maxConns {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	// Echo the matched subprotocol back so the browser accepts the upgrade.
	var responseHeader http.Header
	if proto := c.matchedSubprotocol(r); proto != "" {
		responseHeader = http.Header{"Sec-WebSocket-Protocol": {proto}}
	}

	conn, err := c.upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		logger.ErrorCF("jame", "WebSocket upgrade failed", map[string]any{
			"error": err.Error(),
		})
		return
	}

	// Determine session ID from query param or generate one
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	pc := &jameConn{
		id:        uuid.New().String(),
		conn:      conn,
		sessionID: sessionID,
	}

	c.connections.Store(pc.id, pc)
	c.connCount.Add(1)

	logger.InfoCF("jame", "WebSocket client connected", map[string]any{
		"conn_id":    pc.id,
		"session_id": sessionID,
	})

	go c.readLoop(pc)
}

// authenticate checks the request for a valid token:
//  1. Authorization: Bearer <token> header
//  2. Sec-WebSocket-Protocol "token.<value>" (for browsers that can't set headers)
//  3. Query parameter "token" (only when AllowTokenQuery is on)
func (c *JameChannel) authenticate(r *http.Request) bool {
	token := c.config.Token()
	if token == "" {
		return false
	}

	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		if after == token {
			return true
		}
	}

	// Check Sec-WebSocket-Protocol subprotocol ("token.<value>")
	if c.matchedSubprotocol(r) != "" {
		return true
	}

	// Check query parameter only when explicitly allowed
	if c.config.AllowTokenQuery {
		if r.URL.Query().Get("token") == token {
			return true
		}
	}

	return false
}

// matchedSubprotocol returns the "token.<value>" subprotocol that matches
// the configured token, or "" if none do.
func (c *JameChannel) matchedSubprotocol(r *http.Request) string {
	token := c.config.Token()
	for _, proto := range websocket.Subprotocols(r) {
		if after, ok := strings.CutPrefix(proto, "token."); ok && after == token {
			return proto
		}
	}
	return ""
}

// readLoop reads messages from a WebSocket connection.
func (c *JameChannel) readLoop(pc *jameConn) {
	defer func() {
		pc.close()
		c.connections.Delete(pc.id)
		c.connCount.Add(-1)
		logger.InfoCF("jame", "WebSocket client disconnected", map[string]any{
			"conn_id":    pc.id,
			"session_id": pc.sessionID,
		})
	}()

	readTimeout := time.Duration(c.config.ReadTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}

	_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
	pc.conn.SetPongHandler(func(appData string) error {
		_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	// Start ping ticker
	pingInterval := time.Duration(c.config.PingInterval) * time.Second
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	go c.pingLoop(pc, pingInterval)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, rawMsg, err := pc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.DebugCF("jame", "WebSocket read error", map[string]any{
					"conn_id": pc.id,
					"error":   err.Error(),
				})
			}
			return
		}

		_ = pc.conn.SetReadDeadline(time.Now().Add(readTimeout))

		var msg JameMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			errMsg := newError("invalid_message", "failed to parse message")
			pc.writeJSON(errMsg)
			continue
		}

		c.handleMessage(pc, msg)
	}
}

// pingLoop sends periodic ping frames to keep the connection alive.
func (c *JameChannel) pingLoop(pc *jameConn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
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

// handleMessage processes an inbound Jame Protocol message.
func (c *JameChannel) handleMessage(pc *jameConn, msg JameMessage) {
	switch msg.Type {
	case TypePing:
		pong := newMessage(TypePong, nil)
		pong.ID = msg.ID
		pc.writeJSON(pong)

	case TypeMessageSend:
		c.handleMessageSend(pc, msg)

	case TypeMediaSend:
		c.handleMediaSend(pc, msg)

	default:
		errMsg := newError("unknown_type", fmt.Sprintf("unknown message type: %s", msg.Type))
		pc.writeJSON(errMsg)
	}
}

// handleMessageSend processes an inbound message.send from a client.
func (c *JameChannel) handleMessageSend(pc *jameConn, msg JameMessage) {
	content, _ := msg.Payload["content"].(string)
	if strings.TrimSpace(content) == "" {
		errMsg := newError("empty_content", "message content is empty")
		pc.writeJSON(errMsg)
		return
	}

	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = pc.sessionID
	}

	chatID := "jame:" + sessionID
	senderID := "jame-user"

	peer := bus.Peer{Kind: "direct", ID: "jame:" + sessionID}

	metadata := map[string]string{
		"platform":   "jame",
		"session_id": sessionID,
		"conn_id":    pc.id,
	}

	logger.DebugCF("jame", "Received message", map[string]any{
		"session_id": sessionID,
		"preview":    truncate(content, 50),
	})

	sender := bus.SenderInfo{
		Platform:    "jame",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("jame", senderID),
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	c.HandleMessage(c.ctx, peer, msg.ID, senderID, chatID, content, nil, metadata, sender)
}

// handleMediaSend processes inbound browser media such as recorded voice clips,
// images, files, and documents.
func (c *JameChannel) handleMediaSend(pc *jameConn, msg JameMessage) {
	payload := msg.Payload
	if payload == nil {
		pc.writeJSON(newError("missing_payload", "media payload is required"))
		return
	}

	content, _ := payload["content"].(string)
	data, _ := payload["data"].(string)
	contentType, _ := payload["content_type"].(string)
	filename, _ := payload["filename"].(string)
	kind, _ := payload["kind"].(string)
	if strings.TrimSpace(data) == "" {
		pc.writeJSON(newError("missing_media", "media data is required"))
		return
	}

	if filename = sanitizeJameFilename(filename); filename == "" {
		filename = defaultJameFilename(kind, contentType)
	}
	if contentType = strings.TrimSpace(contentType); contentType == "" {
		contentType = "application/octet-stream"
	}

	mediaBytes, err := decodeJameBase64(data)
	if err != nil {
		pc.writeJSON(newError("invalid_media", "media data must be base64 encoded"))
		return
	}
	if len(mediaBytes) == 0 {
		pc.writeJSON(newError("empty_media", "media data is empty"))
		return
	}
	if len(mediaBytes) > maxJameInboundMediaBytes {
		pc.writeJSON(newError("media_too_large", "media file is too large"))
		return
	}

	store := c.GetMediaStore()
	if store == nil {
		pc.writeJSON(newError("media_store_unavailable", "media store is not available"))
		return
	}

	if err = os.MkdirAll(media.TempDir(), 0o700); err != nil {
		pc.writeJSON(newError("media_temp_failed", "could not prepare media storage"))
		return
	}
	tmp, err := os.CreateTemp(media.TempDir(), "jame-voice-*"+filepath.Ext(filename))
	if err != nil {
		pc.writeJSON(newError("media_temp_failed", "could not create media file"))
		return
	}
	tmpPath := tmp.Name()
	if _, err = tmp.Write(mediaBytes); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		pc.writeJSON(newError("media_write_failed", "could not write media file"))
		return
	}
	if err = tmp.Close(); err != nil {
		os.Remove(tmpPath)
		pc.writeJSON(newError("media_write_failed", "could not close media file"))
		return
	}

	sessionID := msg.SessionID
	if sessionID == "" {
		sessionID = pc.sessionID
	}
	chatID := "jame:" + sessionID
	messageID := msg.ID
	if messageID == "" {
		messageID = uuid.NewString()
	}
	scope := channels.BuildMediaScope("jame", chatID, messageID)
	ref, err := store.Store(tmpPath, media.MediaMeta{
		Filename:      filename,
		ContentType:   contentType,
		Source:        "jame",
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		os.Remove(tmpPath)
		pc.writeJSON(newError("media_store_failed", "could not store media"))
		return
	}

	if strings.TrimSpace(content) == "" {
		content = jameMediaAnnotation(filename, contentType)
	} else {
		content = strings.TrimSpace(content) + "\n" + jameMediaAnnotation(filename, contentType)
	}

	senderID := "jame-user"
	peer := bus.Peer{Kind: "direct", ID: "jame:" + sessionID}
	metadata := map[string]string{
		"platform":     "jame",
		"session_id":   sessionID,
		"conn_id":      pc.id,
		"content_type": contentType,
	}
	sender := bus.SenderInfo{
		Platform:    "jame",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("jame", senderID),
	}
	if !c.IsAllowedSender(sender) {
		return
	}

	logger.DebugCF("jame", "Received media message", map[string]any{
		"session_id": sessionID,
		"filename":   filename,
		"bytes":      len(mediaBytes),
	})

	c.HandleMessage(c.ctx, peer, messageID, senderID, chatID, content, []string{ref}, metadata, sender)
}

func defaultJameFilename(kind, contentType string) string {
	if strings.HasPrefix(strings.ToLower(contentType), "audio/") || kind == "audio" {
		return "voice.webm"
	}
	if strings.HasPrefix(strings.ToLower(contentType), "image/") || kind == "image" {
		return "image"
	}
	return "document"
}

func jameMediaAnnotation(filename, contentType string) string {
	switch {
	case utils.IsAudioFile(filename, contentType):
		return "[voice]"
	case strings.HasPrefix(strings.ToLower(contentType), "image/"):
		return "[image: " + filename + "]"
	default:
		return "[file: " + filename + "]"
	}
}

func jameMediaKind(partType, filename, contentType string) string {
	switch {
	case partType != "":
		return partType
	case strings.HasPrefix(strings.ToLower(contentType), "image/"):
		return "image"
	case utils.IsAudioFile(filename, contentType):
		return "audio"
	case strings.HasPrefix(strings.ToLower(contentType), "video/"):
		return "video"
	default:
		return "file"
	}
}

func decodeJameBase64(data string) ([]byte, error) {
	if comma := strings.Index(data, ","); comma >= 0 && strings.Contains(data[:comma], "base64") {
		data = data[comma+1:]
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(data))
}

func sanitizeJameFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', 0:
			return -1
		default:
			return r
		}
	}, name)
}

// truncate truncates a string to maxLen runes.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
