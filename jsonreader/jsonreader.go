package jsonreader

import (
	"bytes"
	"errors"

	"github.com/buger/jsonparser"
	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
)

// Reader represents a JSON reader that extracts ptime and status values based on specified keys.
type Reader struct {
	keys [][]byte
}

// New creates a new Reader instance with the specified ptime key and status keys.
func New(ptimeKey string, statusKeys []string) *Reader {
	keys := make([][]byte, 0, len(statusKeys)+1)
	keys = append(keys, []byte(ptimeKey))
	for _, stKey := range statusKeys {
		keys = append(keys, []byte(stKey))
	}
	return &Reader{keys}
}

var bHif = []byte("-")
var errFlatPathsFound = errors.New("all flat JSON paths found")

// Parse parses the given JSON data and returns the flags, ptime, and status values.
// It returns axslog.PtimeFlag and axslog.StatusFlag based on the presence of the corresponding keys.
// If a key is not found or its value is "-", it is skipped.
// nolint:gocognit
func (r *Reader) Parse(data []byte) (int, []byte, []byte) {
	c := 0
	var pt []byte
	var st []byte

	remaining := len(r.keys)
	stIndex := len(r.keys)
	var found uint64

	err := jsonparser.ObjectEach(data, func(key, value []byte, valueType jsonparser.ValueType, _ int) error {
		// `-` はskip
		if bytes.Equal(value, bHif) || len(value) == 0 {
			return nil
		}
		for i, k := range r.keys {
			bit := uint64(1) << i
			if found&bit != 0 || !bytes.Equal(key, k) {
				continue
			}
			found |= bit
			remaining--
			switch i {
			case 0:
				// ptime key
				c = c | axslog.PtimeFlag
				pt = value
			default:
				// status keys
				// status は先に指定したもの(iが小さい)を優先する
				c = c | axslog.StatusFlag
				if i < stIndex {
					stIndex = i
					st = value
				}
			}
		}
		if remaining == 0 {
			return errFlatPathsFound
		}
		return nil
	})
	if err != nil && err != errFlatPathsFound { //nolint:errorlint
		return 0, []byte(""), []byte("")
	}

	return c, pt, st
}
