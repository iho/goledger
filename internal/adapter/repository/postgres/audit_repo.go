package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iho/goledger/internal/domain"
)

// AuditRepository implements audit log persistence
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Create inserts a new audit log entry
func (r *AuditRepository) Create(log *domain.AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	var beforeStateJSON, afterStateJSON []byte
	var err error

	if log.BeforeState != nil {
		beforeStateJSON, err = json.Marshal(log.BeforeState)
		if err != nil {
			return err
		}
	}

	if log.AfterState != nil {
		afterStateJSON, err = json.Marshal(log.AfterState)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT INTO audit_logs (
			id, user_id, action, resource_type, resource_id,
			ip_address, user_agent, request_id,
			before_state, after_state, status, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.pool.Exec(context.Background(), query,
		log.ID,
		log.UserID,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.IPAddress,
		log.UserAgent,
		log.RequestID,
		beforeStateJSON,
		afterStateJSON,
		log.Status,
		log.ErrorMessage,
		log.CreatedAt,
	)

	return err
}

// List retrieves audit logs with filtering
func (r *AuditRepository) List(filter *domain.AuditFilter) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, user_id, action, resource_type, resource_id,
		       ip_address, user_agent, request_id,
		       before_state, after_state, status, error_message, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []any{}
	argPos := 1

	if filter.UserID != "" {
		query += ` AND user_id = $` + string(rune(argPos))
		args = append(args, filter.UserID)
		argPos++
	}

	if filter.Action != "" {
		query += ` AND action = $` + string(rune(argPos))
		args = append(args, filter.Action)
		argPos++
	}

	if filter.ResourceType != "" {
		query += ` AND resource_type = $` + string(rune(argPos))
		args = append(args, filter.ResourceType)
		argPos++
	}

	query += ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT $` + string(rune(argPos))
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += ` OFFSET $` + string(rune(argPos))
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		var beforeStateJSON, afterStateJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&log.IPAddress,
			&log.UserAgent,
			&log.RequestID,
			&beforeStateJSON,
			&afterStateJSON,
			&log.Status,
			&log.ErrorMessage,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if beforeStateJSON != nil {
			_ = json.Unmarshal(beforeStateJSON, &log.BeforeState)
		}

		if afterStateJSON != nil {
			_ = json.Unmarshal(afterStateJSON, &log.AfterState)
		}

		logs = append(logs, &log)
	}

	return logs, rows.Err()
}

// GetByResourceID retrieves all audit logs for a specific resource
func (r *AuditRepository) GetByResourceID(resourceType, resourceID string) ([]*domain.AuditLog, error) {
	return r.List(&domain.AuditFilter{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}
