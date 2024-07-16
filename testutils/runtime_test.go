package testutils

import (
	"context"
	"sync"
	"testing"

	"github.com/edaniels/golog"
	"github.com/pkg/errors"
	"go.uber.org/zap/zaptest/observer"
	"go.viam.com/test"

	"go.viam.com/utils"
)

func TestContextualMain(t *testing.T) {
	var captured []interface{}
	prevFatal := fatal
	prevError := tError
	defer func() {
		fatal = prevFatal
		tError = prevError
	}()
	fatal = func(t *testing.T, args ...interface{}) {
		t.Helper()
		captured = args
	}
	tError = fatal

	err1 := errors.New("whoops")
	mainWithArgs := func(ctx context.Context, args []string, logger utils.ZapCompatibleLogger) error {
		return err1
	}
	logger := golog.NewTestLogger(t)
	exec := ContextualMain(mainWithArgs, nil, logger)
	exec.Start()
	<-exec.Ready
	exec.Stop()
	test.That(t, <-exec.Done, test.ShouldEqual, err1)
	test.That(t, captured, test.ShouldHaveLength, 0)

	exec = ContextualMain(mainWithArgs, nil, logger)
	exec.Start()
	<-exec.Ready
	exec.QuitSignal(t)
	exec.Stop()
	test.That(t, <-exec.Done, test.ShouldEqual, err1)
	test.That(t, captured, test.ShouldHaveLength, 1)
	test.That(t, captured[0], test.ShouldContainSubstring, "while")
	test.That(t, captured[0], test.ShouldContainSubstring, "whoops")
	captured = nil

	mainWithArgs = func(ctx context.Context, args []string, logger utils.ZapCompatibleLogger) error {
		return nil
	}
	exec = ContextualMain(mainWithArgs, nil, logger)
	exec.Start()
	<-exec.Ready
	exec.Stop()
	test.That(t, <-exec.Done, test.ShouldBeNil)
	test.That(t, captured, test.ShouldHaveLength, 0)

	exec = ContextualMain(mainWithArgs, nil, logger)
	exec.Start()
	<-exec.Ready
	exec.QuitSignal(t)
	exec.Stop()
	test.That(t, <-exec.Done, test.ShouldBeNil)
	test.That(t, captured, test.ShouldHaveLength, 1)
	test.That(t, captured[0], test.ShouldContainSubstring, "while")
	test.That(t, captured[0], test.ShouldNotContainSubstring, "whoops")
	captured = nil

	var capturedArgs []string
	mainWithArgs = func(ctx context.Context, args []string, logger utils.ZapCompatibleLogger) error {
		capturedArgs = args
		utils.ContextMainReadyFunc(ctx)()
		<-utils.ContextMainQuitSignal(ctx)
		return nil
	}

	exec = ContextualMain(mainWithArgs, []string{"1", "2", "3"}, logger)
	exec.Start()
	<-exec.Ready
	exec.QuitSignal(t)
	exec.Stop()
	test.That(t, <-exec.Done, test.ShouldBeNil)
	test.That(t, captured, test.ShouldHaveLength, 0)
	test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
	captured = nil

	mainWithArgs = func(ctx context.Context, args []string, logger utils.ZapCompatibleLogger) error {
		capturedArgs = args
		utils.ContextMainReadyFunc(ctx)()
		utils.ContextMainIterFunc(ctx)()
		utils.ContextMainIterFunc(ctx)()
		<-utils.ContextMainQuitSignal(ctx)
		return err1
	}

	exec = ContextualMain(mainWithArgs, []string{"1", "2", "3"}, logger)
	exec.ExpectIters(t, 2)
	exec.Start()
	<-exec.Ready
	exec.WaitIters(t)
	exec.QuitSignal(t)
	exec.Stop()
	test.That(t, <-exec.Done, test.ShouldEqual, err1)
	test.That(t, captured, test.ShouldHaveLength, 0)
	test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
	captured = nil
}

