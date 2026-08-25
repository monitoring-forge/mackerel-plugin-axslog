package main

import (
	"bytes"
	"log"

	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
	"github.com/monitoring-forge/mackerel-plugin-axslog/jsonreader"
	"github.com/monitoring-forge/mackerel-plugin-axslog/ltsvreader"
)

type parser struct {
	opt    *Opt
	stats  *axslog.Stats
	ar     axslog.Reader
	filter []byte
}

func (opt *Opt) NewParser(stats *axslog.Stats) *parser {

	var ar axslog.Reader
	switch opt.Format {
	case "ltsv":
		ar = ltsvreader.New(opt.PtimeKey, opt.StatusKeys)
	case "json":
		ar = jsonreader.New(opt.PtimeKey, opt.StatusKeys)
	}

	p := &parser{
		opt:   opt,
		stats: stats,
		ar:    ar,
	}

	if opt.Filter != "" {
		p.filter = []byte(opt.Filter)
	}

	return p
}

func (p *parser) filtered(b []byte) bool {
	if p.filter == nil {
		return true
	}
	return !p.opt.InvertFilter == bytes.Contains(b, p.filter)
}

func (p *parser) Parse(b []byte) error {
	if !p.filtered(b) {
		return nil
	}
	if p.opt.SkipUntilBracket {
		i := bytes.IndexByte(b, '{')
		if i >= 0 {
			b = b[i:]
		}
	}
	c, pt, st := p.ar.Parse(b)
	if c&axslog.PtimeFlag == 0 {
		log.Printf("No ptime. continue key:%s", p.opt.PtimeKey)
		return nil
	}
	if c&axslog.StatusFlag == 0 {
		log.Printf("No status. continue key:%v", p.opt.StatusKeys)
		return nil
	}
	ptime, err := axslog.BFloat64(pt)
	if err != nil {
		log.Printf("Failed to convert ptime. continue: %v", err)
		return nil
	}
	status, err := axslog.BInt(st)
	if err != nil {
		log.Printf("Failed to convert status. continue: %v", err)
		return nil
	}
	p.stats.Append(ptime, status)
	return nil
}

func (p *parser) Finish(duration float64) {
	p.stats.SetDuration(duration)
}
