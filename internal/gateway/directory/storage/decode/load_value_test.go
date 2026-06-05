package decode

import (
	"errors"
	"testing"
)

type loadValueFixture struct {
	Name string `json:"name"`
}

type loadValueReader struct {
	value loadValueFixture
	err   error
}

func (r loadValueReader) Read(value any) error {
	if r.err != nil {
		return r.err
	}
	fixture, ok := value.(*loadValueFixture)
	if !ok {
		return errors.New("unexpected value type")
	}
	*fixture = r.value
	return nil
}

func TestLoadValueUsesEmptyValueAndEnsureHook(t *testing.T) {
	got, err := LoadValue(loadValueReader{value: loadValueFixture{Name: "raw"}}, func() loadValueFixture {
		return loadValueFixture{Name: "empty"}
	}, func(value loadValueFixture) loadValueFixture {
		value.Name += ":ensured"
		return value
	})
	if err != nil {
		t.Fatalf("LoadValue error: %v", err)
	}
	if got.Name != "raw:ensured" {
		t.Fatalf("LoadValue Name = %q, want ensured decoded value", got.Name)
	}
}

func TestLoadValueReturnsFreshEmptyValueOnReadError(t *testing.T) {
	got, err := LoadValue(loadValueReader{err: errors.New("read failed")}, func() loadValueFixture {
		return loadValueFixture{Name: "empty"}
	}, nil)
	if err == nil {
		t.Fatal("LoadValue error = nil, want read error")
	}
	if got.Name != "empty" {
		t.Fatalf("LoadValue value = %+v, want fresh empty value", got)
	}
}
