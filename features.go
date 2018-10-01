package toolman

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"toolman.org/base/log"
	"toolman.org/base/osutil"
	"toolman.org/base/signals"
)

func (c *config) setupLogging() error {
	ld, isSet := c.flags.ValueIsSet("log_dir")
	if isSet {
		c.logDir = ld
	}

	if c.logDir == "" {
		c.logDir = osutil.FindEnvDefault("/tmp", "TOOLMAN_LOGDIR", "TMP", "TEMP")
	}

	if !isSet {
		if err := c.flags.Set("log_dir", c.logDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to override --log_dir=%q: %v", c.logDir, err)
		}
	}

	// A specified command-line flag overrides the config value
	if ulf, ok := c.flags.ValueIsSet("logfiles"); ok {
		if v, err := strconv.ParseBool(ulf); err == nil {
			c.logFiles = v
		}
	}

	if c.logToStderr {
		c.flags.Set("logtostderr", "true")
	}

	if c.logFiles {
		if c.mkLogDir {
			if err := os.MkdirAll(c.logDir, 0777); err != nil {
				c.flags.Set("logtostderr", "true")
				log.Warning(err)
			}
		}
		log.Flush()
	} else {
		log.DisableLogFiles()
	}

	return nil
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
		for i, a := range os.Args[1:] {
			log.InfoDepth(2, fmt.Sprintf("              %2d) %s", i+1, a))
		}
	}
}

func setupStdSignals() {
	signals.RegisterSoftHandler(func(os.Signal) bool {
		log.Infof("shutting down now")
		go func() {
			signals.Stop()
			Shutdown()
		}()
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
