package ltsvreader

import (
	"bytes"
	"log"

	"github.com/monitoring-forge/ltsvparser"
	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
)

// Reader struct
type Reader struct {
	keys [][]byte
}

// New :
func New(ptimeKey string, statusKeys []string) *Reader {
	keys := make([][]byte, 0, len(statusKeys)+1)
	keys = append(keys, []byte(ptimeKey))
	for _, stKey := range statusKeys {
		keys = append(keys, []byte(stKey))
	}
	return &Reader{keys}
}

var bHif = []byte("-")

// Parse
func (r *Reader) Parse(data []byte) (int, []byte, []byte) {
	c := 0
	var pt []byte
	var st []byte

	remaining := len(r.keys)
	stIndex := len(r.keys)
	var found uint64

	err := ltsvparser.Each(data, func(idx int, value []byte) error {
		// `-` はskip
		if bytes.Equal(value, bHif) || len(value) == 0 {
			return nil
		}
		bit := uint64(1) << idx
		if found&bit != 0 {
			return nil
		}
		found |= bit
		remaining--
		switch {
		case idx == 0:
			//ptime
			c = c | axslog.PtimeFlag
			pt = value
		case idx > 0:
			//status
			c = c | axslog.StatusFlag
			if idx < stIndex {
				stIndex = idx
				st = value
			}
		}
		if remaining == 0 {
			return ltsvparser.Cancel
		}
		return nil
	}, r.keys...)
	if err != nil {
		log.Printf("Parse error: %v", err)
	}
	return c, pt, st

}
