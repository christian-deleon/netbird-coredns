package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"netbird-coredns/internal/logger"
	"netbird-coredns/pkg/dns"
)

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// ListRecordsHandler handles GET /api/v1/records
func (s *Server) ListRecordsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	records := s.storage.ListRecords()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(records); err != nil {
		logger.Error("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// CreateRecordHandler handles POST /api/v1/records
func (s *Server) CreateRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var record dns.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.storage.SetRecord(&record); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create record: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Record created successfully",
		"record":  record,
	})
}

// UpdateRecordHandler handles PUT /api/v1/records/{domain}/{name}
func (s *Server) UpdateRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/v1/records/{domain}/{name}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/records/"), "/")
	if len(pathParts) != 2 {
		http.Error(w, "Invalid path format. Expected: /api/v1/records/{domain}/{name}", http.StatusBadRequest)
		return
	}

	domain := pathParts[0]
	name := pathParts[1]

	// Normalize "@" to empty string for root domain records
	if name == "@" {
		name = ""
	}

	var record dns.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Override domain and name from URL
	record.Domain = domain
	record.Name = name

	if err := s.storage.SetRecord(&record); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update record: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Record updated successfully",
		"record":  record,
	})
}

// DeleteRecordHandler handles DELETE /api/v1/records/{domain}/{name}
func (s *Server) DeleteRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/v1/records/{domain}/{name}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/records/"), "/")
	if len(pathParts) != 2 {
		http.Error(w, "Invalid path format. Expected: /api/v1/records/{domain}/{name}", http.StatusBadRequest)
		return
	}

	domain := pathParts[0]
	name := pathParts[1]

	// Normalize "@" to empty string for root domain records
	if name == "@" {
		name = ""
	}

	if err := s.storage.DeleteRecord(domain, name); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete record: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Record deleted successfully",
	})
}

// RecordHandler routes record requests based on path
func (s *Server) RecordHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Route based on path pattern
	if path == "/api/v1/records" || path == "/api/v1/records/" {
		switch r.Method {
		case http.MethodGet:
			s.ListRecordsHandler(w, r)
		case http.MethodPost:
			s.CreateRecordHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Pattern: /api/v1/records/{domain}/{name}
	if strings.HasPrefix(path, "/api/v1/records/") {
		switch r.Method {
		case http.MethodPut:
			s.UpdateRecordHandler(w, r)
		case http.MethodDelete:
			s.DeleteRecordHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}
