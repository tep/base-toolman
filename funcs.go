package toolman

import (
	"github.com/spf13/pflag"
)

func Args() []string {
	return pflag.Args()
}

func Arg(i int) string {
	return pflag.Arg(i)
}

func ArgDefault(i int, def string) string {
	if a := pflag.Arg(i); a != "" {
		return a
	}

	return def
}

func ArgList(i int) []string {
	a := pflag.Args()
	if len(a) >= i {
		return a[i:]
	}

	return nil
}
