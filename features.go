package toolman

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"toolman.org/base/flagutil"
	"toolman.org/base/osutil"
	"toolman.org/base/signals"

  log "github.com/golang/glog"
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
	log.InfoDepth(2, fmt.Sprintf("  Start Time: %s", time.Now().Format(time.RFC3339Nano)))
	log.InfoDepth(2, fmt.Sprintf("  Process ID: %d", os.Getpid()))

	if dir, err := os.Getwd(); err != nil {
		info = fmt.Sprintf("not available: %v", err)
	} else {
		info = dir
	}
	log.InfoDepth(2, fmt.Sprintf(" Working Dir: %s", info))

	if u, err := user.Current(); err != nil {
		info = fmt.Sprintf("not available: %v", err)
	} else {
		info = fmt.Sprintf("%s [%s:%s]", u.Username, u.Uid, u.Gid)
	}
	log.InfoDepth(2, fmt.Sprintf("        User: %s", info))

	log.InfoDepth(2, fmt.Sprintf("Command Line: %s", os.Args[0]))
	if len(os.Args) > 1 {
		for _, a := range os.Args[1:] {
			log.InfoDepth(2, fmt.Sprintf("                  %s", a))
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

func (c *config) writePIDFile() {
	if c.pidfile == "" {
		return
	}

	if err := ioutil.WriteFile(c.pidfile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644); err != nil {
		log.Errorf("writing pid file %q: %v", c.pidfile, err)
		return
	}

	RegisterShutdown(func() {
		if err := os.Remove(c.pidfile); err != nil {
			log.Warning("pidfile %q not removed on shutdown: %v", c.pidfile, err)
		}
	})
}
