package toolman

type config struct {
	stdsigs bool
	logDir  string
	logSpam bool
}

func initConfig(opts []InitOption) *config {
	cfg := &config{}

	for _, o := range opts {
		o(cfg)
	}

	return cfg
}

type InitOption func(c *config)

func LogSpam(spam bool) InitOption {
	return func(c *config) {
		c.logSpam = spam
	}
}

func LogDir(dir string) InitOption {
	return func(c *config) {
		c.logDir = dir
	}
}

func Quiet() InitOption {
	return func(c *config) {
		c.logSpam = false
		c.logDir = "/dev/null"
	}
}

func StandardSignals() InitOption {
	return func(c *config) {
		c.stdsigs = true
	}
}
