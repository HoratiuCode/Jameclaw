// Package browserbridge coordinates the local Chrome extension with agent tools.
package browserbridge

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Command struct {
	ID     string         `json:"id"`
	Action string         `json:"action"`
	Args   map[string]any `json:"args"`
}

type Result struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

type client struct {
	queue    chan Command
	lastSeen time.Time
}

type Bridge struct {
	mu      sync.Mutex
	clients map[string]*client
	active  string
	pending map[string]chan Result
	nextID  uint64
}

func New() *Bridge {
	return &Bridge{clients: make(map[string]*client), pending: make(map[string]chan Result)}
}

var Default = New()

func (b *Bridge) Next(ctx context.Context, clientID string) (*Command, error) {
	if clientID == "" {
		return nil, fmt.Errorf("browser client id is required")
	}
	b.mu.Lock()
	c := b.clients[clientID]
	if c == nil {
		c = &client{queue: make(chan Command, 8)}
		b.clients[clientID] = c
	}
	c.lastSeen = time.Now()
	b.active = clientID
	b.mu.Unlock()

	select {
	case command := <-c.queue:
		return &command, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *Bridge) Dispatch(ctx context.Context, action string, args map[string]any) (Result, error) {
	b.mu.Lock()
	c := b.clients[b.active]
	if c == nil || time.Since(c.lastSeen) > 30*time.Second {
		b.mu.Unlock()
		return Result{}, fmt.Errorf("no active Chrome extension tab is connected; open or refresh a normal webpage in Chrome")
	}
	b.nextID++
	id := fmt.Sprintf("browser-%d", b.nextID)
	response := make(chan Result, 1)
	b.pending[id] = response
	b.mu.Unlock()

	command := Command{ID: id, Action: action, Args: args}
	select {
	case c.queue <- command:
	case <-ctx.Done():
		b.removePending(id)
		return Result{}, ctx.Err()
	}

	select {
	case result := <-response:
		return result, nil
	case <-ctx.Done():
		b.removePending(id)
		return Result{}, ctx.Err()
	case <-time.After(20 * time.Second):
		b.removePending(id)
		return Result{}, fmt.Errorf("Chrome extension did not respond; keep the target tab open and refresh it")
	}
}

func (b *Bridge) Complete(id string, result Result) {
	b.mu.Lock()
	response := b.pending[id]
	delete(b.pending, id)
	b.mu.Unlock()
	if response != nil {
		response <- result
	}
}

func (b *Bridge) removePending(id string) {
	b.mu.Lock()
	delete(b.pending, id)
	b.mu.Unlock()
}
