package decode

// Reader is the persisted-value read seam required by LoadValue.
type Reader interface {
	Read(value any) error
}

// LoadValue reads a persisted JSON value using a caller-supplied empty value
// and post-decode normalization hook. Directory stores use this to share the
// same decode lifecycle while keeping their own missing/invalid evidence policy.
func LoadValue[T any](reader Reader, empty func() T, ensure func(T) T) (T, error) {
	value := zeroOrEmpty(empty)
	if err := reader.Read(&value); err != nil {
		return zeroOrEmpty(empty), err
	}
	if ensure != nil {
		value = ensure(value)
	}
	return value, nil
}

func zeroOrEmpty[T any](empty func() T) T {
	if empty != nil {
		return empty()
	}
	var value T
	return value
}
