package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
)

type Transcript struct {
	Text  string
	Final bool
}

type ConversationState int

const (
	StateIdle ConversationState = iota
	StateUserSpeaking
	StateAssistantSpeaking
)

// ConversationManager buffers final transcripts and only sends them
// to the LLM when we've had a "quiet period" with no new finals.
type ConversationManager struct {
	mu          sync.Mutex
	state       ConversationState
	gracePeriod time.Duration
	pendingText strings.Builder
	commitTimer *time.Timer
	output      chan<- string
	timerActive bool
}

func NewConversationManager(gracePeriod time.Duration, out chan<- string) *ConversationManager {
	return &ConversationManager{
		state:       StateIdle,
		gracePeriod: gracePeriod,
		output:      out,
	}
}

func (cm *ConversationManager) Run(ctx context.Context, in <-chan Transcript) {
	for {
		select {
		case <-ctx.Done():
			return
		case tr, ok := <-in:
			if !ok {
				return
			}
			cm.handleTranscript(tr)
		}
	}
}

func (cm *ConversationManager) handleTranscript(tr Transcript) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.state == StateAssistantSpeaking {
		// Ignore while assistant is speaking
		return
	}

	if !tr.Final {
		// Ignore partials for turn-taking; still useful for UI captions if needed
		log.Printf("[CONV] partial transcript (ignored for turn): %q", tr.Text)
		return
	}
	if tr.Text == "" {
		return
	}

	log.Printf("[CONV] buffering transcript: %q", tr.Text)

	if cm.pendingText.Len() > 0 {
		cm.pendingText.WriteString(" ")
	}
	cm.pendingText.WriteString(strings.TrimSpace(tr.Text))
	cm.state = StateUserSpeaking

	// Reset the commit timer (grace period).
	if cm.commitTimer != nil && cm.timerActive {
		if !cm.commitTimer.Stop() {
			// It might already be firing; that's okay.
		}
	}
	cm.commitTimer = time.AfterFunc(cm.gracePeriod, cm.flushIfNeeded)
	cm.timerActive = true
}

func (cm *ConversationManager) flushIfNeeded() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	text := strings.TrimSpace(cm.pendingText.String())
	if text == "" {
		cm.state = StateIdle
		cm.timerActive = false
		return
	}

	cm.pendingText.Reset()
	cm.state = StateIdle
	cm.timerActive = false

	log.Printf("[CONV] FLUSH (no speech for %v) → %q", cm.gracePeriod, text)
	select {
	case cm.output <- text:
	default:
		log.Printf("[CONV] dropping utterance due to full channel")
	}
}

func (cm *ConversationManager) SetAssistantSpeaking(on bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if on {
		log.Println("[CONV] state → ASSISTANT_SPEAKING")
		cm.state = StateAssistantSpeaking
	} else {
		log.Println("[CONV] state → IDLE")
		cm.state = StateIdle
	}
}
