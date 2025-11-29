//go:build !android
// +build !android

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (DEEPGRAM_API_KEY, OPENAI_API_KEY)
	_ = godotenv.Load()

	deepgramAPIKey := os.Getenv("DEEPGRAM_API_KEY")
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")

	if deepgramAPIKey == "" || openaiAPIKey == "" {
		log.Fatal("DEEPGRAM_API_KEY and OPENAI_API_KEY must be set in environment or .env")
	}

	// Initialize PortAudio (audio I/O)
	if err := InitAudio(); err != nil {
		log.Fatalf("failed to init audio: %v", err)
	}
	defer ShutdownAudio()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channels (pipelines between goroutines)
	audioFrames := make(chan []int16, 32) // mic → STT
	transcripts := make(chan Transcript, 32)
	userQueries := make(chan string, 8)      // conversation manager → LLM
	assistantReplies := make(chan string, 8) // LLM → TTS

	// Services
	stt := NewDeepgramSTT(deepgramAPIKey)
	llm := NewOpenAILLM(openaiAPIKey)
	tts := NewDeepgramTTS(deepgramAPIKey)

	// 🔊 QUICK TTS TEST at startup to force tts_output.raw creation
	go func() {
		testCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		log.Println("[TTS-TEST] calling Deepgram TTS test...")
		if err := tts.Speak(testCtx, "This is a Deepgram TTS test from Go."); err != nil {
			log.Printf("[TTS-TEST] error: %v", err)
		} else {
			log.Println("[TTS-TEST] success. Check tts_output.raw in current folder.")
		}
	}()

	// Conversation manager (interruption fix with grace period)
	convMgr := NewConversationManager(1500*time.Millisecond, userQueries)

	var wg sync.WaitGroup

	// Microphone capture → audioFrames
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[Main] starting mic capture...")
		if err := StartMicCapture(ctx, audioFrames); err != nil {
			log.Printf("[Main] mic capture stopped with error: %v", err)
		}
		log.Println("[Main] mic capture stopped.")
	}()

	// Deepgram STT: audioFrames → transcripts
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[Main] starting STT...")
		if err := stt.Run(ctx, audioFrames, transcripts); err != nil {
			log.Printf("[Main] STT stopped with error: %v", err)
		}
		log.Println("[Main] STT stopped.")
	}()

	// Conversation manager: transcripts → userQueries (with grace period)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[Main] starting ConversationManager...")
		convMgr.Run(ctx, transcripts)
		log.Println("[Main] ConversationManager stopped.")
	}()

	// LLM: userQueries → assistantReplies
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[Main] starting LLM loop...")
		for {
			select {
			case <-ctx.Done():
				log.Println("[Main] LLM loop: context cancelled")
				return
			case query, ok := <-userQueries:
				if !ok {
					log.Println("[Main] LLM loop: userQueries channel closed")
					return
				}
				log.Printf("[LLM] Query     : %q", query)

				resp, err := llm.Generate(ctx, query)
				if err != nil {
					log.Printf("[LLM] error: %v", err)
					continue
				}
				log.Printf("[LLM] Response  : %q", resp)

				assistantReplies <- resp
			}
		}
	}()

	// TTS: assistantReplies → speakers
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[Main] starting TTS loop...")
		for {
			select {
			case <-ctx.Done():
				log.Println("[Main] TTS loop: context cancelled")
				return
			case reply, ok := <-assistantReplies:
				if !ok {
					log.Println("[Main] TTS loop: assistantReplies channel closed")
					return
				}
				log.Printf("[TTS] speaking reply: %q", reply)

				convMgr.SetAssistantSpeaking(true)
				if err := tts.Speak(ctx, reply); err != nil {
					log.Printf("[TTS] error: %v", err)
				}
				convMgr.SetAssistantSpeaking(false)

				log.Println("[TTS] finished playback.")
			}
		}
	}()

	log.Println("Microphone started. Speak now... (Ctrl+C to exit)")

	// Handle Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("[Main] Shutting down...")
	cancel()
	wg.Wait()
	log.Println("[Main] Shutdown complete.")
}
