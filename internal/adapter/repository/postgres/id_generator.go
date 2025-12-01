
package postgres

import (
	"github.com/oklog/ulid/v2"
)

// ULIDGenerator generates ULID-based IDs.
type ULIDGenerator struct{}

// NewULIDGenerator creates a new ULIDGenerator.
func NewULIDGenerator() *ULIDGenerator {
	return &ULIDGenerator{}
}

// Generate generates a new ULID.
func (g *ULIDGenerator) Generate() string {
	return ulid.Make().String()
}
