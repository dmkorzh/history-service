package parser

import (
	"errors"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type (
	RawCall    []byte
	ParsedCall []byte
)

const (
	TimeISO8601 = "2006-01-02T15:04:05.999Z"
	TimeResult  = "2006-01-02 15:04:05.000 UTC"
)

func (c RawCall) Get(key string) gjson.Result {
	return gjson.GetBytes(c, key)
}

func getTime(r gjson.Result) (time.Time, error) {
	if t, err := time.ParseInLocation(TimeISO8601, r.String(), time.UTC); err == nil {
		return t, nil
	} else {
		return time.Time{}, errors.New("unknown time format")
	}
}

var _sopt = sjson.Options{
	Optimistic:     true,
	ReplaceInPlace: false,
}

func (pc *ParsedCall) Set(key string, value interface{}) {
	switch v := value.(type) {
	case time.Time:
		*pc, _ = sjson.SetBytesOptions(*pc, key, v.Format(TimeResult), &_sopt)
	default:
		*pc, _ = sjson.SetBytesOptions(*pc, key, value, &_sopt)
	}
}

func (pc *ParsedCall) SetDuration(start, connect, disconnect time.Time) {
	if connect.IsZero() {
		pc.Set("waiting", uint32(disconnect.Sub(start).Milliseconds()))
		pc.Set("duration", 0)
	} else {
		pc.Set("waiting", uint32(connect.Sub(start).Milliseconds()))
		pc.Set("duration", uint32(disconnect.Sub(connect).Milliseconds()))
	}
}
