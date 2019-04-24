package toolman

// NOTE: The *InitOption* functions defined here are (most often) evaluated as
//       they are being passed to toolman.Init(...), which is before flags are
//       parsed. The func's being returned however, are evaluated *after* flags
//       have been parsed.
//
// tl/dr; Define flags in outer funcs, reference them from inner funcs.
//        (see PIDFile as an example)
//
import (
	"time"

	"github.com/spf13/pflag"
)

type config struct {
	stdsigs     bool
	logDir      string
	mkLogDir    bool
	logFiles    bool
	logSpam     bool
	logToStderr bool
	logFlush    time.Duration
	pidfile     string
	flagSet     *pflag.FlagSet
}

func newConfig(opts []*InitOption) *config {
	cfg := &config{
		logSpam:  true,
		logFiles: true,
		flagSet:  pflag.CommandLine,
	}

	for _, o := range opts {
		if o.init != nil {
			o.init(cfg)
		}
	}

	pflag.CommandLine = cfg.flagSet

	return cfg
}

func (c *config) setup(opts []*InitOption) {
	for _, o := range opts {
		if o.setup != nil {
			o.setup(c)
		}
	}
}

type cfgFunc func(*config)

// An InitOption alters the behavior of toolman.Init.
type InitOption struct {
	// init functions are called *before* flag parsing
	init cfgFunc
	// setup functions are called *after* flags are parsed
	setup cfgFunc
	// applied is true when an If condition is true
	applied bool
}

func If(cond func() bool, options ...*InitOption) *InitOption {
	ret := &InitOption{}

	ret.setup = func(c *config) {
		if ret.applied = cond(); ret.applied {
			for _, opt := range options {
				opt.setup(c)
			}
		}
	}

	return ret
}

func (o *InitOption) Else(options ...*InitOption) *InitOption {
	return &InitOption{
		setup: func(c *config) {
			if !o.applied {
				for _, opt := range options {
					opt.setup(c)
				}
			}
		},
	}
}

// FlagSet returns an InitOption that makes fs the primary FlagSet for this
// application. All flags from the existing FlagSet will be merged into fs.
// Flag names already present will be overwritten by those from fs.
func FlagSet(fs *pflag.FlagSet) *InitOption {
	return &InitOption{
		init: func(c *config) {
			fs.AddFlagSet(c.flagSet)
			c.flagSet = fs
		},
	}
}

// AddFlagSet returns an InitOption that adds one or more additional FlagSets
// to the primary FlagSet for this application. Preexisting flags will not
// be overwritten.
func AddFlagSet(sets ...*pflag.FlagSet) *InitOption {
	return &InitOption{
		init: func(c *config) {
			for _, fs := range sets {
				c.flagSet.AddFlagSet(fs)
			}
		},
	}
}

func LogFlushInterval(d time.Duration) *InitOption {
	return &InitOption{init: func(c *config) { c.logFlush = d }}
}

// LogSpam returns an InitOption that enables (spam=true) or disables
// (spam=false) detailed information at the top of a program's log file.
func LogSpam(spam bool) *InitOption {
	return &InitOption{setup: func(c *config) { c.logSpam = spam }}
}

// LogDir returns an InitOption that sets the logging output directory to dir.
func LogDir(dir string) *InitOption {
	return &InitOption{setup: func(c *config) { c.logDir = dir }}
}

// MakeLogDir returns an InitOption indicating that missing log directories
// should be created. If dir creation fails, logging will be redirected to
// stderr automatically.
func MakeLogDir() *InitOption {
	return &InitOption{setup: func(c *config) { c.mkLogDir = true }}
}

func LogToStderr() *InitOption {
	return &InitOption{setup: func(c *config) { c.logToStderr = true; c.logFiles = false }}
}

// Quiet returns an InitOption that disables logging altogether.
func Quiet() *InitOption {
	return &InitOption{
		setup: func(c *config) {
			c.logSpam = false
			c.logFiles = false
		},
	}
}

// StandardSignals returns an InitOption that sets up signal handlers to
// shutdown the program on receipt of SIGHUP, SIGINT or SIGTERM. For more
// fine-grained control of shutdown behavior, see toolman.RegisterShutdown
// and toolman.ShutdownOn.
func StandardSignals() *InitOption {
	return &InitOption{setup: func(c *config) { c.stdsigs = true }}
}

var pidfilename *string

// PIDFile returns an InitOption that tells toolman.Init to write the current
// process ID to the file named by dflt. This InitOption also registers a new
// --pidfile flag that allows the user to change the PID file's path on
// invocation.
func PIDFile(dflt string) *InitOption {
	pidfilename = pflag.String("pidfile", dflt, "Path to file where PID is written")
	return &InitOption{setup: func(c *config) { c.pidfile = *pidfilename }}
}
