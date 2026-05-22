package services

import (
	"context"
	"encoding/json"
	"image"
	"math"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	derrors "github.com/y3owk1n/neru/internal/core/errors"
)

const (
	// EyeCursorFileEnv overrides the exact eye target JSON file to read.
	EyeCursorFileEnv = "NERU_EYE_CURSOR_FILE"
	// EyeDataDirEnv overrides the eye target data directory.
	EyeDataDirEnv = "NERU_EYE_DATA_DIR"
	// DefaultEyeMaxAge is the default staleness limit for gaze samples.
	DefaultEyeMaxAge = 5 * time.Second
)

// EyeCursorPayload is the minimal shape Neru needs from an eye source.
// The source can be native camera processing, a hardware eye tracker, or a
// temporary file emitted by an external prototype.
type EyeCursorPayload struct {
	SessionID string  `json:"sessionId"`
	TS        float64 `json:"ts"`
	ScreenX   float64 `json:"screenX"`
	ScreenY   float64 `json:"screenY"`
	Recording bool    `json:"recording"`
	PageURL   string  `json:"pageUrl"`
}

// EyeTarget is the latest usable gaze target in Neru's top-left screen coordinates.
type EyeTarget struct {
	Payload EyeCursorPayload
	Point   image.Point
	Age     time.Duration
}

// EyeService reads the latest gaze target from an eye source.
// File input is the current narrow provider boundary; native camera or hardware
// providers should feed this service through the same EyeCursorPayload shape.
type EyeService struct {
	cursorFile string
	maxAge     time.Duration
	now        func() time.Time
	logger     *zap.Logger
}

// EyeServiceOption configures EyeService.
type EyeServiceOption func(*EyeService)

// WithEyeCursorFile overrides the eye target payload file.
func WithEyeCursorFile(path string) EyeServiceOption {
	return func(s *EyeService) {
		s.cursorFile = path
	}
}

// WithEyeMaxAge overrides stale-sample rejection. Use 0 to disable it.
func WithEyeMaxAge(maxAge time.Duration) EyeServiceOption {
	return func(s *EyeService) {
		s.maxAge = maxAge
	}
}

// WithEyeNow overrides time for tests.
func WithEyeNow(now func() time.Time) EyeServiceOption {
	return func(s *EyeService) {
		if now != nil {
			s.now = now
		}
	}
}

// NewEyeService creates a gaze target service.
func NewEyeService(logger *zap.Logger, opts ...EyeServiceOption) *EyeService {
	if logger == nil {
		logger = zap.NewNop()
	}

	service := &EyeService{
		cursorFile: DefaultEyeCursorFile(),
		maxAge:     DefaultEyeMaxAge,
		now:        time.Now,
		logger:     logger,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service
}

// CursorFile returns the configured eye target payload file.
func (s *EyeService) CursorFile() string {
	return s.cursorFile
}

// Target returns the latest usable gaze target.
func (s *EyeService) Target(ctx context.Context) (EyeTarget, error) {
	select {
	case <-ctx.Done():
		return EyeTarget{}, derrors.Wrap(ctx.Err(), derrors.CodeContextCanceled, "read eye target canceled")
	default:
	}

	data, err := os.ReadFile(s.cursorFile)
	if err != nil {
		return EyeTarget{}, derrors.Wrap(
			err,
			derrors.CodeInvalidInput,
			"failed to read gaze cursor file",
		)
	}

	var payload EyeCursorPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return EyeTarget{}, derrors.Wrap(
			err,
			derrors.CodeInvalidInput,
			"failed to parse gaze cursor file",
		)
	}

	target, err := s.targetFromPayload(payload)
	if err != nil {
		return EyeTarget{}, err
	}

	s.logger.Debug("Resolved eye target",
		zap.Int("x", target.Point.X),
		zap.Int("y", target.Point.Y),
		zap.Duration("age", target.Age),
		zap.String("session_id", target.Payload.SessionID))

	return target, nil
}

func (s *EyeService) targetFromPayload(payload EyeCursorPayload) (EyeTarget, error) {
	if payload.SessionID == "" {
		return EyeTarget{}, derrors.New(
			derrors.CodeInvalidInput,
			"gaze cursor payload is missing sessionId",
		)
	}

	if !isFiniteEyeNumber(payload.TS) ||
		!isFiniteEyeNumber(payload.ScreenX) ||
		!isFiniteEyeNumber(payload.ScreenY) {
		return EyeTarget{}, derrors.New(
			derrors.CodeInvalidInput,
			"gaze cursor payload contains invalid numeric values",
		)
	}

	age := s.now().Sub(unixSecondsToTime(payload.TS))
	if age < 0 {
		age = -age
	}

	if s.maxAge > 0 && age > s.maxAge {
		return EyeTarget{}, derrors.Newf(
			derrors.CodeInvalidInput,
			"gaze cursor sample is stale: age %s exceeds %s",
			age.Round(time.Millisecond),
			s.maxAge,
		)
	}

	return EyeTarget{
		Payload: payload,
		Point: image.Point{
			X: int(math.Round(payload.ScreenX)),
			Y: int(math.Round(payload.ScreenY)),
		},
		Age: age,
	}, nil
}

// DefaultEyeCursorFile returns the eye target file Neru should read.
func DefaultEyeCursorFile() string {
	if cursorFile := os.Getenv(EyeCursorFileEnv); cursorFile != "" {
		return cursorFile
	}

	if dataDir := os.Getenv(EyeDataDirEnv); dataDir != "" {
		return filepath.Join(dataDir, "latest-cursor.json")
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".local", "share", "neru", "eye", "latest-cursor.json")
	}

	return filepath.Join("eye", "latest-cursor.json")
}

func unixSecondsToTime(seconds float64) time.Time {
	whole, frac := math.Modf(seconds)

	return time.Unix(int64(whole), int64(frac*float64(time.Second)))
}

func isFiniteEyeNumber(value float64) bool {
	return !math.IsInf(value, 0) && !math.IsNaN(value)
}
