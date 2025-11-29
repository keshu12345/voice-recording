package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DeepgramTTS struct {
	apiKey string
	client *http.Client
}

func NewDeepgramTTS(apiKey string) *DeepgramTTS {
	return &DeepgramTTS{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

type deepgramTTSPayload struct {
	Text  string `json:"text"`
	Voice string `json:"voice,omitempty"`
}

// Speak calls Deepgram TTS and plays the resulting audio via PortAudio.
// It also stores the TTS audio into tts_output.raw (raw PCM).
func (t *DeepgramTTS) Speak(ctx context.Context, text string) error {
	bodyStruct := deepgramTTSPayload{
		Text:  text,
		Voice: "aura-asteria",
	}
	bodyBytes, err := json.Marshal(bodyStruct)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepgram.com/v1/speak", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/x-raw;encoding=linear16;rate=16000;channels=1")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deepgram TTS error: status %d: %s", resp.StatusCode, string(b))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := os.WriteFile("tts_output.raw", audioData, 0644); err != nil {
		fmt.Printf("[TTS] error writing tts_output.raw: %v\n", err)
	} else {
		fmt.Printf("[TTS] saved TTS audio to tts_output.raw (%d bytes)\n", len(audioData))
	}

	return PlayPCM16(audioData)
}
