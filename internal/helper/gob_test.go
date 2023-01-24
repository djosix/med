package helper

import "testing"

func TestGobEncodeDecode(t *testing.T) {
	type Inner struct {
		Int    int
		String string
	}
	type Outer struct {
		Int    int
		String string
		Inner  Inner
	}
	in := Outer{
		Int:    123,
		String: "outer",
		Inner: Inner{
			Int:    321,
			String: "inner",
		},
	}

	if data, err := Encode(in); err != nil {
		t.Error(err)
	} else {
		var out Outer

		// encode T decode &T
		if err := Decode(data, &out); err != nil {
			t.Error(err)
		} else if in != out {
			t.Error("not equal")
		}

		// encode T decode as T
		if out, err := DecodeAs[Outer](data); err != nil {
			t.Error(err)
		} else if in != out {
			t.Error("not equal")
		}
	}

	if data, err := Encode(&in); err != nil {
		t.Error(err)
	} else {
		var out Outer

		// encode &T decode &T
		if err := Decode(data, &out); err != nil {
			t.Error(err)
		} else if in != out {
			t.Error("not equal")
		}

		// encode &T decode as T
		if out, err := DecodeAs[Outer](data); err != nil {
			t.Error(err)
		} else if in != out {
			t.Error("not equal")
		}
	}
}
