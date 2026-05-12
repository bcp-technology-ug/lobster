package convert

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTimestampFromDB_nil(t *testing.T) {
	t.Parallel()
	if TimestampFromDB(nil) != nil {
		t.Error("nil input should return nil")
	}
}

func TestTimestampFromDB_empty(t *testing.T) {
	t.Parallel()
	s := ""
	if TimestampFromDB(&s) != nil {
		t.Error("empty string should return nil")
	}
}

func TestTimestampFromDB_valid(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	s := now.Format(time.RFC3339)
	ts := TimestampFromDB(&s)
	if ts == nil {
		t.Fatal("expected non-nil timestamp")
	}
	got := ts.AsTime().UTC().Truncate(time.Second)
	if !got.Equal(now) {
		t.Errorf("got %v want %v", got, now)
	}
}

func TestTimestampFromDB_nanoseconds(t *testing.T) {
	t.Parallel()
	ref := time.Date(2024, 1, 2, 15, 4, 5, 123456789, time.UTC)
	s := ref.Format(RFC3339)
	ts := TimestampFromDB(&s)
	if ts == nil {
		t.Fatal("expected non-nil timestamp")
	}
	if !ts.AsTime().Equal(ref) {
		t.Errorf("got %v want %v", ts.AsTime(), ref)
	}
}

func TestTimestampFromDBStr_empty(t *testing.T) {
	t.Parallel()
	if TimestampFromDBStr("") != nil {
		t.Error("empty string should return nil")
	}
}

func TestTimestampFromDBStr_valid(t *testing.T) {
	t.Parallel()
	ref := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s := ref.Format(RFC3339)
	ts := TimestampFromDBStr(s)
	if ts == nil {
		t.Fatal("expected non-nil")
	}
	if !ts.AsTime().UTC().Equal(ref) {
		t.Errorf("got %v want %v", ts.AsTime().UTC(), ref)
	}
}

func TestTimestampToDB_nil(t *testing.T) {
	t.Parallel()
	if s := TimestampToDB(nil); s != "" {
		t.Errorf("nil input should return empty string, got %q", s)
	}
}

func TestTimestampToDB_roundTrip(t *testing.T) {
	t.Parallel()
	ref := time.Date(2024, 3, 15, 10, 30, 0, 500000000, time.UTC)
	ts := timestamppb.New(ref)
	s := TimestampToDB(ts)
	if s == "" {
		t.Fatal("expected non-empty string")
	}
	back := TimestampFromDBStr(s)
	if back == nil {
		t.Fatal("expected non-nil timestamp back")
	}
	diff := back.AsTime().UTC().Sub(ref).Abs()
	if diff > time.Microsecond {
		t.Errorf("round-trip diff too large: %v", diff)
	}
}

func TestNowDB_format(t *testing.T) {
	t.Parallel()
	s := NowDB()
	if s == "" {
		t.Fatal("NowDB() returned empty string")
	}
	if _, err := time.Parse(RFC3339, s); err != nil {
		t.Errorf("NowDB() %q is not RFC3339Nano: %v", s, err)
	}
}

func TestDurationFromNanos_zero(t *testing.T) {
	t.Parallel()
	d := DurationFromNanos(0)
	if d == nil {
		t.Fatal("expected non-nil duration")
	}
	if d.AsDuration() != 0 {
		t.Errorf("expected 0, got %v", d.AsDuration())
	}
}

func TestDurationFromNanos_roundTrip(t *testing.T) {
	t.Parallel()
	ns := int64(3_500_000_000) // 3.5s
	d := DurationFromNanos(ns)
	if d.AsDuration().Nanoseconds() != ns {
		t.Errorf("got %d want %d", d.AsDuration().Nanoseconds(), ns)
	}
}

func TestNanosFromDuration_nil(t *testing.T) {
	t.Parallel()
	if got := NanosFromDuration(nil); got != 0 {
		t.Errorf("nil should return 0, got %d", got)
	}
}

func TestNanosFromDuration_roundTrip(t *testing.T) {
	t.Parallel()
	ns := int64(7_000_000) // 7ms
	got := NanosFromDuration(durationpb.New(time.Duration(ns)))
	if got != ns {
		t.Errorf("got %d want %d", got, ns)
	}
}
