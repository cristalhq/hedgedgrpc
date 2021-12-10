package hedgedgrpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

const infiniteTimeout = 30 * 24 * time.Hour // domain specific infinite

// NewClient returns a new hedging unary client interceptor.
func NewClient(timeout time.Duration, upto int) (grpc.UnaryClientInterceptor, error) {
	interceptor, _, err := NewClientWithStats(timeout, upto)
	return interceptor, err
}

// NewClientWithStats returns a new hedging unary client interceptor with stats.
func NewClientWithStats(timeout time.Duration, upto int) (grpc.UnaryClientInterceptor, *Stats, error) {
	switch {
	case timeout < 0:
		return nil, nil, errors.New("timeout cannot be negative")
	case upto < 1:
		return nil, nil, errors.New("upto must be greater than 0")
	default:
		return newHedgedGRPC(timeout, upto)
	}
}

func newHedgedGRPC(timeout time.Duration, upto int) (grpc.UnaryClientInterceptor, *Stats, error) {
	stats := &Stats{}
	interceptor := func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		errOverall := &MultiError{}
		resultCh := make(chan indexedResp, upto)
		errorCh := make(chan error, upto)

		stats.requestedRoundTripsInc()

		resultIdx := -1
		cancels := make([]func(), upto)

		defer runInPool(func() {
			for i, cancel := range cancels {
				if i != resultIdx && cancel != nil {
					stats.canceledSubRequestsInc()
					cancel()
				}
			}
		})

		for sent := 0; len(errOverall.Errors) < upto; sent++ {
			if sent < upto {
				idx := sent
				var reply interface{}
				subCtx, cancel := reqWithCtx(parentCtx, idx != 0)
				cancels[idx] = cancel

				runInPool(func() {
					stats.actualRoundTripsInc()
					err := invoker(subCtx, method, req, reply, cc, opts...)
					if err != nil {
						stats.failedRoundTripsInc()
						errorCh <- err
					} else {
						resultCh <- indexedResp{idx, reply}
					}
				})
			}

			// all request sent - effectively disabling timeout between requests
			if sent == upto {
				timeout = infiniteTimeout
			}
			resp, err := waitResult(parentCtx, resultCh, errorCh, timeout)

			switch {
			case resp.Resp != nil:
				resultIdx = resp.Index
				return nil
			case parentCtx.Err() != nil:
				stats.canceledByUserRoundTripsInc()
				return parentCtx.Err()
			case err != nil:
				errOverall.Errors = append(errOverall.Errors, err)
			}
		}

		// all request have returned errors
		return errOverall
	}
	return interceptor, stats, nil
}

func waitResult(ctx context.Context, resultCh <-chan indexedResp, errorCh <-chan error, timeout time.Duration) (indexedResp, error) {
	// try to read result first before blocking on all other channels
	select {
	case res := <-resultCh:
		return res, nil
	default:
		timer := getTimer(timeout)
		defer returnTimer(timer)

		select {
		case res := <-resultCh:
			return res, nil

		case reqErr := <-errorCh:
			return indexedResp{}, reqErr

		case <-ctx.Done():
			return indexedResp{}, ctx.Err()

		case <-timer.C:
			return indexedResp{}, nil // it's not a request timeout, it's timeout BETWEEN consecutive requests
		}
	}
}

type indexedResp struct {
	Index int
	Resp  interface{}
}

func reqWithCtx(ctx context.Context, isHedged bool) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	if isHedged {
		ctx = context.WithValue(ctx, hedgedRequest{}, struct{}{})
	}
	return ctx, cancel
}

type hedgedRequest struct{}

// IsHedgedRequest reports when a request is hedged.
func IsHedgedRequest(ctx context.Context) bool {
	val := ctx.Value(hedgedRequest{})
	return val != nil
}

var taskQueue = make(chan func())

func runInPool(task func()) {
	select {
	case taskQueue <- task:
		// submited, everything is ok

	default:
		go func() {
			// do the given task
			task()

			const cleanupDuration = 10 * time.Second
			cleanupTicker := time.NewTicker(cleanupDuration)
			defer cleanupTicker.Stop()

			for {
				select {
				case t := <-taskQueue:
					t()
					cleanupTicker.Reset(cleanupDuration)
				case <-cleanupTicker.C:
					return
				}
			}
		}()
	}
}

// MultiError is an error type to track multiple errors. This is used to
// accumulate errors in cases and return them as a single "error".
// Insiper by https://github.com/hashicorp/go-multierror
type MultiError struct {
	Errors        []error
	ErrorFormatFn ErrorFormatFunc
}

func (e *MultiError) Error() string {
	fn := e.ErrorFormatFn
	if fn == nil {
		fn = listFormatFunc
	}
	return fn(e.Errors)
}

func (e *MultiError) String() string {
	return fmt.Sprintf("*%#v", e.Errors)
}

// ErrorOrNil returns an error if there are some.
func (e *MultiError) ErrorOrNil() error {
	switch {
	case e == nil || len(e.Errors) == 0:
		return nil
	default:
		return e
	}
}

// ErrorFormatFunc is called by MultiError to return the list of errors as a string.
type ErrorFormatFunc func([]error) string

func listFormatFunc(es []error) string {
	if len(es) == 1 {
		return fmt.Sprintf("1 error occurred:\n\t* %s\n\n", es[0])
	}

	points := make([]string, len(es))
	for i, err := range es {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf("%d errors occurred:\n\t%s\n\n", len(es), strings.Join(points, "\n\t"))
}

var timerPool = sync.Pool{New: func() interface{} {
	return time.NewTimer(time.Second)
}}

func getTimer(duration time.Duration) *time.Timer {
	timer := timerPool.Get().(*time.Timer)
	timer.Reset(duration)
	return timer
}

func returnTimer(timer *time.Timer) {
	timer.Stop()
	select {
	case <-timer.C:
	default:
	}
	timerPool.Put(timer)
}
