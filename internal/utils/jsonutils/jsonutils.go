package jsonutils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
)

func DecodeBytes(b []byte, v interface{}) error {
	rd := bytes.NewReader(b)
	return DecodeJSON(rd, v)
}

func DecodeString(str string, v interface{}) error {
	rd := strings.NewReader(str)
	return DecodeJSON(rd, v)
}

func DecodeJSON(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r)
	d := json.NewDecoder(r)
	d.UseNumber()
	return d.Decode(v)
}

func ConvertJson2Map(src map[string]interface{}) (dst map[string]interface{}) {
	if src == nil {
		return nil
	}
	dst = make(map[string]interface{})
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
		case map[string]interface{}:
			dst[key] = ConvertJson2Map(v)
		default:
			dst[key] = value
		}
	}
	return
}
