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
	return writeAll(e.w, out)
}

func (e *Encoder) EncodeValue(value Value) error {
	out, err := EncodeValue(value)
	if err != nil {
		return err
	}
	return writeAll(e.w, out)
}

func writeAll(w io.Writer, out []byte) error {
	n, err := w.Write(out)
	if err != nil {
		return err
	}
	if n != len(out) {
		return io.ErrShortWrite
	}
	return nil
}

func EncodeJSONTo(w io.Writer, raw []byte) error {
	return NewEncoder(w).EncodeJSON(raw)
}
