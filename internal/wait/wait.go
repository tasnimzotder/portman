package wait

import (
	"context"
	"errors"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
	"github.com/tasnimzotder/portman/internal/scanner"
)

var ErrTimeout = errors.New("timed out waiting for port")

type Result struct {
	Success     bool
	Elapsed     time.Duration
	ProcessName string
	Err         error
}

type statusScanner interface {
	GetPortStatusContext(context.Context, int) (*model.Listener, error)
}

type contextScanner interface {
	GetPortContext(context.Context, int) (*model.Listener, error)
}

func Wait(s scanner.Scanner, port int, timeout, interval time.Duration, invert bool) Result {
	return WaitContext(context.Background(), s, port, timeout, interval, invert)
}

// WaitContext retries transient scan errors, but never interprets them as a free port.
// The deadline covers scans as well as polling delays.
func WaitContext(parent context.Context, s scanner.Scanner, port int, timeout, interval time.Duration, invert bool) Result {
	start := time.Now()
	finish := func(err error) Result { return Result{Elapsed: time.Since(start), Err: err} }
	if timeout <= 0 || interval <= 0 {
		return finish(errors.New("timeout and interval must be positive"))
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	var lastErr error
	for {
		var l *model.Listener
		var err error
		if ss, ok := s.(statusScanner); ok {
			l, err = ss.GetPortStatusContext(ctx, port)
		} else if cs, ok := s.(contextScanner); ok {
			l, err = cs.GetPortContext(ctx, port)
		} else {
			l, err = s.GetPort(port)
		}
		if ctx.Err() != nil {
			if parent.Err() != nil {
				return finish(parent.Err())
			}
			return finish(errors.Join(ErrTimeout, lastErr))
		}
		lastErr = err
		if err == nil && (l != nil) != invert {
			name := ""
			if l != nil {
				name = l.ProcessName()
				if name == "unknown" {
					name = ""
				}
			}
			return Result{Success: true, Elapsed: time.Since(start), ProcessName: name}
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if parent.Err() != nil {
				return finish(parent.Err())
			}
			return finish(errors.Join(ErrTimeout, lastErr))
		case <-timer.C:
		}
	}
}

func IsPortOpen(s scanner.Scanner, port int) bool {
	l, err := s.GetPort(port)
	return err == nil && l != nil
}
