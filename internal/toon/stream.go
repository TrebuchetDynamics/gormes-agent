package toon

import "io"

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) EncodeJSON(raw []byte) error {
	out, err := EncodeJSON(raw)
	if err != nil {
		return err
	}
	_, err = e.w.Write(out)
	return err
}

func (e *Encoder) EncodeValue(value Value) error {
	out, err := EncodeValue(value)
	if err != nil {
		return err
	}
	_, err = e.w.Write(out)
	return err
}

func EncodeJSONTo(w io.Writer, raw []byte) error {
	return NewEncoder(w).EncodeJSON(raw)
}
