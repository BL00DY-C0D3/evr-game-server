package wrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var (
	ErrorFailedToParseJSONData = fmt.Errorf("failed to parse JSON data")
	ErrorFailedToGetSessionID  = fmt.Errorf("failed to get sessionid from JSON data")
)

type SessionFrame struct {
	SessionID string
	Timestamp time.Time
	Data      []byte
}

func newSessionHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxConnsPerHost:       1,
		DisableCompression:    true,
		MaxIdleConns:          0, // Set MaxIdleConns to 0 to close the connection after every request
		MaxIdleConnsPerHost:   0, // Set MaxIdleConnsPerHost to 0 to close the connection after every request
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 10 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Timeout:   3 * time.Second, // Overall request timeout
		Transport: transport,
	}

	return client
}

type SessionAgent struct {
	ctx      context.Context
	cancelFn context.CancelFunc

	logger       *zap.Logger
	client       *http.Client
	requestQueue *EventQueue

	frameCh chan SessionFrame
}

type LoggerMonitor struct {
	logger    FrameLogger
	timestamp time.Time
}

func (m *LoggerMonitor) Log(timestamp time.Time, frame []byte) error {
	m.timestamp = timestamp
	return m.logger.Log(timestamp, frame)
}
func NewSessionAgent(logger *zap.Logger, workerCount int, queueSize int) *SessionAgent {
	ctx, cancel := context.WithCancel(context.Background())

	return &SessionAgent{
		ctx:      ctx,
		cancelFn: cancel,

		logger:       logger,
		client:       newSessionHTTPClient(),
		requestQueue: NewEventQueue(logger, workerCount, queueSize),

		frameCh: make(chan SessionFrame, 16),
	}
}

func (s *SessionAgent) Start() {
	s.Consume()
}

func (s *SessionAgent) Stop() {
	s.cancelFn()
}

func (s *SessionAgent) Consume() error {
	monitors := make(map[string]LoggerMonitor, 0)
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-s.ctx.Done():
			for _, m := range monitors {
				m.logger.Close()
			}
			return nil
		case frame := <-s.frameCh:

			m, ok := monitors[frame.SessionID]
			if !ok {
				logger, err := NewEchoReplayFrameLogger(fmt.Sprintf("%s.echoreplay", frame.SessionID))
				if err != nil {
					s.logger.Error("Failed to create logger", zap.Error(err))
				}
				monitors[frame.SessionID] = LoggerMonitor{
					logger:    logger,
					timestamp: frame.Timestamp,
				}
				m = monitors[frame.SessionID]
			}
			if err := m.logger.Log(frame.Timestamp, frame.Data); err != nil {
				s.logger.Error("Failed to log frame", zap.Error(err))
			}

		case <-ticker.C:
			for sessionID, m := range monitors {
				if time.Since(m.timestamp) > 60*time.Second {
					m.logger.Close()
					delete(monitors, sessionID)
					return nil
				}
			}

		}
	}
}

func (s *SessionAgent) Poll(url string, frequency int) {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	step := time.Second / time.Duration(frequency)
	ticker := time.NewTicker(time.Second / time.Duration(frequency))
	defer ticker.Stop()

	logger := s.logger.With(zap.String("url", url))

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Error("Failed to create request", zap.Error(err))
		return
	}
	var start time.Time
	for {
		start = <-ticker.C
		select {
		case <-ctx.Done():
			logger.Debug("Context cancelled, stopping...")
			return
		default:
		}

		logger.Debug("Making request...")

		resp, err := s.client.Do(request)
		if err != nil {
			if isConnectionError(err) {
				logger.Debug("Connection refused, skipping...")
			} else {
				logger.Error("Failed to make request", zap.Error(err))
			}
			cancel()
			return // Return to avoid further processing
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Failed to read response body", zap.Error(err))
			return // Return to avoid further processing
		}

		elapsed := time.Since(start)
		logger.Debug("Request-response time", zap.Duration("elapsed", elapsed))

		// Adjust timing if needed
		if elapsed > step {
			logger.Warn("Request-response time exceeds frequency (api might be overloaded)", zap.Duration("elapsed", elapsed), zap.Duration("step", step))
		}
		sessionID, err := parseSessionID(body)
		if err != nil {
			logger.Error("Failed to parse session ID from session frame", zap.Error(err))
		}

		s.frameCh <- SessionFrame{
			SessionID: sessionID,
			Timestamp: time.Now(),
			Data:      body,
		}
	}
}

type sessionFrame struct {
	SessionID string `json:"sessionid"`
}

func parseSessionID(data []byte) (string, error) {
	var frame sessionFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return "", ErrorFailedToParseJSONData
	} else if frame.SessionID == "" {
		return "", ErrorFailedToGetSessionID
	}

	return frame.SessionID, nil
}

func isConnectionError(err error) bool {

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) && syscallErr == syscall.ECONNREFUSED {
		return true
	}
	return false
}
