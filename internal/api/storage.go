package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"netbird-coredns/pkg/dns"
)

// Storage manages persistent DNS records storage
type Storage struct {
	filePath string
	mu       sync.RWMutex
	records  map[string]map[string]*dns.Record // domain -> name -> record
}

// NewStorage creates a new storage instance
func NewStorage(filePath string) (*Storage, error) {
	s := &Storage{
		filePath: filePath,
		records:  make(map[string]map[string]*dns.Record),
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Load existing records
	if err := s.load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load records: %w", err)
		}
	}

	return s, nil
}

// GetRecord retrieves a specific record
func (s *Storage) GetRecord(domain, name string) (*dns.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	domainRecords, ok := s.records[domain]
	if !ok {
		return nil, fmt.Errorf("no records found for domain: %s", domain)
	}

	record, ok := domainRecords[name]
	if !ok {
		return nil, fmt.Errorf("record not found: %s.%s", name, domain)
	}

	return record, nil
}

// ListRecords returns all records
func (s *Storage) ListRecords() map[string]map[string]*dns.Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy to prevent external modification
	result := make(map[string]map[string]*dns.Record)
	for domain, records := range s.records {
		result[domain] = make(map[string]*dns.Record)
		for name, record := range records {
			recordCopy := *record
			result[domain][name] = &recordCopy
		}
	}

	return result
}

// ListRecordsByDomain returns all records for a specific domain
func (s *Storage) ListRecordsByDomain(domain string) map[string]*dns.Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	domainRecords, ok := s.records[domain]
	if !ok {
		return make(map[string]*dns.Record)
	}

	// Deep copy
	result := make(map[string]*dns.Record)
	for name, record := range domainRecords {
		recordCopy := *record
		result[name] = &recordCopy
	}

	return result
}

// SetRecord adds or updates a record
func (s *Storage) SetRecord(record *dns.Record) error {
	if err := record.Validate(); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure domain map exists
	if s.records[record.Domain] == nil {
		s.records[record.Domain] = make(map[string]*dns.Record)
	}

	// Set TTL default if not specified
	if record.TTL == 0 {
		record.TTL = 60
	}

	s.records[record.Domain][record.Name] = record

	// Persist to disk
	return s.save()
}

// DeleteRecord removes a record
func (s *Storage) DeleteRecord(domain, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	domainRecords, ok := s.records[domain]
	if !ok {
		return fmt.Errorf("no records found for domain: %s", domain)
	}

	if _, ok := domainRecords[name]; !ok {
		return fmt.Errorf("record not found: %s.%s", name, domain)
	}

	delete(domainRecords, name)

	// Clean up empty domain maps
	if len(domainRecords) == 0 {
		delete(s.records, domain)
	}

	// Persist to disk
	return s.save()
}

// load reads records from the file with shared locking
func (s *Storage) load() error {
	file, err := os.Open(s.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Acquire shared lock for reading
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); err != nil {
		return fmt.Errorf("failed to acquire shared lock: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	// Decode JSON
	if err := json.NewDecoder(file).Decode(&s.records); err != nil {
		return fmt.Errorf("failed to decode records: %w", err)
	}

	return nil
}

// save writes records to the file with exclusive locking
func (s *Storage) save() error {
	// Create temp file for atomic write
	tempFile := s.filePath + ".tmp"
	
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile) // Clean up on error

	// Acquire exclusive lock for writing
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return fmt.Errorf("failed to acquire exclusive lock: %w", err)
	}

	// Encode JSON with pretty printing
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.records); err != nil {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
		return fmt.Errorf("failed to encode records: %w", err)
	}

	// Release lock and close file
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	file.Close()

	// Atomic rename
	if err := os.Rename(tempFile, s.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Reload reloads records from disk
func (s *Storage) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.load()
}

