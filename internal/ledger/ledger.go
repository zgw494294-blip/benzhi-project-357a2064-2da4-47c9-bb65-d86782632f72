package ledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"meshturn/internal/batch"
)

const CurrentVersion = 1

type Ledger struct {
	Version int           `json:"version"`
	Batches []batch.Batch `json:"batches"`
}

func New() Ledger {
	return Ledger{Version: CurrentVersion, Batches: make([]batch.Batch, 0)}
}

func Load(path string) (Ledger, error) {
	if strings.TrimSpace(path) == "" {
		return Ledger{}, fmt.Errorf("load ledger: path is required")
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return Ledger{}, fmt.Errorf("load ledger %q: %w", path, err)
	}

	var result Ledger
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return Ledger{}, fmt.Errorf("decode ledger %q: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Ledger{}, fmt.Errorf("decode ledger %q: multiple JSON values", path)
		}
		return Ledger{}, fmt.Errorf("decode ledger %q: trailing data: %w", path, err)
	}
	if err := result.Validate(); err != nil {
		return Ledger{}, fmt.Errorf("validate ledger %q: %w", path, err)
	}
	return result, nil
}

func (l Ledger) Validate() error {
	if l.Version != CurrentVersion {
		return fmt.Errorf("unsupported ledger version %d", l.Version)
	}
	seen := make(map[string]struct{}, len(l.Batches))
	for i, b := range l.Batches {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("batch %d: %w", i, err)
		}
		if _, exists := seen[b.ID]; exists {
			return fmt.Errorf("duplicate batch id %q", b.ID)
		}
		seen[b.ID] = struct{}{}
	}
	return nil
}

func (l *Ledger) Add(newBatch batch.Batch) error {
	if err := newBatch.Validate(); err != nil {
		return err
	}
	if _, found := l.Find(newBatch.ID); found {
		return fmt.Errorf("batch %q already exists", newBatch.ID)
	}
	l.Batches = append(l.Batches, newBatch)
	return nil
}

func (l *Ledger) Find(id string) (*batch.Batch, bool) {
	for i := range l.Batches {
		if l.Batches[i].ID == id {
			return &l.Batches[i], true
		}
	}
	return nil, false
}

func (l Ledger) Save(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("save ledger: path is required")
	}
	if err := l.Validate(); err != nil {
		return fmt.Errorf("save ledger: %w", err)
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ledger: %w", err)
	}
	data = append(data, '\n')

	temporaryPath := TemporaryPath(path)
	temporary, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create ledger temporary file %q: %w", temporaryPath, err)
	}
	closed := false
	closeTemporary := func() error {
		if closed {
			return nil
		}
		closed = true
		return temporary.Close()
	}
	failed := func(cause error) error {
		closeErr := closeTemporary()
		removeErr := os.Remove(temporaryPath)
		if errors.Is(removeErr, os.ErrNotExist) {
			removeErr = nil
		}
		return errors.Join(cause, closeErr, removeErr)
	}

	if _, err := temporary.Write(data); err != nil {
		return failed(fmt.Errorf("write ledger temporary file %q: %w", temporaryPath, err))
	}
	if err := temporary.Sync(); err != nil {
		return failed(fmt.Errorf("sync ledger temporary file %q: %w", temporaryPath, err))
	}
	if err := closeTemporary(); err != nil {
		return failed(fmt.Errorf("close ledger temporary file %q: %w", temporaryPath, err))
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return failed(fmt.Errorf("replace ledger %q: %w", path, err))
	}
	return nil
}

func TemporaryPath(path string) string {
	return filepath.Clean(path) + ".tmp"
}
