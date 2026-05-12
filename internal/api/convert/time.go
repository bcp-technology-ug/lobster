// Package convert provides helpers to map between sqlc-generated DB models and
// protobuf messages. No business logic lives here.
package convert

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RFC3339 is the canonical timestamp format stored in SQLite.
const RFC3339 = time.RFC3339Nano

// TimestampFromDB parses a nullable RFC-3339 string from SQLite.
// Returns nil when the input pointer is nil or the string is empty.
func TimestampFromDB(s *string) *timestamppb.Timestamp {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(RFC3339, *s)
	if err != nil {
		// Fall back to second precision
		t, err = time.Parse(time.RFC3339, *s)
		if err != nil {
			return nil
		}
	}
	return timestamppb.New(t)
}

// TimestampFromDBStr parses a non-nullable RFC-3339 string from SQLite.
func TimestampFromDBStr(s string) *timestamppb.Timestamp {
	if s == "" {
		return nil
	}
	return TimestampFromDB(&s)
}

// TimestampToDB formats a proto timestamp to RFC-3339 nano string.
func TimestampToDB(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(RFC3339)
}

// NowDB returns the current UTC time in DB string format.
func NowDB() string {
	return time.Now().UTC().Format(RFC3339)
}

// DurationFromNanos converts nanoseconds stored in SQLite to a protobuf Duration.
func DurationFromNanos(ns int64) *durationpb.Duration {
	return durationpb.New(time.Duration(ns))
}

// NanosFromDuration converts a proto Duration to nanoseconds for SQLite.
func NanosFromDuration(d *durationpb.Duration) int64 {
	if d == nil {
		return 0
	}
	return d.AsDuration().Nanoseconds()
}
