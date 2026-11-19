package blob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateKey(t *testing.T) {
	t.Parallel()
	ok := []string{
		"profile/193848201234567890/avatar/193848201234567891.jpg",
		"profile/1/cover/example-object.jpg",
	}
	for _, key := range ok {
		if err := ValidateKey(key); err != nil {
			t.Errorf("ValidateKey(%q) = %v", key, err)
		}
	}
	bad := []string{
		"",
		"/profile/1/avatar/1.jpg",
		"profile/../avatar/1.jpg",
		"profile/1/avatar/../../etc/passwd.jpg",
		"profile/1/avatar/1.png",
		"profile/1/post/1.jpg",
		"etc/passwd",
		"PROFILE/1/avatar/1.jpg",
	}
	for _, key := range bad {
		if err := ValidateKey(key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("ValidateKey(%q) = %v, want ErrInvalidKey", key, err)
		}
	}
}

func TestPublicURL(t *testing.T) {
	t.Parallel()
	got := PublicURL("http://localhost:8080/v1/media/", "profile/1/avatar/2.jpg")
	want := "http://localhost:8080/v1/media/profile/1/avatar/2.jpg"
	if got != want {
		t.Errorf("PublicURL = %q, want %q", got, want)
	}
	if PublicURL("", "profile/1/avatar/2.jpg") != "" {
		t.Error("empty base should yield empty URL")
	}
}

func TestFSRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store, err := NewFS(t.TempDir())
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	key := "profile/193848201234567890/avatar/193848201234567891.jpg"
	body := []byte("fake-jpeg-bytes")
	if err := store.Put(ctx, key, jpegType, bytes.NewReader(body)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	rc, ct, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	if ct != jpegType {
		t.Errorf("contentType = %q", ct)
	}
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("body = %q", got)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := store.Get(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete: %v", err)
	}
}

func TestFSRejectsTraversal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()
	store, err := NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	if err := store.Put(ctx, "../secret.jpg", jpegType, strings.NewReader("x")); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Put traversal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "secret.jpg")); !errors.Is(err, os.ErrNotExist) {
		t.Error("traversal wrote outside root")
	}
}

func TestOpenFS(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st, err := Open(ctx, Config{Driver: "fs", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, ok := st.(*FS); !ok {
		t.Fatalf("Open fs type %T", st)
	}
}

func TestOpenUnknownDriver(t *testing.T) {
	t.Parallel()
	if _, err := Open(context.Background(), Config{Driver: "gcs"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestEndpointURL(t *testing.T) {
	t.Parallel()
	got, err := endpointURL("127.0.0.1:9000", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:9000" {
		t.Errorf("got %q", got)
	}
	got, err = endpointURL("http://127.0.0.1:9000", true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:9000" {
		t.Errorf("scheme from URL should win, got %q", got)
	}
	if _, err := endpointURL("", false); err == nil {
		t.Fatal("empty endpoint should fail")
	}
}

func TestS3OptionalRoundTrip(t *testing.T) {
	endpoint := os.Getenv("MEDIA_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("MEDIA_S3_ENDPOINT not set")
	}
	ctx := context.Background()
	store, err := NewS3(ctx, S3Config{
		Endpoint:  endpoint,
		Region:    envOr("MEDIA_S3_REGION", "us-east-1"),
		Bucket:    envOr("MEDIA_S3_BUCKET", "friendfold-media"),
		AccessKey: envOr("MEDIA_S3_ACCESS_KEY", "friendfold"),
		SecretKey: envOr("MEDIA_S3_SECRET_KEY", "friendfold"),
		UseSSL:    false,
		PathStyle: true,
	})
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}
	key := "profile/1/avatar/example-object.jpg"
	body := []byte("fake-jpeg-bytes")
	if err := store.Put(ctx, key, jpegType, bytes.NewReader(body)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), key) })
	rc, _, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("body = %q", got)
	}
}

func envOr(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
