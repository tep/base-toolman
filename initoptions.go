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
	"github.com/spf13/pflag"
	"toolman.org/base/flagutil"
)

type config struct {
	stdsigs  bool
	logDir   string
	logFiles bool
	logSpam  bool
	pidfile  string
	flags    *flagutil.FlagsGroup
}

func newConfig(opts []*InitOption) *config {
	cfg := &config{
		flags:    flagutil.NewFlagsGroup(),
		logSpam:  true,
		logFiles: true,
	}

	for _, o := range opts {
		if o.init != nil {
			o.init(cfg)
		}
	}

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
}

// FlagSet returns an InitOption that makes fs the primary FlagSet
// for this app's FlagsGroup.
func FlagSet(fs *pflag.FlagSet) *InitOption {
	return &InitOption{func(c *config) { c.flags.SetPrimary(fs) }, nil}
}

// AddFlagSet returns an InitOption that adds the given FlagSet to the
// group of FlagSets being processed by this application.  This is useful
// for integration with frameworks that provide their own flag parsing.
func AddFlagSet(sets ...*pflag.FlagSet) *InitOption {
	return &InitOption{func(c *config) {
		for _, fs := range sets {
			c.flags.AddFlagSet(fs)
		}
	}, nil}
}

// LogSpam returns an InitOption that enables (spam=true) or disables
// (spam=false) detailed information at the top of a program's log file.
func LogSpam(spam bool) *InitOption {
	return &InitOption{nil, func(c *config) { c.logSpam = spam }}
}

// LogDir returns an InitOption that sets the logging output directory to dir.
func LogDir(dir string) *InitOption {
	return &InitOption{nil, func(c *config) { c.logDir = dir }}
}

// Quiet returns an InitOption that disables logging altogether.
func Quiet() *InitOption {
	return &InitOption{nil, func(c *config) {
		c.logSpam = false
		c.logFiles = false
	}}
}

// StandardSignals returns an InitOption that sets up signal handlers to
// shutdown the program on receipt of SIGHUP, SIGINT or SIGTERM. For more
// fine-grained control of shutdown behavior, see toolman.RegisterShutdown
// and toolman.ShutdownOn.
func StandardSignals() *InitOption {
	return &InitOption{nil, func(c *config) {
		c.stdsigs = true
	}}
}

var pidfilename *string

// PIDFile returns an InitOption that tells toolman.Init to write the current
// process ID to the file named by dflt. This InitOption also registers a new
// --pidfile flag that allows the user to change the PID file's path on
// invocation.
func PIDFile(dflt string) *InitOption {
	pidfilename = pflag.String("pidfile", dflt, "Path to file where PID is written")

	return &InitOption{nil, func(c *config) {
		c.pidfile = *pidfilename
	}}
}
