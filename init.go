// Copyright 2017 The Toolworks Development Trust. All rights reserved.
//
// Package toolman provides common initialization for toolman.org Go programs.
//
// All Go programs for toolman.org should call "toolman.Init()" at or near
// the beginning of function main() and then immediately defer a call to
// "toolman.Shutdown()".  This provides a common mechanism for all toolman.org
// libraries to register actions to be performed both at process startup
// (via RegisterInit) and during normal termination (via RegisterShutdown).
//
// Prior to executing any registered startup functions, toolman.Init() will
// first ensure that logging is properly setup and that flags have been
// parsed.
//
// Program initialization may be customized by passing one or more InitOption
// arguments to toolman.Init().
//
// The following is a typical use case:
//
// 		package fnobish
//
// 		import "toolman.org/base/toolman"
//
//		func init() {
//			toolman.RegisterInit(func(){
//				// Stuff to run on startup
//			})
//
//			toolman.RegisterShutdown(func(){
//				// Stuff to run on shutdown
//			})
//		}
//
// 		func main() {
// 			toolman.Init(toolman.Quiet(), toolman.StandardSignals())
// 			defer toolman.Shutdown()
//
// 			// Do Stuff
//
// 		}
//
package toolman // import "toolman.org/base/toolman"

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"sync"
	"syscall"
	"time"

	"toolman.org/base/flagutil"
	"toolman.org/base/osutil"
	"toolman.org/base/signals"

	log "github.com/golang/glog"
)

var (
	initialized bool
	finalized   bool
	initfuncs   []func()
	downactions []*shutdownAction
	initmutex   sync.Mutex
	downmutex   sync.Mutex
)

func RegisterInit(f func()) {
	initmutex.Lock()
	defer initmutex.Unlock()

	if initialized {
		log.ErrorDepth(1, "Cannot register new Init function after calling Init()")
		return
	}

	initfuncs = append(initfuncs, f)
}

func Init(opts ...InitOption) {
	initmutex.Lock()
	defer initmutex.Unlock()
	defer func() { initialized = true }()

	if initialized {
		panic("toolman.Init() called multiple times!")
	}

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg := initConfig(opts)

	if cfg.stdsigs {
		setupStdSignals()
	}

	cfg.setupLogging()

	if cfg.logSpam {
		addLogSpam()
	}

	if cfg.pidfile != "" {
		cfg.writePIDFile()
	}

	RegisterShutdown(func() { log.Flush(); time.Sleep(5 * time.Millisecond) })

	for _, f := range initfuncs {
		f()
	}
}
