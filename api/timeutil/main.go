package timeutil

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ParsePostgresTimestamp parses a timestamp string from PostgreSQL
func ParsePostgresTimestamp(ts time.Time) *timestamppb.Timestamp {
	return timestamppb.New(ts)
}

// ToPostgresTimestamp converts a protobuf timestamp to a PostgreSQL-compatible string
func ToPostgresTimestamp(ts *timestamppb.Timestamp) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: ts.AsTime(), Valid: true}
}
