package ltsvreader

import (
	"bytes"
	"log"

	"github.com/monitoring-forge/ltsvparser"
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
func (r *Reader) Parse(data []byte) ([]byte, []byte) {
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
			pt = value
		case idx > 0:
			//status keyは先に指定したもの(iが小さい)を優先する
			if idx < stIndex {
				stIndex = idx
				st = value
			}
		}
		// If both ptime and status are found and the first status key is set, stop parsing early.
		if remaining == 0 || (pt != nil && st != nil && stIndex == 1) {
			return ltsvparser.Cancel
		}
		return nil
	}, r.keys...)
	if err != nil {
		log.Printf("Parse error: %v", err)
		return nil, nil
	}
	return pt, st

}
