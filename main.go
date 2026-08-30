package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/mackerelio/golib/pluginutil"
	"github.com/monitoring-forge/flagrun"
	"github.com/monitoring-forge/followparser"
	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
)

var version string

var (
	// MaxReadSizeJSON : Maximum size for read
	MaxReadSizeJSON uint64 = 500 * 1000 * 1000
	// MaxReadSizeLTSV : Maximum size for read
	MaxReadSizeLTSV uint64 = 1000 * 1000 * 1000
	// MaxScanTokenSize : Maximum size for scan
	MaxScanTokenSize = 1 * 1024 * 1024 // 1MiB
	// StartBufSize : Initial buffer size for reading
	StartBufSize = 4096
)

// HumanBytes is a custom type to handle human-readable byte inputs
type HumanBytes uint64

// UnmarshalFlag parses human inputs like "10MB" or "2GiB" using go-humanize
func (hb *HumanBytes) UnmarshalFlag(value string) error {
	bytes, err := humanize.ParseBytes(value)
	if err != nil {
		return err
	}
	*hb = HumanBytes(bytes)
	return nil
}

type Opt struct {
	LogFile          string     `long:"logfile" description:"path to nginx ltsv logfiles. multiple log files can be specified, separated by commas." required:"true"`
	Format           string     `long:"format" default:"ltsv" description:"format of logfile. support json and ltsv"`
	KeyPrefix        string     `long:"key-prefix" description:"Metric key prefix" required:"true"`
	PtimeKey         string     `long:"ptime-key" default:"ptime" description:"key name for request_time"`
	StatusKeys       []string   `long:"status-key" default:"status" description:"key name for response status"`
	Filter           string     `long:"filter" default:"" description:"select lines contain a specified text from log"`
	SkipUntilBracket bool       `long:"skip-until-json" description:"skip reading until first { for json log with plain text header"`
	InvertFilter     bool       `long:"invert-filter" description:"select lines don't contain a specified text from log if a filter is specified"`
	MaxReadSize      HumanBytes `long:"max-read-size" description:"maximum size of log file to read (e.g. 10MB, 2GiB). 0 uses the default per format"`
	Quiet            bool       `short:"q" long:"quiet" description:"Suppress output"`
	Version          bool       `short:"v" long:"version" description:"Show version"`
	workdir          string
}

func (opt *Opt) getFileStats(posFile, logFile string) (*axslog.Stats, error) {
	stats := axslog.NewStats()
	if opt.Format != "ltsv" && opt.Format != "json" {
		return stats, fmt.Errorf("format %s is not supported", opt.Format)
	}
	maxReadSizeU := uint64(opt.MaxReadSize)
	if maxReadSizeU == 0 {
		if opt.Format == "ltsv" {
			maxReadSizeU = MaxReadSizeLTSV
		} else {
			maxReadSizeU = MaxReadSizeJSON
		}
	}
	if maxReadSizeU > uint64(^uint64(0)>>1) {
		return stats, fmt.Errorf("max-read-size %d overflows int64", maxReadSizeU)
	}
	maxReadSize := int64(maxReadSizeU)

	if opt.workdir == "" {
		opt.workdir = pluginutil.PluginWorkDir()
	}

	parser := opt.NewParser(stats)
	fp := &followparser.Parser{
		WorkDir:      opt.workdir,
		Callback:     parser,
		Silent:       opt.Quiet,
		MaxReadSize:  maxReadSize,
		StartBufSize: StartBufSize,
		MaxBufSize:   MaxScanTokenSize,
	}
	_, err := fp.Parse(posFile, logFile)

	return stats, err
}

func (opt *Opt) getStats() error {
	curUser, _ := user.Current()
	uid := "0"
	if curUser != nil {
		uid = curUser.Uid
	}

	logfiles := strings.Split(opt.LogFile, ",")

	if len(logfiles) == 1 {
		posFile := fmt.Sprintf("%s-axslog-v5-%s", uid, opt.KeyPrefix)
		stats, err := opt.getFileStats(posFile, opt.LogFile)
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
			stats, err := opt.getFileStats(posFile, logfile)
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

func (opt *Opt) Run(_ []string) (any, int) {
	err := opt.getStats()
	if err != nil {
		return fmt.Errorf("getStats: %w", err), flagrun.CRITICAL
	}
	return "", flagrun.OK
}

func main() {
	os.Exit(flagrun.Go(&Opt{}, flagrun.Version(version)))
}
