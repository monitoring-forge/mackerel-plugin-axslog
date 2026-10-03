package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
	"github.com/stretchr/testify/require"
)

func TestHumanBytesUnmarshalFlag(t *testing.T) {
	var hb HumanBytes
	if err := hb.UnmarshalFlag("10MB"); err != nil {
		t.Fatalf("UnmarshalFlag error: %v", err)
	}
	if hb != 10*1000*1000 {
		t.Errorf("HumanBytes = %d; want 10000000", hb)
	}
}

func TestHumanBytesUnmarshalFlagInvalid(t *testing.T) {
	var hb HumanBytes
	if err := hb.UnmarshalFlag("invalid"); err == nil {
		t.Error("UnmarshalFlag should return error for invalid input")
	}
}

func TestValidate(t *testing.T) {
	opt := &Opt{
		Percentiles: "50,90,99",
	}
	err := opt.validate(nil)
	require.NoError(t, err)
}

func TestValidateInvalid(t *testing.T) {
	opt := &Opt{
		Percentiles: "",
	}
	err := opt.validate(nil)
	require.Error(t, err)
}

func generateFile(b testing.TB, dir, filename string, numLines int, format string) {
	b.Helper()
	var template string
	switch format {
	case "json":
		template = `{"time": "%s", "status": "%d", "reqtime": "%.3f", "host": "%s", "req": "%s", "method": "%s", "size": "%d", "ua": "%s"}`
	case "ltsv":
		template = "time:%s\tstatus:%d\treqtime:%.3f\thost:%s\treq:%s\tmethod:%s\tsize:%d\tua:%s"
	default:
		b.Fatalf("unsupported format: %s", format)
	}
	filepath := fmt.Sprintf("%s/%s", dir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		b.Fatalf("error creating file: %v", err)
	}
	defer file.Close()
	r := rand.New(rand.NewPCG(1, 2))
	for i := range numLines {
		line := fmt.Sprintf(template,
			time.Now().Format(time.RFC3339),
			200+i%5,
			float64(r.IntN(500))/1000,
			"10.20.30.40",
			"GET /example/path HTTP/1.1",
			"GET",
			941,
			"Mozilla/5.0 (Linux; Android 4.4.2; SO-01F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.90 Mobile Safari/537.36",
		)
		_, err := file.WriteString(line + "\n")
		if err != nil {
			b.Fatalf("error writing to file: %v", err)
		}
	}
}

func generateJSONLFile(b testing.TB, dir, filename string, numLines int) {
	b.Helper()
	generateFile(b, dir, filename, numLines, "json")
}

func generateLTSVFile(b testing.TB, dir, filename string, numLines int) {
	b.Helper()
	generateFile(b, dir, filename, numLines, "ltsv")
}

func resetFollowParserStateFile(b testing.TB, dir, filename, posFile string) error {
	b.Helper()
	stateFilepath := fmt.Sprintf("%s-%d", posFile, os.Geteuid())
	stateFile, err := os.Create(filepath.Join(dir, stateFilepath))
	if err != nil {
		return err
	}
	defer stateFile.Close()

	filepath := fmt.Sprintf("%s/%s", dir, filename)
	stats, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	inode := stats.Sys().(*syscall.Stat_t).Ino
	dev := stats.Sys().(*syscall.Stat_t).Dev

	_, err = fmt.Fprintf(stateFile, `{"pos": %d, "time": %f, "inode": %d, "dev": %d}`, 0, float64(time.Now().Unix()-10), inode, dev)
	if err != nil {
		return err
	}

	return nil
}

func benchParserAndDisplay(b *testing.B, dir, filename string, numLines int, doOutput bool) {
	b.Helper()
	keyPrefix := "test"
	format := "json"
	if strings.HasSuffix(filename, ".ltsv") {
		format = "ltsv"
	}

	switch format {
	case "json":
		generateJSONLFile(b, dir, filename, numLines)
	case "ltsv":
		generateLTSVFile(b, dir, filename, numLines)
	}

	curUser, _ := user.Current()
	uid := "0"
	if curUser != nil {
		uid = curUser.Uid
	}
	posFile := fmt.Sprintf("%s-axslog-v5-%s", uid, keyPrefix)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		require.NoError(b, resetFollowParserStateFile(b, dir, filename, posFile))
		b.StartTimer()
		opt := &Opt{
			Format:     format,
			PtimeKey:   "reqtime",
			StatusKeys: []string{"status"},
			Filter:     "",
			LogFile:    fmt.Sprintf("%s/%s", dir, filename),
			KeyPrefix:  keyPrefix,
			Quiet:      true,
			workdir:    dir,
			percentiles: []axslog.PercentileTarget{
				{Name: "50_percentile", Value: 50.0},
				{Name: "90_percentile", Value: 90.0},
				{Name: "99_percentile", Value: 99.0},
			},
		}

		s, err := opt.getFileStats(posFile, opt.LogFile)
		require.NoError(b, err)
		require.NotNil(b, s)

		if doOutput {
			_ = s.Display(opt.percentiles, keyPrefix)
		}
		b.StopTimer()
		require.Equal(b, float64(numLines), s.Dump()["total"])
		b.StartTimer()
	}
}

// generate 100k JSONL file and parse benchmark
func BenchmarkMainParse_jsonl(b *testing.B) {
	tmpDir := b.TempDir()
	benchParserAndDisplay(b, tmpDir, "test.jsonl", 100000, false)
}

func BenchmarkMainParse_ltsv(b *testing.B) {
	tmpDir := b.TempDir()
	benchParserAndDisplay(b, tmpDir, "test.ltsv", 100000, false)
}

func BenchmarkMainParse_jsonl_and_output(b *testing.B) {
	tmpDir := b.TempDir()
	benchParserAndDisplay(b, tmpDir, "test.jsonl", 100000, true)
}

func BenchmarkMainParse_ltsv_and_output(b *testing.B) {
	tmpDir := b.TempDir()
	benchParserAndDisplay(b, tmpDir, "test.ltsv", 100000, true)
}
