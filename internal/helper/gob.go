package helper

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

func Decode(buf []byte, out any) error {
	r := bytes.NewReader(buf)
	d := gob.NewDecoder(r)
	return d.Decode(out)
}

func DecodeAs[T any](buf []byte) (T, error) {
	var out T
	if err := Decode(buf, &out); err != nil {
		return out, err
	}
	return out, nil
}

func Encode(in any) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	err := gob.NewEncoder(buf).Encode(in)
	return buf.Bytes(), err
}

func MustEncode(in any) []byte {
	out, err := Encode(in)
	if err != nil {
		panic(fmt.Sprintf("cannot encode %#v: %v", in, err))
	}
	return out
}
