package cron

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"go.etcd.io/bbolt"
)

// ErrJobNotFound is returned by Get / Delete / Update when the target
// job ID isn't in the cron_jobs bucket.
var ErrJobNotFound = errors.New("cron: job not found")

// ErrJobNameTaken is returned by Create when another job already uses
// the requested Name. Names are unique; IDs are unique too but
// separately (IDs are random, names are operator-assigned).
var ErrJobNameTaken = errors.New("cron: job name already taken")

const cronJobsBucket = "cron_jobs"

// Store is the bbolt-backed Job persistence layer. The underlying
// *bbolt.DB is owned by the caller (typically the same *bbolt.DB the
// Phase 2.C session map uses, so a single file on disk).
type Store struct {
	db *bbolt.DB
}

// NewStore opens/creates the cron_jobs bucket and returns a ready-to-use
// Store. Safe to call multiple times.
func NewStore(db *bbolt.DB) (*Store, error) {
	if err := secureCronStatePath(db.Path()); err != nil {
		return nil, err
	}
	err := db.Update(func(tx *bbolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists([]byte(cronJobsBucket))
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("cron: init bucket: %w", err)
	}
	return &Store{db: db}, nil
}

func secureCronStatePath(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.Chmod(dir, 0o700); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cron: secure state dir: %w", err)
		}
	}
	if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cron: secure state file: %w", err)
	}
	return nil
}

// Create persists a new job. Fails with ErrJobNameTaken if Name is
// already used.
func (s *Store) Create(j Job) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		var dup bool
		_ = b.ForEach(func(k, v []byte) error {
			var other Job
			if err := json.Unmarshal(v, &other); err != nil {
				return nil
			}
			if other.Name == j.Name {
				dup = true
			}
			return nil
		})
		if dup {
			return ErrJobNameTaken
		}
		blob, err := json.Marshal(j)
		if err != nil {
			return err
		}
		return b.Put([]byte(j.ID), blob)
	})
}

// Get loads one job by ID.
func (s *Store) Get(id string) (Job, error) {
	var j Job
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		blob := b.Get([]byte(id))
		if blob == nil {
			return ErrJobNotFound
		}
		return json.Unmarshal(blob, &j)
	})
	return j, err
}

// List returns every job in the bucket. Corrupt rows are silently
// skipped so one bad blob doesn't block operation.
func (s *Store) List() ([]Job, error) {
	var out []Job
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		return b.ForEach(func(k, v []byte) error {
			var j Job
			if err := json.Unmarshal(v, &j); err != nil {
				return nil // skip corrupt
			}
			out = append(out, j)
			return nil
		})
	})
	return out, err
}

// Update overwrites an existing job by ID. Errors with ErrJobNotFound
// if the ID isn't present.
func (s *Store) Update(j Job) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		if b.Get([]byte(j.ID)) == nil {
			return ErrJobNotFound
		}
		blob, err := json.Marshal(j)
		if err != nil {
			return err
		}
		return b.Put([]byte(j.ID), blob)
	})
}

// Delete removes a job by ID. No-op on missing keys (bbolt convention).
func (s *Store) Delete(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		return b.Delete([]byte(id))
	})
}

type SkillRefRewriteReport struct {
	UpdatedJobs int `json:"updated_jobs"`
	Replaced    int `json:"replaced"`
	Removed     int `json:"removed"`
}

func (s *Store) SnapshotSkillRefs() (map[string][]string, error) {
	out := map[string][]string{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		return b.ForEach(func(k, v []byte) error {
			var j Job
			if err := json.Unmarshal(v, &j); err != nil {
				return nil
			}
			out[j.ID] = append([]string(nil), j.Skills...)
			return nil
		})
	})
	return out, err
}

func (s *Store) RewriteSkillRefs(replacements map[string]string, removals []string) (SkillRefRewriteReport, error) {
	removeSet := map[string]bool{}
	for _, name := range removals {
		if strings.TrimSpace(name) != "" {
			removeSet[strings.TrimSpace(name)] = true
		}
	}
	report := SkillRefRewriteReport{}
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		return b.ForEach(func(k, v []byte) error {
			var j Job
			if err := json.Unmarshal(v, &j); err != nil {
				return nil
			}
			next, replaced, removed := rewriteSkillList(j.Skills, replacements, removeSet)
			if reflect.DeepEqual(next, j.Skills) {
				return nil
			}
			j.Skills = next
			blob, err := json.Marshal(j)
			if err != nil {
				return err
			}
			if err := b.Put(k, blob); err != nil {
				return err
			}
			report.UpdatedJobs++
			report.Replaced += replaced
			report.Removed += removed
			return nil
		})
	})
	return report, err
}

func (s *Store) RestoreSkillRefs(snapshot map[string][]string) (int, error) {
	var restored int
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(cronJobsBucket))
		for id, skills := range snapshot {
			blob := b.Get([]byte(id))
			if blob == nil {
				continue
			}
			var j Job
			if err := json.Unmarshal(blob, &j); err != nil {
				continue
			}
			if reflect.DeepEqual(j.Skills, skills) {
				continue
			}
			j.Skills = append([]string(nil), skills...)
			next, err := json.Marshal(j)
			if err != nil {
				return err
			}
			if err := b.Put([]byte(id), next); err != nil {
				return err
			}
			restored++
		}
		return nil
	})
	return restored, err
}

func rewriteSkillList(skills []string, replacements map[string]string, removals map[string]bool) ([]string, int, int) {
	out := make([]string, 0, len(skills))
	seen := map[string]bool{}
	var replaced, removed int
	for _, skill := range skills {
		next := strings.TrimSpace(skill)
		if next == "" {
			continue
		}
		if removals[next] {
			removed++
			continue
		}
		if replacement := strings.TrimSpace(replacements[next]); replacement != "" {
			next = replacement
			replaced++
		}
		if seen[next] {
			continue
		}
		seen[next] = true
		out = append(out, next)
	}
	return out, replaced, removed
}
