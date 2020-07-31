package version

import (
	"fmt"
	"runtime"
	"time"
)

// package global var, value set by linker using ldflag -X
//
// Makefile example:
//
// VERSION    :=`git describe --tags 2>/dev/null`
// COMMIT     :=`git rev-parse --short HEAD 2>/dev/null`
// DATE       :=`date "+%FT%T%z"`
//
// LDBASE     := github.com/gertd/mycd/pkg/vesion
// LDFLAGS    := -ldflags "-w -s -X $(LDBASE).ver=${VERSION} \
//		  						 -X $(LDBASE).date=${DATE} \
// 								 -X $(LDBASE).commit=${COMMIT}"
//
var (
	ver    string
	date   string
	commit string
)

// Info - version info.
type Info struct {
	Version string
	Date    string
	Commit  string
}

// GetInfo - get version stamp information.
func GetInfo() Info {
	if ver == "" {
		ver = "0.0.0"
	}

	if date == "" {
		date = time.Now().Format(time.RFC3339)
	}

	if commit == "" {
		commit = "develop"
	}

	return Info{
		Version: ver,
		Date:    date,
		Commit:  commit,
	}
}

// String() -- return version info string.
func (vi Info) String() string {
	return fmt.Sprintf("%s #%s-%s-%s [%s]",
		vi.Version,
		vi.Commit,
		runtime.GOOS,
		runtime.GOARCH,
		vi.Date,
	)
}
