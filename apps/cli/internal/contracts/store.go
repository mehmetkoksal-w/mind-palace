package contracts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Store provides persistence for contracts.
type Store struct {
	db *sql.DB
}

// NewStore creates a new contract store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateTables creates the necessary database tables for contracts.
func (s *Store) CreateTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS contracts (
			id TEXT PRIMARY KEY,
			method TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			endpoint_pattern TEXT,

			backend_file TEXT,
			backend_line INTEGER,
			backend_framework TEXT,
			backend_handler TEXT,
			backend_request_schema TEXT,
			backend_response_schema TEXT,

			frontend_call_count INTEGER DEFAULT 0,

			status TEXT DEFAULT 'discovered',
			authority TEXT DEFAULT 'proposed',
			confidence REAL DEFAULT 0.0,

			first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS contract_frontend_calls (
			id TEXT PRIMARY KEY,
			contract_id TEXT REFERENCES contracts(id) ON DELETE CASCADE,
			file_path TEXT NOT NULL,
			line_number INTEGER NOT NULL,
			call_type TEXT,
			expected_schema TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS contract_mismatches (
			id TEXT PRIMARY KEY,
			contract_id TEXT REFERENCES contracts(id) ON DELETE CASCADE,
			field_path TEXT NOT NULL,
			mismatch_type TEXT NOT NULL,
			severity TEXT DEFAULT 'warning',
			description TEXT,
			backend_type TEXT,
			frontend_type TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_contracts_endpoint ON contracts(endpoint)`,
		`CREATE INDEX IF NOT EXISTS idx_contracts_method ON contracts(method)`,
		`CREATE INDEX IF NOT EXISTS idx_contracts_status ON contracts(status)`,
		`CREATE INDEX IF NOT EXISTS idx_contract_calls_contract ON contract_frontend_calls(contract_id)`,
		`CREATE INDEX IF NOT EXISTS idx_contract_mismatches_contract ON contract_mismatches(contract_id)`,
		`CREATE INDEX IF NOT EXISTS idx_contract_mismatches_type ON contract_mismatches(mismatch_type)`,
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(context.Background(), q); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// SaveContract saves or updates a contract.
func (s *Store) SaveContract(contract *Contract) error {
	requestSchema, _ := json.Marshal(contract.Backend.RequestSchema)
	responseSchema, _ := json.Marshal(contract.Backend.ResponseSchema)

	_, err := s.db.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO contracts (
			id, method, endpoint, endpoint_pattern,
			backend_file, backend_line, backend_framework, backend_handler,
			backend_request_schema, backend_response_schema,
			frontend_call_count, status, authority, confidence,
			first_seen, last_seen, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		contract.ID, contract.Method, contract.Endpoint, contract.EndpointPattern,
		contract.Backend.File, contract.Backend.Line, contract.Backend.Framework, contract.Backend.Handler,
		string(requestSchema), string(responseSchema),
		len(contract.FrontendCalls), string(contract.Status), contract.Authority, contract.Confidence,
		contract.FirstSeen, contract.LastSeen, time.Now(), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to save contract: %w", err)
	}

	// Save frontend calls
	for _, call := range contract.FrontendCalls {
		if err := s.saveFrontendCall(contract.ID, &call); err != nil {
			return err
		}
	}

	// Save mismatches
	for _, mismatch := range contract.Mismatches {
		if err := s.saveMismatch(contract.ID, &mismatch); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) saveFrontendCall(contractID string, call *FrontendCall) error {
	expectedSchema, _ := json.Marshal(call.ExpectedSchema)

	_, err := s.db.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO contract_frontend_calls (
			id, contract_id, file_path, line_number, call_type, expected_schema
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		call.ID, contractID, call.File, call.Line, call.CallType, string(expectedSchema),
	)
	return err
}

func (s *Store) saveMismatch(contractID string, mismatch *FieldMismatch) error {
	id := GenerateID("mm")

	_, err := s.db.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO contract_mismatches (
			id, contract_id, field_path, mismatch_type, severity, description, backend_type, frontend_type
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id, contractID, mismatch.FieldPath, string(mismatch.Type), mismatch.Severity,
		mismatch.Description, mismatch.BackendType, mismatch.FrontendType,
	)
	return err
}

// GetContract retrieves a contract by ID.
func (s *Store) GetContract(id string) (*Contract, error) {
	row := s.db.QueryRowContext(context.Background(), `
		SELECT id, method, endpoint, endpoint_pattern,
			backend_file, backend_line, backend_framework, backend_handler,
			backend_request_schema, backend_response_schema,
			status, authority, confidence, first_seen, last_seen
		FROM contracts WHERE id = ?
	`, id)

	contract := &Contract{}
	var requestSchema, responseSchema string
	var status string

	err := row.Scan(
		&contract.ID, &contract.Method, &contract.Endpoint, &contract.EndpointPattern,
		&contract.Backend.File, &contract.Backend.Line, &contract.Backend.Framework, &contract.Backend.Handler,
		&requestSchema, &responseSchema,
		&status, &contract.Authority, &contract.Confidence, &contract.FirstSeen, &contract.LastSeen,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	contract.Status = ContractStatus(status)

	// Unmarshal schemas
	if requestSchema != "" {
		_ = json.Unmarshal([]byte(requestSchema), &contract.Backend.RequestSchema)
	}
	if responseSchema != "" {
		_ = json.Unmarshal([]byte(responseSchema), &contract.Backend.ResponseSchema)
	}

	// Load frontend calls
	calls, err := s.getFrontendCalls(id)
	if err != nil {
		return nil, err
	}
	contract.FrontendCalls = calls

	// Load mismatches
	mismatches, err := s.getMismatches(id)
	if err != nil {
		return nil, err
	}
	contract.Mismatches = mismatches

	return contract, nil
}

func (s *Store) getFrontendCalls(contractID string) ([]FrontendCall, error) {
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT id, file_path, line_number, call_type, expected_schema
		FROM contract_frontend_calls WHERE contract_id = ?
	`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []FrontendCall
	for rows.Next() {
		var call FrontendCall
		var expectedSchema string
		if err := rows.Scan(&call.ID, &call.File, &call.Line, &call.CallType, &expectedSchema); err != nil {
			return nil, err
		}
		if expectedSchema != "" {
			_ = json.Unmarshal([]byte(expectedSchema), &call.ExpectedSchema)
		}
		calls = append(calls, call)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return calls, nil
}

func (s *Store) getMismatches(contractID string) ([]FieldMismatch, error) {
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT field_path, mismatch_type, severity, description, backend_type, frontend_type
		FROM contract_mismatches WHERE contract_id = ?
	`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mismatches []FieldMismatch
	for rows.Next() {
		var m FieldMismatch
		var mType string
		if err := rows.Scan(&m.FieldPath, &mType, &m.Severity, &m.Description, &m.BackendType, &m.FrontendType); err != nil {
			return nil, err
		}
		m.Type = MismatchType(mType)
		mismatches = append(mismatches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mismatches, nil
}

// ListContracts returns all contracts with optional filtering.
func (s *Store) ListContracts(filter ContractFilter) ([]*Contract, error) {
	query := `SELECT id FROM contracts WHERE 1=1`
	args := []interface{}{}

	if filter.Method != "" {
		query += " AND method = ?"
		args = append(args, filter.Method)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.Endpoint != "" {
		query += " AND endpoint LIKE ?"
		args = append(args, "%"+filter.Endpoint+"%")
	}
	if filter.HasMismatches {
		query += " AND id IN (SELECT DISTINCT contract_id FROM contract_mismatches)"
	}

	query += " ORDER BY last_seen DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*Contract
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		contract, err := s.GetContract(id)
		if err != nil {
			return nil, err
		}
		if contract != nil {
			contracts = append(contracts, contract)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return contracts, nil
}

// ContractFilter defines filtering options for listing contracts.
type ContractFilter struct {
	Method        string
	Status        ContractStatus
	Endpoint      string
	HasMismatches bool
	Limit         int
}

// UpdateStatus updates the status of a contract.
func (s *Store) UpdateStatus(id string, status ContractStatus) error {
	_, err := s.db.ExecContext(context.Background(), `
		UPDATE contracts SET status = ?, updated_at = ? WHERE id = ?
	`, string(status), time.Now(), id)
	return err
}

// DeleteContract deletes a contract and its related data.
func (s *Store) DeleteContract(id string) error {
	// Cascading deletes should handle related data
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM contracts WHERE id = ?`, id)
	return err
}

// GetStats returns contract statistics.
func (s *Store) GetStats() (*ContractStats, error) {
	stats := &ContractStats{
		ByMethod: make(map[string]int),
	}

	// Total contracts
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contracts`).Scan(&stats.Total)

	// Contracts by status
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contracts WHERE status = 'discovered'`).Scan(&stats.Discovered)
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contracts WHERE status = 'verified'`).Scan(&stats.Verified)
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contracts WHERE status = 'mismatch'`).Scan(&stats.Mismatch)
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contracts WHERE status = 'ignored'`).Scan(&stats.Ignored)

	// Contracts by method
	rows, err := s.db.QueryContext(context.Background(), `SELECT method, COUNT(*) FROM contracts GROUP BY method`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var method string
		var count int
		_ = rows.Scan(&method, &count)
		stats.ByMethod[method] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Total mismatches (errors and warnings)
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contract_mismatches WHERE severity = 'error'`).Scan(&stats.TotalErrors)
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contract_mismatches WHERE severity = 'warning'`).Scan(&stats.TotalWarnings)

	// Total frontend calls
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM contract_frontend_calls`).Scan(&stats.TotalCalls)

	return stats, nil
}

// ClearMismatches removes all mismatches for a contract.
func (s *Store) ClearMismatches(contractID string) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM contract_mismatches WHERE contract_id = ?`, contractID)
	return err
}

// BulkSave saves multiple contracts efficiently.
func (s *Store) BulkSave(contracts []*Contract) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, contract := range contracts {
		if err := s.SaveContract(contract); err != nil {
			return err
		}
	}

	return tx.Commit()
}
