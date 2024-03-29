// Copyright 2017, 2018, 2019 Timothy E. Peoples
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

/*
Package toolman provides common initialization for toolman.org Go programs.

All Go programs for toolman.org should call "toolman.Init()" at or near
the beginning of function main() and then immediately defer a call to
"toolman.Shutdown()".  This provides a common mechanism for all toolman.org
libraries to register actions to be performed both at process startup
(via RegisterInit) and during normal termination (via RegisterShutdown).

Prior to executing any registered startup functions, toolman.Init() will
first ensure that logging is properly setup and that flags have been
parsed.

Program initialization may be customized by passing one or more InitOption
arguments to toolman.Init().

The following is a typical use case:

		package fnobish

		import "toolman.org/base/toolman"

		func init() {
			toolman.RegisterInit(func() {
				// Stuff to run on startup
			})

			toolman.RegisterShutdown(func() {
				// Stuff to run on shutdown
			})
		}

		func main() {
			toolman.Init(toolman.Quiet(), toolman.StandardSignals())
			defer toolman.Shutdown()

			// Do Stuff

		}
*/
package toolman // import "toolman.org/base/toolman/v2"

import (
	"sync"
	"time"

	"github.com/spf13/pflag"

	"toolman.org/base/log/v2"
)

var (
	initialized bool
	finalized   bool
	initfuncs   []InitFunc
	downactions []*shutdownAction
	initmutex   sync.Mutex
	downmutex   sync.Mutex
)

// InitFunc is a function registered via RegisterInit.
type InitFunc func()

// Init is the common initialization method for all toolman.org Go programs
// and should usually be the first call at the top of main().  Zero or more
// InitOptions may be provided to alter Init's behavior.
//
// Please note, Init may only be called once; any subsequent calls to Init
// will cause a panic.
func Init(opts ...*InitOption) {
	initmutex.Lock()
	defer initmutex.Unlock()
	defer func() { initialized = true }()

	if initialized {
		panic("toolman.Init() called multiple times!")
	}

	cfg := newConfig(opts)

	pflag.Parse()

	cfg.setup(opts)

	if cfg.stdsigs {
		setupStdSignals()
	}

	if err := cfg.setupLogging(); err != nil {
		panic(err)
	}

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

// DumbInit is deprecated, please use InitCLI instead.
func DumbInit() {
	Init(Quiet())
	log.Warning("DumbInit is deprecated; please switch to InitCLI")
}

// InitCLI calls the standard Init() routine with only the Quiet InitOption.
// This function is provided as a convenience for times when no logging is
// desired and no other options are needed, but the program still needs to
// process command line flags and execute InitFuncs registed by library code.
func InitCLI() {
	Init(Quiet())
}

// RegisterInit registers an initialization function to be executed by
// toolman.Init. Calls to RegisterInit are most commonly made from a library's
// own init() function but execution of the registered function does not happen
// until after toolman.Init has: a) merged and parsed all command line flags,
// b) processed all provided InitOptions and c) setup standard logging.
//
// RegisterInit will only register InitFuncs before calls to Init(); once
// Init has been called, RegisterInit will log an error message indicating
// that it refused to register the InitFunc.
//
// Functions registered via RegisterInit should avoid much heavy lifting as
// there are likely many and each one is executed upon startup of every
// participating Go program.  These functions also have no mechanism for
// dealing with error conditions (other than reporting them via standard
// logging) so risky operations should also be avoided. However, if your
// library simply cannot function properly due to an initialization failure,
// your registered functions should panic.
func RegisterInit(f InitFunc) {
	initmutex.Lock()
	defer initmutex.Unlock()

	if initialized {
		log.ErrorDepth(1, "InitFunc not registered after call to Init()")
		return
	}

	initfuncs = append(initfuncs, f)
}
