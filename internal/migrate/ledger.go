package migrate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LedgerEntry records one completed migration.
type LedgerEntry struct {
	Source    string    `json:"source"`  // original path in old repo
	BookID    string    `json:"book_id"` // new catalog ID
	Shelf     string    `json:"shelf"`   // target shelf name
	Timestamp time.Time `json:"timestamp"`
}

// Ledger is a JSONL append-only migration log.
type Ledger struct {
	path string
}

// DefaultLedgerPath returns the default path for the migration ledger.
func DefaultLedgerPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "shelfctl", "migrated.jsonl")
}

// OpenLedger opens (or creates) the ledger at path.
func OpenLedger(path string) (*Ledger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, err
	}
	return &Ledger{path: path}, nil
}

// Append adds an entry to the ledger.
func (l *Ledger) Append(e LedgerEntry) error {
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	e.Timestamp = time.Now().UTC()
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(f, string(data))
	return err
}

// Contains reports whether the given source path has already been migrated.
func (l *Ledger) Contains(source string) (bool, error) {
	f, err := os.Open(l.path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e LedgerEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		if e.Source == source {
			return true, nil
		}
	}
	return false, sc.Err()
}

// Entries returns all ledger entries.
func (l *Ledger) Entries() ([]LedgerEntry, error) {
	f, err := os.Open(l.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []LedgerEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e LedgerEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, sc.Err()
}
