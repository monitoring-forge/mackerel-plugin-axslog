package main

import (
	"strings"
	"testing"

	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
)

func TestFiltered(t *testing.T) {
	opt := &Opt{
		Filter:       "test",
		InvertFilter: false,
	}
	stats := axslog.NewStats()
	p := opt.NewParser(stats)

	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"Contains filter", []byte("This is a test log line."), true},
		{"Does not contain filter", []byte("This is a log line."), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.filtered(tt.input)
			if result != tt.expected {
				t.Errorf("filtered() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	opt := &Opt{
		Format:     "ltsv",
		PtimeKey:   "ptime",
		StatusKeys: []string{"status"},
	}
	stats := axslog.NewStats()
	p := opt.NewParser(stats)

	input := []byte("time:08/Mar/2017:14:12:40 +0900	status:200	ptime:0.030	host:10.20.30.40	req:GET /example/path HTTP/1.1	method:GET	size:941	ua:Mozilla/5.0 (Linux; Android 4.4.2; SO-01F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.90 Mobile Safari/537.36")
	err := p.Parse(input)
	if err != nil {
		t.Errorf("Parse() returned an error: %v", err)
	}

	dump := stats.Dump()
	if dump["total"] != 1.0 {
		t.Errorf("Total = %f; want 1.0", dump["total"])
	}
	if dump["c2xx"] != 1.0 {
		t.Errorf("C2xx = %f; want 1.0", dump["c2xx"])
	}
}

func TestParseMultipleStatusKeys(t *testing.T) {
	testCases := []struct {
		statusKeys  []string
		expected2xx float64
		expected4xx float64
	}{
		{[]string{"status", "http_status"}, 1.0, 0.0},
		{[]string{"http_status", "status"}, 0.0, 1.0},
	}
	for _, tt := range testCases {
		t.Run(strings.Join(tt.statusKeys, ","), func(t *testing.T) {
			opt := &Opt{
				Format:     "ltsv",
				PtimeKey:   "ptime",
				StatusKeys: tt.statusKeys,
			}
			stats := axslog.NewStats()
			p := opt.NewParser(stats)

			input := []byte("time:08/Mar/2017:14:12:40 +0900	status:200	http_status:404	ptime:0.030	host:10.20.30.40	req:GET /example/path HTTP/1.1	method:GET	size:941	ua:Mozilla/5.0 (Linux; Android 4.4.2; SO-01F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.90 Mobile Safari/537.36")
			err := p.Parse(input)
			if err != nil {
				t.Errorf("Parse() returned an error: %v", err)
			}

			dump := stats.Dump()
			if dump["total"] != 1.0 {
				t.Errorf("Total = %f; want 1.0", dump["total"])
			}
			if dump["c2xx"] != tt.expected2xx {
				t.Errorf("C2xx = %f; want %f", dump["c2xx"], tt.expected2xx)
			}
			if dump["c4xx"] != tt.expected4xx {
				t.Errorf("C4xx = %f; want %f", dump["c4xx"], tt.expected4xx)
			}

		})
	}
}

func TestValidateStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"Valid status code", []byte("200"), true},
		{"Valid status code 499", []byte("499"), true},
		{"Invalid status code - too short", []byte("20"), false},
		{"Invalid status code - non-numeric", []byte("2a0"), false},
		{"Invalid status code - too long", []byte("2000"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateStatusCode(tt.input)
			if result != tt.expected {
				t.Errorf("validateStatusCode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func BenchmarkParse_LTSVParse(b *testing.B) {
	opt := &Opt{
		Format:     "ltsv",
		PtimeKey:   "ptime",
		StatusKeys: []string{"status"},
		Filter:     "",
	}
	stats := axslog.NewStats()
	p := opt.NewParser(stats)

	data := []byte("time:08/Mar/2017:14:12:40 +0900	status:200	ptime:0.030	host:10.20.30.40	req:GET /example/path HTTP/1.1	method:GET	size:941	ua:Mozilla/5.0 (Linux; Android 4.4.2; SO-01F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.90 Mobile Safari/537.36")

	b.ReportAllocs()
	for b.Loop() {
		_ = p.Parse(data)
	}
}

func BenchmarkParse_JSONParse(b *testing.B) {
	opt := &Opt{
		Format:     "json",
		PtimeKey:   "ptime",
		StatusKeys: []string{"status"},
		Filter:     "",
	}
	stats := axslog.NewStats()
	p := opt.NewParser(stats)

	data := []byte(`{"time":"08/Mar/2017:14:12:40 +0900","status":"200","ptime":"0.030","host":"10.20.30.40","req":"GET /example/path HTTP/1.1","method":"GET","size":"941","ua":"Mozilla/5.0 (Linux; Android 4.4.2; SO-01F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.90 Mobile Safari/537.36"}`)

	b.ReportAllocs()
	for b.Loop() {
		_ = p.Parse(data)
	}
}
