package gateway

// StreamEvent is a typed presentation event emitted by the agent/runtime and
// rendered by channel adapters or web clients.
type StreamEvent struct {
	Type       string         `json:"type"`
	SessionID  string         `json:"session_id,omitempty"`
	MessageID  string         `json:"message_id,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`
	Content    string         `json:"content,omitempty"`
	Final      bool           `json:"final,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	OccurredAt int64          `json:"occurred_at,omitempty"`
}

const (
	StreamEventMessageChunk = "message.chunk"
	StreamEventMessageStop  = "message.stop"
	StreamEventToolStart    = "tool.start"
	StreamEventToolProgress = "tool.progress"
	StreamEventToolFinish   = "tool.finish"
	StreamEventNotice       = "notice"
)

type StreamSink interface {
	DispatchStreamEvent(event StreamEvent) error
}

type StreamDispatcher struct {
	sinks []StreamSink
}

func NewStreamDispatcher(sinks ...StreamSink) *StreamDispatcher {
	return &StreamDispatcher{sinks: append([]StreamSink(nil), sinks...)}
}

func (d *StreamDispatcher) Dispatch(event StreamEvent) []error {
	if d == nil {
		return nil
	}
	var errs []error
	for _, sink := range d.sinks {
		if sink == nil {
			continue
		}
		if err := sink.DispatchStreamEvent(event); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
