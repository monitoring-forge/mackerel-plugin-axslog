package jsonreader

import (
	"bytes"
	"errors"
	"log"

	"github.com/buger/jsonparser"
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
func (r *Reader) Parse(data []byte) ([]byte, []byte) {
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
				pt = value
			default:
				// status key は先に指定したもの(iが小さい)を優先する
				if i < stIndex {
					stIndex = i
					st = value
				}
			}
		}
		// If both ptime and status are found and the first status key is set, stop parsing early.
		if remaining == 0 || (pt != nil && st != nil && stIndex == 1) {
			return errFlatPathsFound
		}
		return nil
	})
	if err != nil && err != errFlatPathsFound { //nolint:errorlint
		log.Printf("Parse error: %v", err)
		return nil, nil
	}

	return pt, st
}
