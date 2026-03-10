package store

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapDBError_UniqueViolation(t *testing.T) {
	err := mapDBError(&pgconn.PgError{Code: "23505", ConstraintName: "uq_tenants_name", Message: "duplicate key"})
	var dbErr *DBError
	if !errors.As(err, &dbErr) {
		t.Fatalf("expected DBError, got %T", err)
	}
	if dbErr.Kind != ErrorKindConflict {
		t.Fatalf("expected %s, got %s", ErrorKindConflict, dbErr.Kind)
	}
}

func TestMapDBError_ForeignKeyViolation(t *testing.T) {
	err := mapDBError(&pgconn.PgError{Code: "23503", ConstraintName: "users_tenant_id_fkey", Message: "fk failed"})
	var dbErr *DBError
	if !errors.As(err, &dbErr) {
		t.Fatalf("expected DBError, got %T", err)
	}
	if dbErr.Kind != ErrorKindForeignKey {
		t.Fatalf("expected %s, got %s", ErrorKindForeignKey, dbErr.Kind)
	}
}

func TestMapDBError_Validation(t *testing.T) {
	err := mapDBError(&pgconn.PgError{Code: "22P02", Message: "invalid input syntax for type uuid"})
	var dbErr *DBError
	if !errors.As(err, &dbErr) {
		t.Fatalf("expected DBError, got %T", err)
	}
	if dbErr.Kind != ErrorKindValidation {
		t.Fatalf("expected %s, got %s", ErrorKindValidation, dbErr.Kind)
	}
	if dbErr.Message == "" {
		t.Fatal("expected validation message")
	}
}

func TestMapDBError_CheckViolation(t *testing.T) {
	err := mapDBError(&pgconn.PgError{Code: "23514", ConstraintName: "chk_tenant_policies_device_limit_positive"})
	var dbErr *DBError
	if !errors.As(err, &dbErr) {
		t.Fatalf("expected DBError, got %T", err)
	}
	if dbErr.Kind != ErrorKindValidation {
		t.Fatalf("expected validation kind, got %s", dbErr.Kind)
	}
	if dbErr.Message != "device_limit must be > 0" {
		t.Fatalf("unexpected validation message: %s", dbErr.Message)
	}
}
