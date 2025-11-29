# Voice Assistant

A real-time voice assistant built with Go that captures speech, processes it through AI, and responds with synthesized voice output.

## Features

-  Real-time microphone audio capture
-  Speech-to-Text via Deepgram
-  Conversational AI powered by OpenAI GPT-4o-mini
-  Text-to-Speech via Deepgram
-  Smart conversation grace period handling

## Architecture

```
User Speech (Mic)
    ↓
PortAudio Capture         (audio.go → StartMicCapture)
    ↓
Deepgram STT             (stt_deepgram.go → Run)
    ↓
ConversationManager      (conversation.go → grace period)
    ↓
OpenAI GPT-4o-mini       (llm_openai.go → Generate)
    ↓
Deepgram TTS             (tts_deepgram.go → Speak)
    ↓
PortAudio Playback       (audio.go → PlayPCM16)
    ↓
User Hears Response
```

## Prerequisites

### 1. Install Go

Download and install Go from the [official website](https://golang.org/dl/):

```bash
# Verify installation
go version
```

### 2. Install PortAudio

PortAudio is required for audio capture and playback.

**macOS:**
```bash
brew install portaudio
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install portaudio19-dev
```

**Linux (Fedora):**
```bash
sudo dnf install portaudio-devel
```

**Windows:**
- Download pre-built binaries from [PortAudio website](http://www.portaudio.com/download.html)
- Or use [MSYS2](https://www.msys2.org/):
  ```bash
  pacman -S mingw-w64-x86_64-portaudio
  ```

### 3. API Keys

You'll need API keys for:
- **Deepgram** (STT/TTS): [Get API key](https://deepgram.com/)
- **OpenAI** (LLM): [Get API key](https://platform.openai.com/)

Create a `.env` file in the project root:
```env
DEEPGRAM_API_KEY=your_deepgram_key_here
OPENAI_API_KEY=your_openai_key_here
```

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd <project-directory>
```

2. Install dependencies:
```bash
go mod download
```

## Usage

### Running the Application

```bash
go run .
```

The assistant will start listening to your microphone. Speak naturally, and it will respond with voice output.

### Configuration

Edit configuration values in your main application file or environment variables:
- **Sample Rate**: 16000 Hz (default)
- **Channels**: Mono (1 channel)
- **Bit Depth**: 16-bit PCM
- **Grace Period**: Configurable pause detection

## Audio Output

The application generates raw audio files during operation.

### Playing Raw Audio

The `mic_input.raw` file contains:
- RAW audio format (not MP3/WAV)
- 16-bit PCM
- 16000 Hz sample rate
- Mono channel

**Play with ffplay:**
```bash
ffplay -f s16le -ar 16000 -ac 1 mic_input.raw
```

**Convert to WAV (for easier playback):**
```bash
ffmpeg -f s16le -ar 16000 -ac 1 -i mic_input.raw mic_input.wav
```

After conversion, you can double-click the WAV file to play it in your default audio player.

## Project Structure

```
.
├── audio.go              # PortAudio capture and playback
├── stt_deepgram.go      # Speech-to-Text integration
├── llm_openai.go        # OpenAI GPT integration
├── tts_deepgram.go      # Text-to-Speech integration
├── conversation.go      # Conversation flow manager
├── main.go              # Application entry point
└── README.md
```

## Troubleshooting

### PortAudio Issues

**macOS: "library not loaded" error**
```bash
brew reinstall portaudio
```

**Linux: "portaudio.h: No such file"**
```bash
sudo apt-get install portaudio19-dev
```

### Audio Quality Issues

- Ensure your microphone is properly connected
- Check system audio input settings
- Verify sample rate compatibility (16000 Hz)
- Test with: `ffplay -f s16le -ar 16000 -ac 1 mic_input.raw`

### API Connection Issues

- Verify API keys in `.env` file
- Check internet connectivity
- Review API rate limits and quotas

## Dependencies

- [PortAudio](http://www.portaudio.com/) - Audio I/O library
- [Deepgram](https://deepgram.com/) - STT and TTS services
- [OpenAI](https://openai.com/) - Language model API

## License

[Your License Here]

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing documentation
- Review API provider documentation

---

**Note**: Make sure to keep your API keys secure and never commit them to version control.