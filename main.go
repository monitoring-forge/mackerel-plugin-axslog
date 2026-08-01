package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/mackerelio/golib/pluginutil"
	"github.com/monitoring-forge/followparser"
	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
)

var version string
var commit string

const (
	OK = iota
	WARNING
	CRITICAL
	UNKNOWN
)

var (
	// MaxReadSizeJSON : Maximum size for read
	maxReadSizeJSON int64 = 500 * 1000 * 1000

	// MaxReadSizeLTSV : Maximum size for read
	maxReadSizeLTSV int64 = 1000 * 1000 * 1000

	maxScanTokenSize = 1 * 1024 * 1024 // 1MiB
	startBufSize     = 4096
)

func getFileStats(opt *axslog.Opt, posFile, logFile string) (*axslog.Stats, error) {
	stats := axslog.NewStats()
	if opt.Format != "ltsv" && opt.Format != "json" {
		return stats, fmt.Errorf("format %s is not supported", opt.Format)
	}
	maxReadSizeU := uint64(opt.MaxReadSize)
	if maxReadSizeU == 0 {
		if opt.Format == "ltsv" {
			maxReadSizeU = uint64(maxReadSizeLTSV)
		} else {
			maxReadSizeU = uint64(maxReadSizeJSON)
		}
	}
	if maxReadSizeU > uint64(^uint64(0)>>1) {
		return stats, fmt.Errorf("max-read-size %d overflows int64", maxReadSizeU)
	}
	maxReadSize := int64(maxReadSizeU)

	parser := NewParser(opt, stats)
	fp := &followparser.Parser{
		WorkDir:      pluginutil.PluginWorkDir(),
		Callback:     parser,
		Silent:       false,
		MaxReadSize:  maxReadSize,
		StartBufSize: startBufSize,
		MaxBufSize:   maxScanTokenSize,
	}
	_, err := fp.Parse(posFile, logFile)

	return stats, err
}

func getStats(opt *axslog.Opt) error {
	curUser, _ := user.Current()
	uid := "0"
	if curUser != nil {
		uid = curUser.Uid
	}

	logfiles := strings.Split(opt.LogFile, ",")

	if len(logfiles) == 1 {
		posFile := fmt.Sprintf("%s-axslog-v5-%s", uid, opt.KeyPrefix)
		stats, err := getFileStats(opt, posFile, opt.LogFile)
		if err != nil {
			return err
		}
		out := stats.Display(opt.KeyPrefix)
		_, printErr := fmt.Print(out)
		return printErr
	}

	sCh := make(chan axslog.StatsCh, len(logfiles))
	defer close(sCh)
	for _, l := range logfiles {
		logfile := l
		go func(logfile string) {
			escaped := url.PathEscape(logfile)
			posFile := fmt.Sprintf("%s-axslog-v5-%s-%s", uid, opt.KeyPrefix, escaped)
			stats, err := getFileStats(opt, posFile, logfile)
			sCh <- axslog.StatsCh{
				Stats:   stats,
				Logfile: logfile,
				Err:     err,
			}
		}(logfile)
	}
	errCnt := 0
	var statsAll []*axslog.Stats
	for range logfiles {
		s := <-sCh
		if s.Err != nil {
			errCnt++
			if len(logfiles) == errCnt {
				return s.Err
			}
			// warnings and ignore
			log.Printf("getStats file:%s :%v", s.Logfile, s.Err)
		} else {
			statsAll = append(statsAll, s.Stats)
		}
	}

	out := axslog.DisplayAll(statsAll, opt.KeyPrefix)
	_, printErr := fmt.Print(out)
	return printErr
}

func main() {
	os.Exit(_main())
}

func _main() int {
	opt := &axslog.Opt{}
	psr := flags.NewParser(opt, flags.HelpFlag|flags.PassDoubleDash)
	_, err := psr.Parse()
	if opt.Version {
		if commit == "" {
			commit = "dev"
		}
		fmt.Printf(
			"%s-%s\n%s/%s, %s, %s\n",
			filepath.Base(os.Args[0]),
			version,
			runtime.GOOS,
			runtime.GOARCH,
			runtime.Version(),
			commit)
		return OK
	} else if flags.WroteHelp(err) {
		fmt.Fprintf(os.Stdout, "%v\n", err)
		return OK
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return UNKNOWN
	}

	err = getStats(opt)
	if err != nil {
		log.Printf("getStats: %v", err)
		return CRITICAL
	}
	return OK
}
