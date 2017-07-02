package toolman

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"syscall"
	"time"

	"toolman.org/base/flagutil"
	"toolman.org/base/osutil"
	"toolman.org/base/signals"
)

func (c *config) setupLogging() {
	ld, isSet := flagutil.ValueIsSet("log_dir")

	if isSet {
		c.logDir = ld
	}

	if c.logDir == "" {
		c.logDir = osutil.FindEnvDefault("/tmp", "TOOLMAN_LOGDIR", "TMP", "TEMP")
	}

	if !isSet {
		if err := flag.Set("log_dir", c.logDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to override --log_dir=%q: %v", c.logDir, err)
		}
	}

}

func addLogSpam() {
	var info string
	log.Infof("  Start Time: %s", time.Now().Format(time.RFC3339Nano))
	log.Infof("  Process ID: %d", os.Getpid())

	if dir, err := os.Getwd(); err != nil {
		info = fmt.Sprintf("not available: %v", err)
	} else {
		info = dir
	}
	log.Infof(" Working Dir: %s", info)

	if u, err := user.Current(); err != nil {
		info = fmt.Sprintf("not available: %v", err)
	} else {
		info = fmt.Sprintf("%s [%s:%s]", u.Username, u.Uid, u.Gid)
	}
	log.Infof("        User: %s", info)

	log.Infof("Command Line: %s", os.Args[0])
	if len(os.Args) > 1 {
		for _, a := range os.Args[1:] {
			log.Infof("                  %s", a)
		}
	}
}

func setupStdSignals() {
	signals.RegisterSoftHandler(func(os.Signal) bool {
		log.Infof("shutting down")
		signals.Stop()
		Shutdown()
		return true
	}, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
}
