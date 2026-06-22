package postgresutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func UUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: value,
		Valid: true,
	}
}

func UUIDValue(value pgtype.UUID) uuid.UUID {
	return uuid.UUID(value.Bytes)
}

func Timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}

func OptionalTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return Timestamptz(*value)
}

func TimestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func TextPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func OptionalText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *value, Valid: true}
}
