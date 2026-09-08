package serve

import (
	"context"
	"fmt"
	"strings"

	"reasonix/internal/control"
	"reasonix/internal/provider"
)

// SpawnSubSession implements control.SubSessionSpawner for the serving
// frontend (qwen daemon session bridge): an independent session is built with
// the same boot options as the parent, the prompt is delivered, and
// completion="first-turn" waits for its first turn and returns the output.
// "sent" returns on delivery and keeps the session alive under serve.
func (s *Server) SpawnSubSession(ctx context.Context, prompt, completion string) (string, error) {
	parent, ok := s.ctl().(*control.Controller)
	if !ok || parent == nil {
		return "", fmt.Errorf("create_sub_session: parent controller unavailable")
	}
	ref := currentModelRef(s.ctl())
	ctrl, _, err := s.buildTagged(ctx, ref, false)
	if err != nil {
		return "", fmt.Errorf("create_sub_session: build sub-session: %w", err)
	}
	link := ctrl.SessionPath()
	if completion == "sent" {
		ctrl.Submit(prompt)
		s.keepSubSession(link, ctrl)
		return fmt.Sprintf("Sub-session %s spawned; prompt delivered.", link), nil
	}
	// first-turn (default): run one synchronous turn and return its output.
	if err := ctrl.RunTurn(ctx, prompt); err != nil {
		ctrl.Close()
		return "", fmt.Errorf("create_sub_session: first turn failed: %w", err)
	}
	out := lastAssistantText(ctrl.History())
	ctrl.Close()
	if strings.TrimSpace(out) == "" {
		out = "(no output)"
	}
	return fmt.Sprintf("Sub-session %s first-turn result:\n\n%s", link, out), nil
}

// lastAssistantText returns the most recent assistant message body.
func lastAssistantText(history []provider.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" && strings.TrimSpace(history[i].Content) != "" {
			return history[i].Content
		}
	}
	return ""
}

// keepSubSession retains a sent-mode sub-session until server shutdown so the
// background work it started is not torn down with the request.
func (s *Server) keepSubSession(link string, ctrl *control.Controller) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	if s.subSessions == nil {
		s.subSessions = map[string]*control.Controller{}
	}
	s.subSessions[link] = ctrl
}

// closeSubSessions tears down retained sent-mode sub-sessions.
func (s *Server) closeSubSessions() {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for link, ctrl := range s.subSessions {
		ctrl.Close()
		delete(s.subSessions, link)
	}
}
