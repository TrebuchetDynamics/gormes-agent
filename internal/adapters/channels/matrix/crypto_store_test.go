package matrix

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMatrixCryptoStoreDeviceIDBindingBeforeAccountPut(t *testing.T) {
	store := &fakeMatrixCryptoStore{}
	evidence := BindMatrixCryptoStoreDeviceID(context.Background(), store, "DEV1")
	if evidence.Evidence != "" {
		t.Fatalf("BindMatrixCryptoStoreDeviceID evidence = %+v, want success", evidence)
	}
	store.PutAccount()

	want := []string{"put_device_id:DEV1", "put_account"}
	if !reflect.DeepEqual(store.calls, want) {
		t.Fatalf("crypto store calls = %#v, want %#v", store.calls, want)
	}
}

func TestMatrixCryptoStoreDeviceIDBindingRequiresDeviceID(t *testing.T) {
	store := &fakeMatrixCryptoStore{}
	evidence := BindMatrixCryptoStoreDeviceID(context.Background(), store, " ")
	if evidence.Evidence != MatrixEvidenceE2EEUnavailable {
		t.Fatalf("blank device evidence = %+v, want %s", evidence, MatrixEvidenceE2EEUnavailable)
	}
	if !strings.Contains(evidence.Error, "device_id") {
		t.Fatalf("blank device error = %q, want device_id guidance", evidence.Error)
	}
	if len(store.calls) != 0 {
		t.Fatalf("blank device should not touch store, got calls %#v", store.calls)
	}
}

func TestMatrixCryptoStoreDeviceIDBindingReportsStoreFailure(t *testing.T) {
	store := &fakeMatrixCryptoStore{err: errors.New("db locked")}
	evidence := BindMatrixCryptoStoreDeviceID(context.Background(), store, "DEV1")
	if evidence.Evidence != MatrixEvidenceE2EEUnavailable {
		t.Fatalf("store error evidence = %+v, want %s", evidence, MatrixEvidenceE2EEUnavailable)
	}
	if !strings.Contains(evidence.Error, "db locked") {
		t.Fatalf("store error = %q, want wrapped store error", evidence.Error)
	}
}

type fakeMatrixCryptoStore struct {
	calls []string
	err   error
}

func (f *fakeMatrixCryptoStore) PutDeviceID(_ context.Context, deviceID string) error {
	f.calls = append(f.calls, "put_device_id:"+deviceID)
	return f.err
}

func (f *fakeMatrixCryptoStore) PutAccount() {
	f.calls = append(f.calls, "put_account")
}
