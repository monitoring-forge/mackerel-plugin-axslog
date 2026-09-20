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

func generateJSONLFile(b testing.TB, dir, filename string, numLines int) error {
	b.Helper()
	filepath := fmt.Sprintf("%s/%s", dir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < numLines; i++ {
		line := fmt.Sprintf(`{"time": "%s", "status": "%d", "reqtime": "%f", "host": "%s", "req": "%s", "method": "%s", "size": "%d", "ua": "%s"}`,
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
			return err
		}
	}

	return nil
}

func generateLTSVFile(b testing.TB, dir, filename string, numLines int) error {
	b.Helper()
	filepath := fmt.Sprintf("%s/%s", dir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	r := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < numLines; i++ {
		line := fmt.Sprintf("time:%s\tstatus:%d\treqtime:%f\thost:%s\treq:%s\tmethod:%s\tsize:%d\tua:%s",
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
			return err
		}
	}

	return nil
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

	_, err = stateFile.WriteString(fmt.Sprintf(`{"pos": %d, "time": %f, "inode": %d, "dev": %d}`, 0, float64(time.Now().Unix()-10), inode, dev))
	if err != nil {
		return err
	}

	return nil
}

func benchParser(b *testing.B, dir, filename string, numLines int, doOutput bool) {
	b.Helper()
	keyPrefix := "test"
	format := "json"
	if strings.HasSuffix(filename, ".ltsv") {
		format = "ltsv"
	}

	if format == "json" {
		if err := generateJSONLFile(b, dir, filename, numLines); err != nil {
			b.Fatal(err)
		}
	} else if format == "ltsv" {
		if err := generateLTSVFile(b, dir, filename, numLines); err != nil {
			b.Fatal(err)
		}
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
		if err := resetFollowParserStateFile(b, dir, filename, posFile); err != nil {
			b.Fatal(err)
		}
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
		}

		s, err := opt.getFileStats(posFile, opt.LogFile)
		if err != nil {
			b.Fatal(err)
		}
		if doOutput {
			_ = s.Display(keyPrefix)
		}
		b.StopTimer()
		if s == nil {
			b.Fatal("Stats is nil")
		}
		if s.Dump()["total"] != float64(numLines) {
			b.Fatalf("Total = %f; want %f", s.Dump()["total"], float64(numLines))
		}
		b.StartTimer()
	}

	return
}

// generate 100k JSONL file and parse benchmark
func BenchmarkMainParse_jsonl(b *testing.B) {
	tmpDir := b.TempDir()
	benchParser(b, tmpDir, "test.jsonl", 100000, false)
}

func BenchmarkMainParse_ltsv(b *testing.B) {
	tmpDir := b.TempDir()
	benchParser(b, tmpDir, "test.ltsv", 100000, false)
}

func BenchmarkMainParse_jsonl_and_output(b *testing.B) {
	tmpDir := b.TempDir()
	benchParser(b, tmpDir, "test.jsonl", 100000, true)
}

func BenchmarkMainParse_ltsv_and_output(b *testing.B) {
	tmpDir := b.TempDir()
	benchParser(b, tmpDir, "test.ltsv", 100000, true)
}
