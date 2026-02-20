// Package nudge provides the shared NudgeSession function for sending prompts
// to Gas Town agents via tmux, used by both the WebSocket and REST handlers.
package nudge

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gastownhall/tmux-adapter/internal/agents"
	"github.com/gastownhall/tmux-adapter/internal/tmux"
)

// Per-agent mutexes prevent interleaved sends to the same session.
var (
	locks   = make(map[string]*sync.Mutex)
	locksMu sync.Mutex
)

// GetLock returns the per-agent serialization mutex, creating it on first use.
func GetLock(agentName string) *sync.Mutex {
	locksMu.Lock()
	defer locksMu.Unlock()
	if _, ok := locks[agentName]; !ok {
		locks[agentName] = &sync.Mutex{}
	}
	return locks[agentName]
}

// Session sends a prompt to an agent's tmux session.
// Full sequence: literal text → 500ms paste pause → Escape → Enter (3x retry) → SIGWINCH wake.
// The caller must hold GetLock(agent.Name) before calling.
func Session(ctrl *tmux.ControlMode, agent agents.Agent, prompt string) error {
	session := agent.Name

	// 1. Send text in literal mode
	if err := ctrl.SendKeysLiteral(session, prompt); err != nil {
		return fmt.Errorf("send literal: %w", err)
	}

	// 2. Wait 500ms for paste to complete
	time.Sleep(500 * time.Millisecond)

	// 3. Send Escape (clears vim mode / any partial input state)
	if err := ctrl.SendKeysRaw(session, "Escape"); err != nil {
		return fmt.Errorf("send Escape: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	// 4. Send Enter with 3x retry, 200ms backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(200 * time.Millisecond)
		}
		if err := ctrl.SendKeysRaw(session, "Enter"); err != nil {
			lastErr = err
			continue
		}

		// 5. Wake detached sessions via SIGWINCH resize dance
		if !agent.Attached {
			if err := ctrl.ResizePane(session, "-1"); err != nil {
				log.Printf("nudge(%s): wake shrink failed: %v", session, err)
			}
			time.Sleep(50 * time.Millisecond)
			if err := ctrl.ResizePane(session, "+1"); err != nil {
				log.Printf("nudge(%s): wake restore failed: %v", session, err)
			}
		}

		return nil
	}

	if lastErr != nil {
		return fmt.Errorf("failed to send Enter after 3 attempts: %w", lastErr)
	}
	return fmt.Errorf("failed to send Enter after 3 attempts")
}
