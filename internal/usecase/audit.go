package usecase

import (
	"context"

	"github.com/iho/goledger/internal/domain"
)

// auditActor resolves the acting user ID and request attribution metadata
// from context for stamping audit log rows. Falls back to "system" when no
// authenticated user is present in context (e.g. auth disabled, background
// jobs, internal calls).
func auditActor(ctx context.Context) (userID, requestID, ipAddress, userAgent string) {
	userID = "system"
	if user, ok := domain.UserFromContext(ctx); ok {
		userID = user.ID
	}

	if meta, ok := domain.RequestMetaFromContext(ctx); ok {
		requestID = meta.RequestID
		ipAddress = meta.IPAddress
		userAgent = meta.UserAgent
	}

	return userID, requestID, ipAddress, userAgent
}
