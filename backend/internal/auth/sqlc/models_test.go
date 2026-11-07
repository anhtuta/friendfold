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
		t.Error("oauth_accounts waits for phase 4; auth_tokens is migration 00003")
	}
}

func TestAuthTokensMigrationHasPhase2Columns(t *testing.T) {
	t.Parallel()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "../../../db/migrations/00003_auth_tokens.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	needles := []string{
		"CREATE TABLE auth_tokens",
		"id BIGINT PRIMARY KEY",
		"user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE",
		"purpose TEXT NOT NULL",
		"token_hash BYTEA NOT NULL",
		"expires_at TIMESTAMPTZ NOT NULL",
		"consumed_at TIMESTAMPTZ",
		"created_at TIMESTAMPTZ NOT NULL",
		"verify_email",
		"reset_password",
		"octet_length(token_hash) = 32",
		"auth_tokens_token_hash_key",
		"auth_tokens_unused_purpose_key",
		"WHERE consumed_at IS NULL",
		"DROP TABLE IF EXISTS auth_tokens",
	}
	for _, needle := range needles {
		if !strings.Contains(sql, needle) {
			t.Errorf("migration missing %q", needle)
		}
	}
	if strings.Contains(sql, "CREATE TABLE oauth_accounts") {
		t.Error("oauth_accounts waits for phase 4")
	}
}
