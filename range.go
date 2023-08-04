package gcsproxy

import (
	"errors"
	"net/textproto"
	"strconv"
	"strings"
)

type bodyRange struct {
	offset int64
	length int64
}

func (r *bodyRange) isAll() bool {
	return r.offset == 0 && r.length == -1
}

var allReange = bodyRange{offset: 0, length: -1}

func parseRange(raw string) (bodyRange, error) {
	if raw == "" {
		return allReange, nil
	}
	if !strings.HasPrefix(raw, "bytes=") {
		return bodyRange{}, errors.New("unsupported unit")
	}
	if strings.IndexByte(raw, ',') != -1 {
		return bodyRange{}, errors.New("multiple range is not yet supported")
	}

	r := bodyRange{length: -1}
	raw = textproto.TrimString(strings.TrimPrefix(raw, "bytes="))
	if i := strings.IndexByte(raw, '-'); i == -1 {
		return bodyRange{}, errors.New("invalid format")
	} else if i == 0 { // Range: <unit>=-<suffix-length>
		var err error
		r.offset, err = strconv.ParseInt(raw[i:], 10, 64)
		if err != nil {
			return bodyRange{}, err
		}
		return r, nil
	} else {
		// Range: <unit>=<range-start>-
		// Range: <unit>=<range-start>-<range-end>
		var err error
		r.offset, err = strconv.ParseInt(raw[:i], 10, 64)
		if err != nil {
			return bodyRange{}, err
		}
		if i+1 == len(raw) {
			return r, nil
		}

		var end int64
		end, err = strconv.ParseInt(raw[i+1:], 10, 64)
		if err != nil {
			return bodyRange{}, err
		}
		if end < r.offset {
			return bodyRange{}, errors.New("incorrect number range")
		}

		r.length = end - r.offset + 1
		return r, nil
	}
}
