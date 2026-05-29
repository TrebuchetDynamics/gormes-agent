package cron

import (
	"bytes"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
)

func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "session.db")
	db, err := bbolt.Open(dbPath, 0o600, nil)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(db)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return s, func() { _ = db.Close() }
}

func TestStore_CreateAndGet(t *testing.T) {
	s, done := newTestStore(t)
	defer done()

	j := NewJob("morning", "0 8 * * *", "status")
	if err := s.Create(j); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(j.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "morning" || got.Schedule != "0 8 * * *" || got.Prompt != "status" {
		t.Errorf("got = %+v, want name/sched/prompt intact", got)
	}
}

func TestStore_List(t *testing.T) {
	s, done := newTestStore(t)
	defer done()
	_ = s.Create(NewJob("a", "@daily", "x"))
	_ = s.Create(NewJob("b", "@hourly", "y"))
	_ = s.Create(NewJob("c", "@every 1m", "z"))
	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestStore_Update(t *testing.T) {
	s, done := newTestStore(t)
	defer done()
	j := NewJob("m", "@daily", "p")
	_ = s.Create(j)
	j.Paused = true
	j.LastRunUnix = 1700000000
	j.LastStatus = "success"
	if err := s.Update(j); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get(j.ID)
	if !got.Paused || got.LastRunUnix != 1700000000 || got.LastStatus != "success" {
		t.Errorf("after Update, got = %+v", got)
	}
}

func TestStore_Delete(t *testing.T) {
	s, done := newTestStore(t)
	defer done()
	j := NewJob("x", "@daily", "y")
	_ = s.Create(j)
	if err := s.Delete(j.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(j.ID); err == nil {
		t.Error("Get after Delete returned no error, want ErrJobNotFound")
	}
}

func TestStore_GetMissingReturnsTypedError(t *testing.T) {
	s, done := newTestStore(t)
	defer done()
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing id")
	}
	if err != ErrJobNotFound {
		t.Errorf("err = %v, want ErrJobNotFound", err)
	}
}

func TestStore_CreateRejectsDuplicateName(t *testing.T) {
	s, done := newTestStore(t)
	defer done()
	_ = s.Create(NewJob("same", "@daily", "p1"))
	err := s.Create(NewJob("same", "@hourly", "p2"))
	if err == nil {
		t.Fatal("expected error on duplicate name")
	}
	if err != ErrJobNameTaken {
		t.Errorf("err = %v, want ErrJobNameTaken", err)
	}
}

func TestStore_ListNormalizesPartialLegacyRecords(t *testing.T) {
	s, done := newTestStore(t)
	defer done()

	key := "abc123deadbe"
	raw := []byte(`{"name":null,"prompt":null,"schedule":null,"paused":false}`)
	seedRawCronJob(t, s, key, raw)

	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ID != key {
		t.Fatalf("ID = %q, want bucket key %q", got[0].ID, key)
	}
	if got[0].Name != key {
		t.Fatalf("Name = %q, want fallback to id %q", got[0].Name, key)
	}
	if got[0].Prompt != "" {
		t.Fatalf("Prompt = %q, want empty string", got[0].Prompt)
	}
	if got[0].Schedule != "?" {
		t.Fatalf("Schedule = %q, want ?", got[0].Schedule)
	}
	assertRawCronJobUnchanged(t, s, key, raw)
}

func TestStore_GetNormalizesPartialLegacyRecord(t *testing.T) {
	s, done := newTestStore(t)
	defer done()

	key := "abc123deadbe"
	raw := []byte(`{"name":null,"prompt":null,"schedule":null,"paused":true}`)
	seedRawCronJob(t, s, key, raw)

	got, err := s.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != key || got.Name != key || got.Prompt != "" || got.Schedule != "?" || !got.Paused {
		t.Fatalf("normalized job = %+v, want id/name=%q prompt empty schedule ? paused", got, key)
	}
	assertRawCronJobUnchanged(t, s, key, raw)
}

func seedRawCronJob(t *testing.T, s *Store, key string, raw []byte) {
	t.Helper()
	if err := s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte(cronJobsBucket)).Put([]byte(key), raw)
	}); err != nil {
		t.Fatalf("seed raw cron job: %v", err)
	}
}

func assertRawCronJobUnchanged(t *testing.T, s *Store, key string, want []byte) {
	t.Helper()
	var got []byte
	if err := s.db.View(func(tx *bbolt.Tx) error {
		blob := tx.Bucket([]byte(cronJobsBucket)).Get([]byte(key))
		got = append([]byte(nil), blob...)
		return nil
	}); err != nil {
		t.Fatalf("read raw cron job: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("raw cron job changed\n got: %s\nwant: %s", got, want)
	}
}
