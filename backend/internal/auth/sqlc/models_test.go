package sqlc

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUsersMigrationHasPhase1Columns(t *testing.T) {
	t.Parallel()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "../../../db/migrations/00002_users.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	needles := []string{
		"CREATE TABLE users",
		"id BIGINT PRIMARY KEY",
		"email TEXT NOT NULL",
		"email_verified_at TIMESTAMPTZ",
		"password_hash TEXT",
		"display_name TEXT NOT NULL",
		"date_of_birth DATE NOT NULL",
		"terms_accepted_at TIMESTAMPTZ NOT NULL",
		"created_at TIMESTAMPTZ NOT NULL",
		"updated_at TIMESTAMPTZ NOT NULL",
		"deleted_at TIMESTAMPTZ",
		"users_email_live_key",
		"WHERE deleted_at IS NULL",
		"DROP TABLE IF EXISTS users",
	}
	for _, needle := range needles {
		if !strings.Contains(sql, needle) {
			t.Errorf("migration missing %q", needle)
		}
	}
	if strings.Contains(strings.ToLower(sql), "citext") {
		t.Error("email is TEXT stored lowercased; do not add citext in this phase")
	}
	if strings.Contains(sql, "CREATE TABLE oauth_accounts") || strings.Contains(sql, "CREATE TABLE auth_tokens") {
		t.Error("oauth_accounts and auth_tokens wait for later phases")
	}
}
