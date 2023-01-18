package jsonutils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
)

// DecodeBytes decode json byte array
func DecodeBytes(b []byte, v any) error {
	rd := bytes.NewReader(b)
	return DecodeJSON(rd, v)
}

// DecodeString decode json string
func DecodeString(str string, v any) error {
	rd := strings.NewReader(str)
	return DecodeJSON(rd, v)
}

// DecodeJSON decode from reader interface
func DecodeJSON(r io.Reader, v any) error {
	defer io.Copy(ioutil.Discard, r)
	d := json.NewDecoder(r)
	d.UseNumber()
	return d.Decode(v)
}

// ConvertJSON2Map convert into a flatted map
func ConvertJSON2Map(src map[string]any) (dst map[string]any) {
	if src == nil {
		return nil
	}
	dst = make(map[string]any)
	for key, value := range src {
		switch v := value.(type) {
		case json.Number:
			iv, err := v.Int64()
			if err == nil {
				dst[key] = iv
			} else {
				fv, err := v.Float64()
				if err == nil {
					dst[key] = fv
				} else {
					dst[key] = v.String()
				}
			}
		case map[string]any:
			dst[key] = ConvertJSON2Map(v)
		default:
			dst[key] = value
		}
	}
	return
}
