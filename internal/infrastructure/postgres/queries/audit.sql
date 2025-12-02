-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
  id, user_id, action, resource_type, resource_id, 
  ip_address, user_agent, request_id, 
  before_state, after_state, status, error_message, created_at
) VALUES (
  $1, $2, $3, $4, $5, 
  $6, $7, $8, 
  $9, $10, $11, $12, $13
);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAuditLogsByResource :many
SELECT * FROM audit_logs
WHERE resource_type = $1 AND resource_id = $2
ORDER BY created_at DESC;

-- name: GetAuditLogsByUser :many
SELECT * FROM audit_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
