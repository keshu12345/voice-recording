package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type DeepgramSTT struct {
	apiKey string
}

func NewDeepgramSTT(apiKey string) *DeepgramSTT {
	return &DeepgramSTT{apiKey: apiKey}
}

type deepgramResponse struct {
	Channel struct {
		Alternatives []struct {
			Transcript string `json:"transcript"`
		} `json:"alternatives"`
		IsFinal bool `json:"is_final"`
	} `json:"channel"`
}

func (d *DeepgramSTT) Run(ctx context.Context, audio <-chan []int16, out chan<- Transcript) error {
	defer close(out)

	u := url.URL{
		Scheme:   "wss",
		Host:     "api.deepgram.com",
		Path:     "/v1/listen",
		RawQuery: "model=nova-2-general&encoding=linear16&sample_rate=16000&channels=1",
	}

	header := http.Header{}
	header.Set("Authorization", "Token "+d.apiKey)

	log.Printf("[STT] connecting to Deepgram WebSocket: %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return err
	}
	defer func() {
		log.Println("[STT] closing Deepgram WebSocket")
		conn.Close()
	}()

	// 	// Writer: send audio to Deepgram
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("[STT] context cancelled, sending close message")
				_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
				return
			case frame, ok := <-audio:
				if !ok {
					log.Println("[STT] audio channel closed, sending close message")
					_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "no more audio"))
					return
				}
				data := int16SliceToBytes(frame)
				log.Printf("[STT → Deepgram] sending %d bytes", len(data))
				if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
					log.Printf("[STT] deepgram write error: %v", err)
					return
				}
			}
		}
	}()
	conn.SetReadDeadline(time.Time{}) // disable deadline for streaming
	for {
		select {
		case <-ctx.Done():
			log.Println("[STT] context cancelled, stopping reader")
			return nil
		default:
		}

		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[STT] deepgram read error: %v", err)
			return err
		}

		if msgType != websocket.TextMessage {
			// Deepgram usually sends JSON text messages; ignore others
			continue
		}

		var resp deepgramResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			log.Printf("[STT] unable to parse message: %s", string(msg))
			continue
		}

		if len(resp.Channel.Alternatives) == 0 {
			continue
		}

		text := resp.Channel.Alternatives[0].Transcript
		isFinal := resp.Channel.IsFinal

		// 👇 VERY IMPORTANT: this prints *every* transcript
		log.Printf("[STT] transcript from Deepgram: %q (final=%v)", text, isFinal)

		out <- Transcript{
			Text:  text,
			Final: isFinal,
		}
	}
}
