package toolman

import (
	"fmt"
	"os"
	"time"

	"toolman.org/base/runtimeutil"
	"toolman.org/base/signals"

	log "github.com/golang/glog"
)

type shutdownAction struct {
	downFunc   func()
	allowance  time.Duration
	onAbort    bool
	onShutdown bool
}

type ShutdownOption func(sa *shutdownAction)

func OnShutdown() ShutdownOption {
	return func(sa *shutdownAction) {
		sa.onShutdown = true
	}
}

func OnAbort() ShutdownOption {
	return func(sa *shutdownAction) {
		sa.onAbort = true
	}
}

func TimeAllowance(dur time.Duration) ShutdownOption {
	return func(sa *shutdownAction) {
		sa.allowance = dur
	}
}

func RegisterShutdown(f func(), opts ...ShutdownOption) {
	downmutex.Lock()
	defer downmutex.Unlock()

	if finalized {
		log.ErrorDepth(1, "Cannot register new Shutdown function after calling Shutdown()")
		return
	}

	sa := &shutdownAction{
		downFunc:  f,
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

func Shutdown() {
	shutdown(0, "")
}

func Abort(mesg string) {
	shutdown(1, mesg)
}

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
