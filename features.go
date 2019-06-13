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

package toolman

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"toolman.org/base/log/v2"
	"toolman.org/base/osutil"
	"toolman.org/base/signals"
)

const logDirFlag = "log_dir"

func (c *config) setupLogging() error {
	ld, isSet := flagIsSet(logDirFlag)
	if isSet {
		c.logDir = ld
	}

	if c.logDir == "" {
		c.logDir = osutil.FindEnvDefault("/tmp", "TOOLMAN_LOGDIR", "TMP", "TEMP")
	}

	if !isSet {
		if err := pflag.Set(logDirFlag, c.logDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to override --%s=%q: %v", logDirFlag, c.logDir, err)
		}

		if ldf := pflag.Lookup(logDirFlag); ldf != nil {
			ldf.Changed = false
		}
	}

	// A specified command-line flag overrides the config value
	if ulf, ok := flagIsSet("logfiles"); ok {
		if v, err := strconv.ParseBool(ulf); err == nil {
			c.logFiles = v
		}
	}

	if c.logToStderr {
		pflag.Set("logtostderr", "true")
	}

	if c.logFiles {
		if c.logFlush != 0 {
			log.UpdateFlushInterval(c.logFlush)
		}

		if c.mkLogDir {
			if err := os.MkdirAll(c.logDir, 0777); err != nil {
				pflag.Set("logtostderr", "true")
				log.Warning(err)
			}
		}
		log.Flush()
	} else {
		log.DisableLogFiles()
	}

	return nil
}

func flagIsSet(name string) (string, bool) {
	if f := pflag.Lookup(name); f != nil {
		return f.Value.String(), f.Changed
	}
	return "", false
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
			log.Warningf("pidfile %q not removed on shutdown: %v", c.pidfile, err)
		}
	})
}
