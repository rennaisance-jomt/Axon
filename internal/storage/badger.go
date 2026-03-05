package storage

import (

	"encoding/binary"
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

func (d *DB) ListWithPrefix(prefix string) ([]string, error) {
	var keys []string
	prefixBytes := []byte(prefix)
	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			key := it.Item().Key()
			// Return key without "session:" prefix for consistency with SetSession/GetSession usage
			// SetSession adds "session:", so if we list "session:foo", we want to return "foo"
			keys = append(keys, string(key[len("session:"):]))
		}
		return nil
	})
	return keys, err
}

// Audit operations
func (d *DB) AppendAuditLog(data []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		// Get current audit index
		var index uint64
		item, err := txn.Get([]byte("audit:index"))
		if err == nil {
			val, _ := item.ValueCopy(nil)
			if len(val) == 8 {
				index = binary.BigEndian.Uint64(val)
			}
		}

		// Store new entry
		key := fmt.Sprintf("audit:log:%016d", index)
		if err := txn.Set([]byte(key), data); err != nil {
			return err
		}

		// Increment index
		newIndex := make([]byte, 8)
		binary.BigEndian.PutUint64(newIndex, index+1)
		return txn.Set([]byte("audit:index"), newIndex)
	})
}

func (d *DB) GetAuditLogs(limit, offset int) ([][]byte, error) {
	var logs [][]byte
	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("audit:log:")
		count := 0
		skipped := 0

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
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

// StoreElementMemory stores a single intent -> ref mapping
func (d *DB) StoreElementMemory(key, value string) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(value))
	})
}

// GetElementMemoryByKey retrieves a specific intent mapping
func (d *DB) GetElementMemoryByKey(key string) (string, error) {
	var value []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return "", err
	}
	return string(value), nil
}

// ListElementMemories lists all memories for a domain
func (d *DB) ListElementMemories(domain string) (map[string]string, error) {
	memories := make(map[string]string)
	prefix := []byte("intent:" + domain + ":")
	
	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := string(it.Item().Key())
			value, _ := it.Item().ValueCopy(nil)
			memories[key] = string(value)
		}
		return nil
	})
	
	return memories, err
}