func TestTestMain(t *testing.T) {
	var captured []interface{}
	prevFatal := fatal
	prevError := tError
	defer func() {
		fatal = prevFatal
		tError = prevError
	}()
	var mu sync.Mutex
	fatal = func(t *testing.T, args ...interface{}) {
		t.Helper()
		mu.Lock()
		captured = args
		mu.Unlock()
	}
	tError = fatal

	err1 := errors.New("whoops")
	var capturedArgs []string
	mainWithArgs := func(ctx context.Context, args []string, logger utils.ZapCompatibleLogger) error {
		capturedArgs = args
		return err1
	}

	TestMain(t, mainWithArgs, []MainTestCase{
		{
			Name: "",
			Args: []string{"1", "2", "3"},
			Err:  err1.Error(),
			Before: func(t *testing.T, logger utils.ZapCompatibleLogger, exec *ContextualMainExecution) {
				t.Helper()
				captured = nil
				test.That(t, logger, test.ShouldNotBeNil)
			},
			After: func(t *testing.T, logs *observer.ObservedLogs) {
				t.Helper()
				test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
				test.That(t, captured, test.ShouldBeNil)
				test.That(t, logs, test.ShouldNotBeNil)
			},
		},
		{
			Name: "next",
			Args: []string{"1", "2", "3"},
			Err:  err1.Error(),
			Before: func(t *testing.T, logger utils.ZapCompatibleLogger, exec *ContextualMainExecution) {
				t.Helper()
				captured = nil
				test.That(t, logger, test.ShouldNotBeNil)
				logger.Info("hi")
			},
			During: func(ctx context.Context, t *testing.T, exec *ContextualMainExecution) {
				t.Helper()
				<-exec.Ready
				exec.QuitSignal(t)
				exec.Stop()
				test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
				test.That(t, exec, test.ShouldNotBeNil)
			},
			After: func(t *testing.T, logs *observer.ObservedLogs) {
				t.Helper()
				test.That(t, captured, test.ShouldHaveLength, 1)
				test.That(t, captured[0], test.ShouldContainSubstring, "while")
				test.That(t, captured[0], test.ShouldContainSubstring, "whoops")
				test.That(t, logs, test.ShouldNotBeNil)
				test.That(t, logs.FilterMessage("hi").All(), test.ShouldHaveLength, 1)
			},
		},
	})

	mainWithArgs = func(ctx context.Context, args []string, logger utils.ZapCompatibleLogger) error {
		capturedArgs = args
		utils.ContextMainReadyFunc(ctx)()
		<-utils.ContextMainQuitSignal(ctx)
		<-ctx.Done()
		return err1
	}
	TestMain(t, mainWithArgs, []MainTestCase{
		{
			Name: "",
			Args: []string{"1", "2", "3"},
			Err:  err1.Error(),
			Before: func(t *testing.T, logger utils.ZapCompatibleLogger, exec *ContextualMainExecution) {
				t.Helper()
				captured = nil
				test.That(t, logger, test.ShouldNotBeNil)
			},
			During: func(ctx context.Context, t *testing.T, exec *ContextualMainExecution) {
				t.Helper()
				<-exec.Ready
				exec.QuitSignal(t)
				test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
				test.That(t, exec, test.ShouldNotBeNil)
			},
			After: func(t *testing.T, logs *observer.ObservedLogs) {
				t.Helper()
				test.That(t, captured, test.ShouldBeNil)
				test.That(t, logs, test.ShouldNotBeNil)
			},
		},
		{
			Name: "next",
			Args: []string{"1", "2", "3"},
			Err:  err1.Error(),
			Before: func(t *testing.T, logger utils.ZapCompatibleLogger, exec *ContextualMainExecution) {
				t.Helper()
				captured = nil
				test.That(t, logger, test.ShouldNotBeNil)
				logger.Info("hi")
			},
			During: func(ctx context.Context, t *testing.T, exec *ContextualMainExecution) {
				t.Helper()
				<-exec.Ready
				exec.QuitSignal(t)
				test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
				test.That(t, exec, test.ShouldNotBeNil)
			},
			After: func(t *testing.T, logs *observer.ObservedLogs) {
				t.Helper()
				test.That(t, captured, test.ShouldHaveLength, 0)
				test.That(t, logs, test.ShouldNotBeNil)
				test.That(t, logs.FilterMessage("hi").All(), test.ShouldHaveLength, 1)
			},
		},
	})

	mainWithArgs = func(ctx context.Context, args []string, logger utils.ZapCompatibleLogger) error {
		capturedArgs = args
		utils.ContextMainReadyFunc(ctx)()
		utils.ContextMainIterFunc(ctx)()
		utils.ContextMainIterFunc(ctx)()
		<-utils.ContextMainQuitSignal(ctx)
		<-ctx.Done()
		return nil
	}
	TestMain(t, mainWithArgs, []MainTestCase{
		{
			Name: "",
			Args: []string{"1", "2", "3"},
			Err:  "",
			Before: func(t *testing.T, logger utils.ZapCompatibleLogger, exec *ContextualMainExecution) {
				t.Helper()
				captured = nil
				test.That(t, logger, test.ShouldNotBeNil)
				exec.ExpectIters(t, 2)
			},
			During: func(ctx context.Context, t *testing.T, exec *ContextualMainExecution) {
				t.Helper()
				<-exec.Ready
				exec.WaitIters(t)
				exec.QuitSignal(t)
				test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
				test.That(t, exec, test.ShouldNotBeNil)
			},
			After: func(t *testing.T, logs *observer.ObservedLogs) {
				t.Helper()
				test.That(t, captured, test.ShouldBeNil)
				test.That(t, logs, test.ShouldNotBeNil)
			},
		},
		{
			Name: "next",
			Args: []string{"1", "2", "3"},
			Err:  "",
			Before: func(t *testing.T, logger utils.ZapCompatibleLogger, exec *ContextualMainExecution) {
				t.Helper()
				captured = nil
				test.That(t, logger, test.ShouldNotBeNil)
				logger.Info("hi")
			},
			During: func(ctx context.Context, t *testing.T, exec *ContextualMainExecution) {
				t.Helper()
				<-exec.Ready
				exec.QuitSignal(t)
				test.That(t, capturedArgs, test.ShouldResemble, []string{"main", "1", "2", "3"})
				test.That(t, exec, test.ShouldNotBeNil)
			},
			After: func(t *testing.T, logs *observer.ObservedLogs) {
				t.Helper()
				test.That(t, captured, test.ShouldHaveLength, 0)
				test.That(t, logs, test.ShouldNotBeNil)
				test.That(t, logs.FilterMessage("hi").All(), test.ShouldHaveLength, 1)
			},
		},
	})
}
