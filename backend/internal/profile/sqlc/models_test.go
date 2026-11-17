package sqlc

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestProfilesMigrationHasFeature03Columns(t *testing.T) {
	t.Parallel()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "../../../db/migrations/00004_profiles.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	needles := []string{
		"CREATE TABLE profiles",
		"user_id BIGINT PRIMARY KEY",
		"REFERENCES users (id) ON DELETE CASCADE",
		"username TEXT",
		"username_changed_at TIMESTAMPTZ",
		"bio TEXT NOT NULL DEFAULT ''",
		"avatar_object_key TEXT",
		"cover_object_key TEXT",
		"profiles_username_key",
		"WHERE username IS NOT NULL",
		"CREATE TABLE media_objects",
		"kind TEXT NOT NULL",
		"'avatar', 'cover'",
		"object_key TEXT NOT NULL",
		"media_objects_object_key_key",
		"media_objects_user_created_idx",
		"DROP TABLE IF EXISTS media_objects",
		"DROP TABLE IF EXISTS profiles",
	}
	for _, needle := range needles {
		if !strings.Contains(sql, needle) {
			t.Errorf("migration missing %q", needle)
		}
	}
	if strings.Contains(sql, "ALTER TABLE users") {
		t.Error("profile columns belong on profiles, not an ALTER of users")
	}
	if strings.Contains(sql, "kind IN ('avatar', 'cover', 'post')") {
		t.Error("post media waits for feature 05; do not add a post kind here")
	}
}
