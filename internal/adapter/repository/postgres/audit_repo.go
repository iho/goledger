package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
	"github.com/iho/goledger/internal/usecase"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.create(ctx, r.pool, log)
}

// CreateTx inserts a new audit log entry within a transaction
func (r *AuditRepository) CreateTx(ctx context.Context, tx usecase.Transaction, log *domain.AuditLog) error {
	pgTx, ok := tx.(*Tx)
	if !ok {
		return fmt.Errorf("transaction is not *Tx")
	}
	return r.create(ctx, pgTx.PgxTx(), log)
}

func (r *AuditRepository) create(ctx context.Context, db generated.DBTX, log *domain.AuditLog) error {
	queries := generated.New(db)

	var beforeState, afterState []byte
	var err error

	if log.BeforeState != nil {
		beforeState, err = json.Marshal(log.BeforeState)
		if err != nil {
			return err
		}
	}

	if log.AfterState != nil {
		afterState, err = json.Marshal(log.AfterState)
		if err != nil {
			return err
		}
	}

	return queries.CreateAuditLog(ctx, generated.CreateAuditLogParams{
		ID:           log.ID,
		UserID:       log.UserID,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		IpAddress:    &log.IPAddress,
		UserAgent:    &log.UserAgent,
		RequestID:    &log.RequestID,
		BeforeState:  beforeState,
		AfterState:   afterState,
		Status:       log.Status,
		ErrorMessage: &log.ErrorMessage,
		CreatedAt: pgtype.Timestamptz{
			Time:  log.CreatedAt,
			Valid: true,
		},
	})
}

// List retrieves audit logs with filtering
func (r *AuditRepository) List(ctx context.Context, filter domain.AuditFilter) ([]*domain.AuditLog, error) {
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
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, filter.UserID)
		argPos++
	}

	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argPos)
		args = append(args, filter.Action)
		argPos++
	}

	if filter.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argPos)
		args = append(args, filter.ResourceType)
		argPos++
	}

	if filter.ResourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argPos)
		args = append(args, filter.ResourceID)
		argPos++
	}

	query += ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	} else {
		query += ` LIMIT 100`
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
		argPos++
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		var beforeStateJSON, afterStateJSON []byte
		var createdAt pgtype.Timestamptz

		var ipAddress, userAgent, requestID, errorMessage *string

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&ipAddress,
			&userAgent,
			&requestID,
			&beforeStateJSON,
			&afterStateJSON,
			&log.Status,
			&errorMessage,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		log.CreatedAt = createdAt.Time
		if ipAddress != nil {
			log.IPAddress = *ipAddress
		}
		if userAgent != nil {
			log.UserAgent = *userAgent
		}
		if requestID != nil {
			log.RequestID = *requestID
		}
		if errorMessage != nil {
			log.ErrorMessage = *errorMessage
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
func (r *AuditRepository) GetByResourceID(ctx context.Context, resourceType, resourceID string) ([]*domain.AuditLog, error) {
	// Use generated code for this specific query as it's optimized
	queries := generated.New(r.pool)
	logs, err := queries.GetAuditLogsByResource(ctx, generated.GetAuditLogsByResourceParams{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*domain.AuditLog, len(logs))
	for i, log := range logs {
		result[i] = &domain.AuditLog{
			ID:           log.ID,
			UserID:       log.UserID,
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   log.ResourceID,
			Status:       log.Status,
			CreatedAt:    log.CreatedAt.Time,
		}
		if log.IpAddress != nil {
			result[i].IPAddress = *log.IpAddress
		}
		if log.UserAgent != nil {
			result[i].UserAgent = *log.UserAgent
		}
		if log.RequestID != nil {
			result[i].RequestID = *log.RequestID
		}
		if log.ErrorMessage != nil {
			result[i].ErrorMessage = *log.ErrorMessage
		}

		if log.BeforeState != nil {
			_ = json.Unmarshal(log.BeforeState, &result[i].BeforeState)
		}
		if log.AfterState != nil {
			_ = json.Unmarshal(log.AfterState, &result[i].AfterState)
		}
	}
	return result, nil
}
