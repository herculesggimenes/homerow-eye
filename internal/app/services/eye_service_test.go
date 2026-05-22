package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestEyeServiceTarget(t *testing.T) {
	now := time.Unix(1000, 0)
	path := writeEyeCursorFixture(t, `{
		"sessionId": "session-1",
		"ts": 999.75,
		"screenX": 100.4,
		"screenY": 200.6
	}`)

	service := NewEyeService(
		zap.NewNop(),
		WithEyeCursorFile(path),
		WithEyeMaxAge(2*time.Second),
		WithEyeNow(func() time.Time { return now }),
	)

	target, err := service.Target(context.Background())
	if err != nil {
		t.Fatalf("Target() error = %v", err)
	}

	if target.Point.X != 100 || target.Point.Y != 201 {
		t.Fatalf("Target() point = %v, want (100,201)", target.Point)
	}

	if target.Payload.SessionID != "session-1" {
		t.Fatalf("Target() session = %q, want session-1", target.Payload.SessionID)
	}
}

func TestEyeServiceTargetRejectsStaleSample(t *testing.T) {
	path := writeEyeCursorFixture(t, `{
		"sessionId": "session-1",
		"ts": 990,
		"screenX": 100,
		"screenY": 200
	}`)

	service := NewEyeService(
		zap.NewNop(),
		WithEyeCursorFile(path),
		WithEyeMaxAge(2*time.Second),
		WithEyeNow(func() time.Time { return time.Unix(1000, 0) }),
	)

	_, err := service.Target(context.Background())
	if err == nil {
		t.Fatal("Target() expected stale sample error")
	}

	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("Target() error = %v, want stale sample error", err)
	}
}

func TestEyeServiceTargetRejectsMissingSession(t *testing.T) {
	path := writeEyeCursorFixture(t, `{
		"ts": 1000,
		"screenX": 100,
		"screenY": 200
	}`)

	service := NewEyeService(
		zap.NewNop(),
		WithEyeCursorFile(path),
		WithEyeNow(func() time.Time { return time.Unix(1000, 0) }),
	)

	_, err := service.Target(context.Background())
	if err == nil {
		t.Fatal("Target() expected missing session error")
	}

	if !strings.Contains(err.Error(), "sessionId") {
		t.Fatalf("Target() error = %v, want sessionId error", err)
	}
}

func TestEyeServiceStartCreatesDataDir(t *testing.T) {
	cursorFile := filepath.Join(t.TempDir(), "nested", "eye", "latest-cursor.json")
	service := NewEyeService(zap.NewNop(), WithEyeCursorFile(cursorFile))

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	info, err := os.Stat(filepath.Dir(cursorFile))
	if err != nil {
		t.Fatalf("expected eye data dir to exist: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("eye data path is not a directory: %s", filepath.Dir(cursorFile))
	}
}

func TestEyeServiceTargetMissingFileIsReadyForSource(t *testing.T) {
	cursorFile := filepath.Join(t.TempDir(), "eye", "latest-cursor.json")
	service := NewEyeService(zap.NewNop(), WithEyeCursorFile(cursorFile))

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	_, err := service.Target(context.Background())
	if err == nil {
		t.Fatal("Target() expected missing target error")
	}

	if !strings.Contains(err.Error(), "ready and waiting for an eye source") {
		t.Fatalf("Target() error = %v, want ready/waiting error", err)
	}
}

func writeEyeCursorFixture(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "latest-cursor.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	return path
}
