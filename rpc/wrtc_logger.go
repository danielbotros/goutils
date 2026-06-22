package rpc

import (
	"strings"

	"github.com/pion/logging"
	"go.uber.org/zap"

	"go.viam.com/utils"
)

// isBenignTURNRefreshNoise reports whether a pion log message is the benign TURN
// credential-refresh failure that recurs on long-lived connections.
//
// pion's TURN client logs "Fail/Failed to refresh permissions" and "Failed to
// refresh allocation" at error/warn when an allocation's time-limited credential
// has expired and the TURN server rejects the periodic refresh. For an in-use
// relay this is automatically recovered by a re-dial with fresh credentials; for
// an idle/unused allocation (a connection that selected a direct candidate pair
// but still gathered a relay candidate) it is harmless and repeats every ~2min.
// Either way it does not reflect connection health, so we demote it to debug.
func isBenignTURNRefreshNoise(msg string) bool {
	return strings.Contains(msg, "refresh permissions") || strings.Contains(msg, "refresh allocation")
}

// WebRTCLoggerFactory wraps a utils.ZapCompatibleLogger for use with pion's webrtc logging system.
type WebRTCLoggerFactory struct {
	Logger utils.ZapCompatibleLogger
}

type webrtcLogger struct {
	logger utils.ZapCompatibleLogger
}

func (l webrtcLogger) loggerWithSkip() utils.ZapCompatibleLogger {
	return l.logger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
}

func (l webrtcLogger) Trace(msg string) {
	l.loggerWithSkip().Debug(msg)
}

func (l webrtcLogger) Tracef(format string, args ...interface{}) {
	l.loggerWithSkip().Debugf(format, args...)
}

func (l webrtcLogger) Debug(msg string) {
	l.loggerWithSkip().Debug(msg)
}

func (l webrtcLogger) Debugf(format string, args ...interface{}) {
	l.loggerWithSkip().Debugf(format, args...)
}

func (l webrtcLogger) Info(msg string) {
	l.loggerWithSkip().Info(msg)
}

func (l webrtcLogger) Infof(format string, args ...interface{}) {
	l.loggerWithSkip().Infof(format, args...)
}

func (l webrtcLogger) Warn(msg string) {
	l.loggerWithSkip().Warn(msg)
}

func (l webrtcLogger) Warnf(format string, args ...interface{}) {
	l.loggerWithSkip().Warnf(format, args...)
}

func (l webrtcLogger) Error(msg string) {
	l.loggerWithSkip().Error(msg)
}

func (l webrtcLogger) Errorf(format string, args ...interface{}) {
	l.loggerWithSkip().Errorf(format, args...)
}

// NewLogger returns a new webrtc logger under the given scope.
func (lf WebRTCLoggerFactory) NewLogger(scope string) logging.LeveledLogger {
	return webrtcLogger{utils.Sublogger(lf.Logger, scope)}
}

// demoteTURNNoiseLoggerFactory wraps another pion logging.LoggerFactory and demotes
// the benign TURN credential-refresh noise (see isBenignTURNRefreshNoise) to debug,
// passing every other log through to the wrapped factory unchanged. This lets us
// silence that recurring error without otherwise altering pion's logging behavior or
// volume.
type demoteTURNNoiseLoggerFactory struct {
	base logging.LoggerFactory
}

func (f demoteTURNNoiseLoggerFactory) NewLogger(scope string) logging.LeveledLogger {
	return demoteTURNNoiseLogger{f.base.NewLogger(scope)}
}

// demoteTURNNoiseLogger demotes benign TURN credential-refresh error/warn messages to
// debug. All other levels (and all other messages) are inherited unchanged from the
// embedded logger.
type demoteTURNNoiseLogger struct {
	logging.LeveledLogger
}

func (l demoteTURNNoiseLogger) Warn(msg string) {
	if isBenignTURNRefreshNoise(msg) {
		l.LeveledLogger.Debug(msg)
		return
	}
	l.LeveledLogger.Warn(msg)
}

func (l demoteTURNNoiseLogger) Warnf(format string, args ...interface{}) {
	if isBenignTURNRefreshNoise(format) {
		l.LeveledLogger.Debugf(format, args...)
		return
	}
	l.LeveledLogger.Warnf(format, args...)
}

func (l demoteTURNNoiseLogger) Error(msg string) {
	if isBenignTURNRefreshNoise(msg) {
		l.LeveledLogger.Debug(msg)
		return
	}
	l.LeveledLogger.Error(msg)
}

func (l demoteTURNNoiseLogger) Errorf(format string, args ...interface{}) {
	if isBenignTURNRefreshNoise(format) {
		l.LeveledLogger.Debugf(format, args...)
		return
	}
	l.LeveledLogger.Errorf(format, args...)
}
