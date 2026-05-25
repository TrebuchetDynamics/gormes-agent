package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var defaultFileMutationQueue = NewFileMutationQueue()

// FileMutationQueue serializes file mutation windows by canonical target path.
// Existing files, including symlinks, key by their real path; missing/new files
// key by the cleaned absolute path that will be created.
type FileMutationQueue struct {
	mu    sync.Mutex
	locks map[string]*fileMutationLock
}

type fileMutationLock struct {
	ch chan struct{}
}

func NewFileMutationQueue() *FileMutationQueue {
	return &FileMutationQueue{locks: make(map[string]*fileMutationLock)}
}

func (q *FileMutationQueue) Key(path string) (string, error) {
	return fileMutationQueueKey(path)
}

func (q *FileMutationQueue) Run(ctx context.Context, path string, fn func(context.Context) error) error {
	return q.RunMany(ctx, []string{path}, fn)
}

func (q *FileMutationQueue) RunMany(ctx context.Context, paths []string, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if fn == nil {
		return nil
	}
	if q == nil {
		return fn(ctx)
	}
	keys, err := q.keys(paths)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return fn(ctx)
	}
	locks := make([]*fileMutationLock, 0, len(keys))
	for _, key := range keys {
		lock := q.lockForKey(key)
		if err := lock.acquire(ctx); err != nil {
			for i := len(locks) - 1; i >= 0; i-- {
				locks[i].release()
			}
			return err
		}
		locks = append(locks, lock)
	}
	defer func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i].release()
		}
	}()
	return fn(ctx)
}

func (q *FileMutationQueue) keys(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(paths))
	for _, path := range paths {
		key, err := fileMutationQueueKey(path)
		if err != nil {
			return nil, err
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func (q *FileMutationQueue) lockForKey(key string) *fileMutationLock {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.locks == nil {
		q.locks = make(map[string]*fileMutationLock)
	}
	lock := q.locks[key]
	if lock == nil {
		lock = &fileMutationLock{ch: make(chan struct{}, 1)}
		lock.ch <- struct{}{}
		q.locks[key] = lock
	}
	return lock
}

func (l *fileMutationLock) acquire(ctx context.Context) error {
	if l == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.ch:
		return nil
	}
}

func (l *fileMutationLock) release() {
	if l == nil {
		return
	}
	select {
	case l.ch <- struct{}{}:
	default:
	}
}

func fileMutationQueueKey(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("file mutation path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if realPath, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(realPath), nil
	}
	info, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return abs, nil
	}
	linkTarget, err := os.Readlink(abs)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(linkTarget) {
		linkTarget = filepath.Join(filepath.Dir(abs), linkTarget)
	}
	return filepath.Clean(linkTarget), nil
}

func fileTaskMutationQueue(cfg FileTaskToolConfig) *FileMutationQueue {
	if cfg.MutationQueue != nil {
		return cfg.MutationQueue
	}
	return defaultFileMutationQueue
}
