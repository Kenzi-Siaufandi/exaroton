package exaroton

import (
	"context"
	"encoding/json"
)

// SubscribeConsole starts the console stream. tail selects how many past lines are replayed.
func (s *Session) SubscribeConsole(ctx context.Context, tail int) error {
	data, err := json.Marshal(map[string]int{"tail": tail})
	if err != nil {
		return err
	}

	return s.Send(ctx, wsEnvelope{
		Stream: "console",
		Type:   "start",
		Data:   data,
	})
}

// UnsubscribeConsole stops the console stream.
func (s *Session) UnsubscribeConsole(ctx context.Context) error {
	s.markStreamStopped("console")

	return s.Send(ctx, wsEnvelope{
		Stream: "console",
		Type:   "stop",
	})
}

// ExecuteCommand sends a console command over the websocket console stream.
func (s *Session) ExecuteCommand(ctx context.Context, cmd string) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	return s.Send(ctx, wsEnvelope{
		Stream: "console",
		Type:   "command",
		Data:   data,
	})
}

// SubscribeTick starts the tick stream, which reports the in-game tick speed.
// It is only available for Minecraft Java Edition 1.16+. SubscribeTick checks
// the server software version via REST and fails fast on older servers; when
// the version is unknown it proceeds and lets the server decide.
func (s *Session) SubscribeTick(ctx context.Context) error {
	if err := s.requireMinVersion(ctx, 1, 16, 0, "tick"); err != nil {
		return err
	}
	return s.Send(ctx, wsEnvelope{
		Stream: "tick",
		Type:   "start",
	})
}

// UnsubscribeTick stops the tick stream.
func (s *Session) UnsubscribeTick(ctx context.Context) error {
	s.markStreamStopped("tick")

	return s.Send(ctx, wsEnvelope{
		Stream: "tick",
		Type:   "stop",
	})
}

// SubscribeStats starts the stats stream.
func (s *Session) SubscribeStats(ctx context.Context) error {
	return s.Send(ctx, wsEnvelope{
		Stream: "stats",
		Type:   "start",
	})
}

// UnsubscribeStats stops the stats stream.
func (s *Session) UnsubscribeStats(ctx context.Context) error {
	s.markStreamStopped("stats")
	return s.Send(ctx, wsEnvelope{
		Stream: "stats",
		Type:   "stop",
	})
}

// SubscribeHeap starts the heap stream.
func (s *Session) SubscribeHeap(ctx context.Context) error {
	return s.Send(ctx, wsEnvelope{
		Stream: "heap",
		Type:   "start",
	})
}

// UnsubscribeHeap stops the heap stream.
func (s *Session) UnsubscribeHeap(ctx context.Context) error {
	s.markStreamStopped("heap")
	return s.Send(ctx, wsEnvelope{
		Stream: "heap",
		Type:   "stop",
	})
}

// SubscribeManagement starts the management stream for the Minecraft server
// management protocol introduced in 1.21.9. It allows controlling server
// settings without restarts and receiving event notifications; exaroton
// handles the server.properties setup automatically. SubscribeManagement
// checks the server software version via REST and fails fast on older
// servers; when the version is unknown it proceeds and lets the server decide.
func (s *Session) SubscribeManagement(ctx context.Context) error {
	if err := s.requireMinVersion(ctx, 1, 21, 9, "management"); err != nil {
		return err
	}
	return s.Send(ctx, wsEnvelope{
		Stream: "management",
		Type:   "start",
	})
}

// UnsubscribeManagement stops the management stream.
func (s *Session) UnsubscribeManagement(ctx context.Context) error {
	s.markStreamStopped("management")
	return s.Send(ctx, wsEnvelope{
		Stream: "management",
		Type:   "stop",
	})
}
