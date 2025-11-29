package main

import (
	"context"
	"encoding/binary"
	"log"
	"os"

	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate      = 16000
	channels        = 1
	framesPerBuffer = 1024
	micRowFile      = "mic_input.raw"
)

func InitAudio() error {
	log.Println("[Audio] initializing PortAudio...")
	return portaudio.Initialize()
}

func ShutdownAudio() {
	log.Println("[Audio] terminating PortAudio...")
	if err := portaudio.Terminate(); err != nil {
		log.Printf("[Audio] error terminating PortAudio: %v", err)
	}
}

// StartMicCapture reads from default mic and sends int16 frames to out.
// Additionally, it stores all captured audio into mic_input.raw (raw PCM).
func StartMicCapture(ctx context.Context, out chan<- []int16) error {
	buffer := make([]int16, framesPerBuffer)

	//  Open file to store microphone input (raw PCM 16-bit, 16kHz, mono).
	f, err := os.Create(micRowFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("[Audio] error closing mic_input.raw: %v", err)
		}
		log.Println("[Audio] mic_input.raw closed")
	}()

	stream, err := portaudio.OpenDefaultStream(channels, 0, sampleRate, len(buffer), &buffer)
	if err != nil {
		return err
	}
	if err := stream.Start(); err != nil {
		return err
	}
	defer func() {
		_ = stream.Stop()
		_ = stream.Close()
		close(out)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := stream.Read(); err != nil {
			log.Printf("[Audio] mic read error: %v", err)
			return err
		}

		frame := make([]int16, len(buffer))
		copy(frame, buffer)

		log.Printf("[Audio] captured %d samples", len(frame))

		// STORE AUDIO: write frame to mic_input.raw as little-endian PCM
		data := int16SliceToBytes(frame)
		if _, err := f.Write(data); err != nil {
			log.Printf("[Audio] error writing to mic_input.raw: %v", err)
		}

		select {
		case out <- frame:
		case <-ctx.Done():
			return nil
		}
	}
}

func int16SliceToBytes(samples []int16) []byte {
	b := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(b[2*i:], uint16(s))
	}
	return b
}

func bytesToInt16Slice(data []byte) []int16 {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	n := len(data) / 2
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		out[i] = int16(binary.LittleEndian.Uint16(data[2*i:]))
	}
	return out
}

// PlayPCM16 plays raw linear16 PCM data (16-bit, 16kHz, mono).
func PlayPCM16(data []byte) error {
	samples := bytesToInt16Slice(data)
	buffer := make([]int16, framesPerBuffer)

	stream, err := portaudio.OpenDefaultStream(0, channels, sampleRate, len(buffer), &buffer)
	if err != nil {
		return err
	}
	if err := stream.Start(); err != nil {
		return err
	}
	defer func() {
		_ = stream.Stop()
		_ = stream.Close()
	}()

	offset := 0
	for offset < len(samples) {
		n := copy(buffer, samples[offset:])
		offset += n
		if err := stream.Write(); err != nil {
			return err
		}
	}
	return nil
}
