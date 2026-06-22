package rpc

import (
	"testing"

	"go.viam.com/test"
)

func TestIsBenignTURNRefreshNoise(t *testing.T) {
	for _, tc := range []struct {
		msg    string
		benign bool
	}{
		{"Fail to refresh permissions: CreatePermission error response (error 401: Unauthorized)", true},
		{"Failed to refresh permissions: CreatePermission error response (error 400: Bad Request)", true},
		{"Failed to refresh allocation: stale nonce", true},
		{"Fail to refresh permissions: %s", true},
		{"some unrelated error", false},
		{"ICE connection state changed: failed", false},
	} {
		test.That(t, isBenignTURNRefreshNoise(tc.msg), test.ShouldEqual, tc.benign)
	}
}

// recordingLeveledLogger records the level at which the most recent message was logged.
type recordingLeveledLogger struct {
	level string
}

func (r *recordingLeveledLogger) Trace(string)          { r.level = "trace" }
func (r *recordingLeveledLogger) Tracef(string, ...any) { r.level = "trace" }
func (r *recordingLeveledLogger) Debug(string)          { r.level = "debug" }
func (r *recordingLeveledLogger) Debugf(string, ...any) { r.level = "debug" }
func (r *recordingLeveledLogger) Info(string)           { r.level = "info" }
func (r *recordingLeveledLogger) Infof(string, ...any)  { r.level = "info" }
func (r *recordingLeveledLogger) Warn(string)           { r.level = "warn" }
func (r *recordingLeveledLogger) Warnf(string, ...any)  { r.level = "warn" }
func (r *recordingLeveledLogger) Error(string)          { r.level = "error" }
func (r *recordingLeveledLogger) Errorf(string, ...any) { r.level = "error" }

func TestDemoteTURNNoiseLogger(t *testing.T) {
	rec := &recordingLeveledLogger{}
	l := demoteTURNNoiseLogger{rec}

	// Benign TURN refresh noise is demoted to debug regardless of the level pion used.
	l.Error("Fail to refresh permissions: CreatePermission error response (error 401: Unauthorized)")
	test.That(t, rec.level, test.ShouldEqual, "debug")

	l.Errorf("Fail to refresh permissions: %s", "boom")
	test.That(t, rec.level, test.ShouldEqual, "debug")

	l.Warnf("Failed to refresh allocation: %s", "stale nonce")
	test.That(t, rec.level, test.ShouldEqual, "debug")

	// Unrelated errors/warns pass through unchanged.
	l.Error("a real error")
	test.That(t, rec.level, test.ShouldEqual, "error")

	l.Warn("a real warning")
	test.That(t, rec.level, test.ShouldEqual, "warn")
}
