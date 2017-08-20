package toolman

import (
	"fmt"
	"os"
	"time"

	"toolman.org/base/runtimeutil"
	"toolman.org/base/signals"

	log "github.com/golang/glog"
)

// ShutdownFunc is a function that is registered for execution upon Shutdown or
// Abort.
type ShutdownFunc func()

type shutdownAction struct {
	downFunc   ShutdownFunc
	allowance  time.Duration
	onAbort    bool
	onShutdown bool
}

// ShutdownOption is used to modify behavior of ShutdownFuncs registered by
// RegisterShutdown.
type ShutdownOption func(sa *shutdownAction)

// OnShutdown returns a ShutdownOption indicating that the registered
// ShutdownFunc should only be executed on a clean shutdown (as opposed to
// an abort).
func OnShutdown() ShutdownOption {
	return func(sa *shutdownAction) {
		sa.onShutdown = true
	}
}

// OnAbort returns a ShutdownOption indicating that the registered
// ShutdownFunc should only be executed on an abnormal termination via Abort
// (as opposed to a clean shutdown).
func OnAbort() ShutdownOption {
	return func(sa *shutdownAction) {
		sa.onAbort = true
	}
}

// TimeAllowance returns a ShutdownOption that changes the default 100ms
// allotted for each ShutdownFunc. See Shutdown for details on how these
// TimeAllowance values are used.
func TimeAllowance(dur time.Duration) ShutdownOption {
	return func(sa *shutdownAction) {
		sa.allowance = dur
	}
}

// RegisterShutdown registers a ShutdownFunc to be executed when the program
// terminates via a call to Shutdown, Abort or receipt of a signal registered
// with ShutdownOn. Zero or more ShutdownOptions may also be provided to alter
// the behavior of the registered ShutdownFunc.
//
// Calling RegisterShutdown with no ShutdownOptions provided is equivalent to
// passing OnShutdown(), OnAbort() and TimeAllowance(100*time.Millisecond). If
// either OnShutdown or OnAbort is proided then the other will not be enabled
// unless it too is given.
func RegisterShutdown(sdf ShutdownFunc, opts ...ShutdownOption) {
	downmutex.Lock()
	defer downmutex.Unlock()

	if finalized {
		log.ErrorDepth(1, "Cannot register new Shutdown function after calling Shutdown()")
		return
	}

	sa := &shutdownAction{
		downFunc:  sdf,
		allowance: 100 * time.Millisecond,
	}

	for _, o := range opts {
		o(sa)
	}

	if !sa.onAbort && !sa.onShutdown {
		sa.onAbort = true
		sa.onShutdown = true
	}

	downactions = append(downactions, sa)
}

// Shutdown performs a clean termination of the current program and exits with
// a return code of 0. Prior to termination, all ShutdownFuncs registered with
// OnShutdown will be executed in reverse registration order and must complete
// within an allotted ammount of time.
//
// By default, each registered ShutdownFunc is given a time allowance of 100ms.
// When a shutdown occurs, and before any ShutdownFuncs are executed, a timer
// is initiated using the sum of all applicable ShutdownFunc time allowances.
// This is the total ammount of time allowed for shutdown operations; the
// program will terminate when all ShutdownFuncs have completed OR when this
// timer expires, whichever comes first.
func Shutdown() {
	shutdown(0, "")
}

// Abort terminates the running program and exits with a return code of 1. If
// the mesg argument is not the empty string, it will be issued on STDERR just
// prior to termination. Similar to Shutdown, Abort will execute all registered
// ShutdownFuncs enabled for abort and employs the same time allowance logic
// as Shutdown.
func Abort(mesg string) {
	shutdown(1, mesg)
}

// ShutdownOn causes Shutdown to be called when the current process receives
// one of the given signals.
func ShutdownOn(sigs ...os.Signal) {
	signals.RegisterHandler(func(os.Signal) bool { Shutdown(); return true }, sigs...)
}

func shutdown(code int, mesg string) {
	downmutex.Lock()
	defer downmutex.Unlock()
	if finalized {
		return
	}
	finalized = true

	// accumulate time allowances for all shutdown actions
	var ta time.Duration
	for _, da := range downactions {
		// TODO(tep): This should only sum allowances for applicable actions
		//            (using da.onShutdown and da.OnAbort)
		ta += da.allowance
	}
	ta += ta / 5 // fudge by an additional 20%

	done := make(chan struct{})

	// Call each shutdown action in reverse registration order.
	// Do this in a goroutine which closes the 'done' channel
	// at the end of the loop.
	go func() {
		defer close(done)
		for i := len(downactions) - 1; i >= 0; i-- {
			if log.V(1) {
				log.Infof("calling shutdown func: %v", runtimeutil.FuncID(downactions[i].downFunc))
			}
			downactions[i].downFunc()
		}
	}()

	tout := time.After(ta)

	// wait for either the goroutine to complete or the timer to expire,
	// whichever comes first.
	select {
	case <-done:
	case <-tout:
	}

	if mesg != "" {
		fmt.Fprintln(os.Stderr, mesg)
	}

	os.Exit(code)
}
