package helper

import (
	"bytes"
	"encoding/gob"
)

func Decode(buf []byte, out any) error {
	return gob.NewDecoder(bytes.NewReader(buf)).Decode(out)
}

func Encode(in any) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	err := gob.NewEncoder(buf).Encode(in)
	return buf.Bytes(), err
}
