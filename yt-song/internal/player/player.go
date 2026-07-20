package player

import (
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
)

// PlayState represents the current state of the audio player.
type PlayState int

const (
	StateStopped PlayState = iota
	StatePlaying
	StatePaused
)

const speakerSampleRate beep.SampleRate = 44100

// Player manages native audio playback of MP3 streams using beep.
type Player struct {
	mu                 sync.Mutex
	state              PlayState
	volumeLevel        float64 // 0.0 (silent) to 1.0 (max), default 0.8
	speakerInitialized bool

	// Beep playback pipeline
	streamer beep.StreamSeeker
	format   beep.Format
	ctrl     *beep.Ctrl
	volume   *effects.Volume

	// File tracker
	currentFile string

	// Callbacks
	onFinishFunc func()
}

// NewPlayer initializes and returns a new Player instance.
func NewPlayer() *Player {
	return &Player{
		volumeLevel: 0.8,
		state:       StateStopped,
	}
}

// SetOnFinish sets the callback to execute when a track finishes playing.
func (p *Player) SetOnFinish(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onFinishFunc = callback
}

// initSpeaker initializes the global beep speaker if not already done.
func (p *Player) initSpeaker() error {
	if p.speakerInitialized {
		return nil
	}

	// 100ms buffer latency for stability on terminal interfaces and VM audio routing
	bufferSize := speakerSampleRate.N(time.Millisecond * 100)
	err := speaker.Init(speakerSampleRate, bufferSize)
	if err != nil {
		return fmt.Errorf("speaker init failed: %w", err)
	}

	p.speakerInitialized = true
	return nil
}

// volumeLogScale maps [0.0, 1.0] to a logarithmic scale for beep's effects.Volume.
func (p *Player) volumeLogScale(level float64) float64 {
	if level <= 0.01 {
		return -10.0 // silent (effectively muted, 2^-10 = 0.00097)
	}
	// Logarithmic scale where level 1.0 is 0.0 (unaltered volume)
	// and level 0.5 is -1.0 (half amplitude)
	return math.Log2(level)
}

// Play opens and plays the MP3 file at filePath.
func (p *Player) Play(filePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Stop any active playback
	p.stopUnderlying()

	// Ensure speaker is initialized
	if err := p.initSpeaker(); err != nil {
		return err
	}

	// Open the downloaded audio file
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open audio file: %w", err)
	}

	// Decode the MP3 format
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to decode mp3: %w", err)
	}

	p.streamer = streamer
	p.format = format
	p.currentFile = filePath

	// Resample streamer if the sample rate doesn't match the speaker
	var resampled beep.Streamer = streamer
	if format.SampleRate != speakerSampleRate {
		resampled = beep.Resample(4, format.SampleRate, speakerSampleRate, streamer)
	}

	// Create volume control wrapper
	p.volume = &effects.Volume{
		Streamer: resampled,
		Base:     2.0,
		Volume:   p.volumeLogScale(p.volumeLevel),
	}

	// Setup finish callback sequence
	var onFinish func()
	if p.onFinishFunc != nil {
		onFinish = p.onFinishFunc
	}

	sequence := beep.Seq(p.volume, beep.Callback(func() {
		p.mu.Lock()
		isStopped := p.state == StateStopped
		p.state = StateStopped
		p.mu.Unlock()

		// Only invoke callback if it wasn't manually stopped
		if !isStopped && onFinish != nil {
			onFinish()
		}
	}))

	// Controller for pausing/resuming
	p.ctrl = &beep.Ctrl{
		Streamer: sequence,
		Paused:   false,
	}

	p.state = StatePlaying

	// Queue playback to speaker
	speaker.Play(p.ctrl)

	return nil
}

// TogglePause pauses playing audio, or resumes it.
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctrl == nil || !p.speakerInitialized {
		return
	}

	if p.state == StatePlaying {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
		p.state = StatePaused
	} else if p.state == StatePaused {
		speaker.Lock()
		p.ctrl.Paused = false
		speaker.Unlock()
		p.state = StatePlaying
	}
}

// Stop halts playback and cleans up audio streaming resources.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopUnderlying()
}

// stopUnderlying stops playback. Caller MUST hold the lock.
func (p *Player) stopUnderlying() {
	p.state = StateStopped
	if p.speakerInitialized {
		speaker.Clear()
	}

	if p.streamer != nil {
		if closer, ok := p.streamer.(io.Closer); ok {
			_ = closer.Close()
		}
		p.streamer = nil
	}
}

// SetVolume sets the volume level (0.0 to 1.0).
func (p *Player) SetVolume(level float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if level < 0.0 {
		level = 0.0
	}
	if level > 1.0 {
		level = 1.0
	}
	p.volumeLevel = level

	if p.volume != nil && p.speakerInitialized {
		speaker.Lock()
		p.volume.Volume = p.volumeLogScale(p.volumeLevel)
		speaker.Unlock()
	}
}

// Seek seeks to a specific time position in seconds.
func (p *Player) Seek(seconds float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.streamer == nil || !p.speakerInitialized {
		return fmt.Errorf("no track loaded or audio device not initialized")
	}

	sampleIndex := int(seconds * float64(p.format.SampleRate))
	if sampleIndex < 0 {
		sampleIndex = 0
	}
	if sampleIndex > p.streamer.Len() {
		sampleIndex = p.streamer.Len()
	}

	speaker.Lock()
	err := p.streamer.Seek(sampleIndex)
	speaker.Unlock()

	return err
}

// Status returns current player metrics.
func (p *Player) Status() (current float64, total float64, volume float64, state PlayState) {
	p.mu.Lock()
	defer p.mu.Unlock()

	volume = p.volumeLevel
	state = p.state

	if p.streamer == nil || !p.speakerInitialized {
		return 0, 0, volume, state
	}

	speaker.Lock()
	pos := p.streamer.Position()
	totalSamples := p.streamer.Len()
	speaker.Unlock()

	current = float64(pos) / float64(p.format.SampleRate)
	total = float64(totalSamples) / float64(p.format.SampleRate)

	return current, total, volume, state
}

// CurrentFile returns the path of the currently loaded audio file.
func (p *Player) CurrentFile() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentFile
}
