package browser

import (
	"fmt"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// StreamFrame represents a single frame in the stream with optional metadata
type StreamFrame struct {
	Data      []byte                 `json:"data"` // JPEG or PNG data
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetadataFunc is a function that returns current metadata for a frame
type MetadataFunc func() map[string]interface{}

// Streamer handles real-time browser view streaming via screencast
type Streamer struct {
	mu           sync.RWMutex
	page         *rod.Page
	frames       chan *StreamFrame
	stopCh       chan struct{}
	running      bool
	quality      int
	format       proto.PageStartScreencastFormat
	sessionID    string
	metadataFunc MetadataFunc
	history      []*StreamFrame // Sprint 27.4: Replay buffer
	maxHistory   int
}

// NewStreamer creates a new streamer for a page
func NewStreamer(page *rod.Page, sessionID string) *Streamer {
	return &Streamer{
		page:       page,
		frames:     make(chan *StreamFrame, 10),
		stopCh:     make(chan struct{}),
		quality:    80,
		format:     proto.PageStartScreencastFormatJpeg,
		sessionID:  sessionID,
		maxHistory: 100, // Keep last 100 frames for replay
	}
}

// SetMetadataFunc sets the function used to capture metadata for frames
func (s *Streamer) SetMetadataFunc(f MetadataFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadataFunc = f
}

// Start initiates the screencast
func (s *Streamer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	logger.Info("[%s] Starting screencast stream", s.sessionID)

	// Enable Page domain
	if err := (proto.PageEnable{}).Call(s.page); err != nil {
		return fmt.Errorf("failed to enable page domain: %w", err)
	}

	// Listen for screencast frames
	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			logger.Info("[%s] Screencast stream stopped", s.sessionID)
		}()

		// Event listener for screencast frames
		wait := s.page.EachEvent(func(e *proto.PageScreencastFrame) {
			select {
			case <-s.stopCh:
				return 
			default:
				// Send frame to channel
				frame := &StreamFrame{
					Data:      e.Data,
					Timestamp: int64(e.Metadata.Timestamp),
				}

				// Sprint 27.2: Add semantic metadata if available
				s.mu.RLock()
				if s.metadataFunc != nil {
					frame.Metadata = s.metadataFunc()
				}

				// Sprint 27.4: Add to history
				s.history = append(s.history, frame)
				if len(s.history) > s.maxHistory {
					s.history = s.history[1:]
				}
				s.mu.RUnlock()

				select {
				case s.frames <- frame:
				default:
					// Drop frame if channel is full
				}

				// Acknowledge the frame
				_ = (proto.PageScreencastFrameAck{
					SessionID: e.SessionID,
				}).Call(s.page)
			}
		})
		wait()
	}()

	// Start screencast
	quality := s.quality
	err := (proto.PageStartScreencast{
		Format:  s.format,
		Quality: &quality,
	}).Call(s.page)

	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to start screencast: %w", err)
	}

	return nil
}

// Stop stops the screencast
func (s *Streamer) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	s.running = false
	s.mu.Unlock()

	// Stop screencast via CDP
	_ = (proto.PageStopScreencast{}).Call(s.page)
}

// GetFrames returns the frames channel
func (s *Streamer) GetFrames() <-chan *StreamFrame {
	return s.frames
}

// GetHistory returns the recorded history
func (s *Streamer) GetHistory() []*StreamFrame {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	history := make([]*StreamFrame, len(s.history))
	copy(history, s.history)
	return history
}

// SetQuality sets JPEG quality (1-100)
func (s *Streamer) SetQuality(q int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quality = q
}
