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
	flag "github.com/spf13/pflag"
)

type config struct {
	stdsigs bool
	logDir  string
	logSpam bool
	pidfile string
}

func initConfig(opts []InitOption) *config {
	cfg := &config{
		logSpam: true,
	}

	for _, o := range opts {
		o(cfg)
	}

	return cfg
}

// An InitOption alters the behavior of toolman.Init.
type InitOption func(c *config)

// LogSpam returns an InitOption that enables (spam=true) or disables
// (spam=false) detailed information at the top of a program's log file.
func LogSpam(spam bool) InitOption {
	return func(c *config) {
		c.logSpam = spam
	}
}

// LogDir returns an InitOption that sets the logging output directory to dir.
func LogDir(dir string) InitOption {
	return func(c *config) {
		c.logDir = dir
	}
}

// Quiet returns an InitOption that disables logging altogether.
func Quiet() InitOption {
	return func(c *config) {
		c.logSpam = false
		c.logDir = "/dev/null"
	}
}

// StandardSignals returns an InitOption that sets up signal handlers to
// shutdown the program on receipt of SIGHUP, SIGINT or SIGTERM. For more
// fine-grained control of shutdown behavior, see toolman.RegisterShutdown
// and toolman.ShutdownOn.
func StandardSignals() InitOption {
	return func(c *config) {
		c.stdsigs = true
	}
}

var pidfilename *string

// PIDFile returns an InitOption that tells toolman.Init to write the current
// process ID to the file named by dflt. This InitOption also registers a new
// --pidfile flag that allows the user to change the PID file's path on
// invocation.
func PIDFile(dflt string) InitOption {
	pidfilename = flag.String("pidfile", dflt, "Path to file where PID is written")

	return func(c *config) {
		c.pidfile = *pidfilename
	}
}
