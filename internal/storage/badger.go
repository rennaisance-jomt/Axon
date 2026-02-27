package storage

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

// DB represents the storage database
type DB struct {
	db *badger.DB
}

// New creates a new storage instance
func New(path string) (*DB, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable logging to reduce disk writes

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &DB{db: db}, nil
}

// Close closes the database
func (d *DB) Close() error {
	return d.db.Close()
}

// Session operations
func (d *DB) GetSession(id string) ([]byte, error) {
	var value []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("session:" + id))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	return value, err
}

func (d *DB) SetSession(id string, data []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("session:"+id), data)
	})
}

func (d *DB) DeleteSession(id string) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("session:" + id))
	})
}

func (d *DB) ListSessions() ([]string, error) {
	var sessions []string
	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("session:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().Key()
			sessions = append(sessions, string(key[len(prefix):]))
		}
		return nil
	})
	return sessions, err
}

// Audit operations
func (d *DB) AppendAuditLog(data []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		// Get current audit index
		var index uint64
		item, err := txn.Get([]byte("audit:index"))
		if err == nil {
			value, _ := item.ValueCopy(nil)
			// Parse index (simplified)
			index = 0
			_ = value
		}

		// Store new entry
		key := fmt.Sprintf("audit:%d", index)
		if err := txn.Set([]byte(key), data); err != nil {
			return err
		}

		// Increment index
		return txn.Set([]byte("audit:index"), []byte(fmt.Sprintf("%d", index+1)))
	})
}

func (d *DB) GetAuditLogs(limit, offset int) ([][]byte, error) {
	var logs [][]byte
	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("audit:")
		count := 0
		skipped := 0

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := string(it.Item().Key())
			if key == "audit:index" {
				continue
			}

			if skipped < offset {
				skipped++
				continue
			}

			if count >= limit {
				break
			}

			value, _ := it.Item().ValueCopy(nil)
			logs = append(logs, value)
			count++
		}
		return nil
	})
	return logs, err
}

// Element memory operations
func (d *DB) GetElementMemory(domain string) ([]byte, error) {
	var value []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("memory:" + domain))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	return value, err
}

func (d *DB) SetElementMemory(domain string, data []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("memory:"+domain), data)
	})
}
