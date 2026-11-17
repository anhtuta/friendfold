package profile

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"io"
	"sync"
	"testing"

	"github.com/anhtuta/friendfold/backend/internal/platform/blob"
)

type memBlob struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMemBlob() *memBlob {
	return &memBlob{data: map[string][]byte{}}
}

func (m *memBlob) Put(_ context.Context, key, _ string, r io.Reader) error {
	if err := blob.ValidateKey(key); err != nil {
		return err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.data[key] = b
	m.mu.Unlock()
	return nil
}

func (m *memBlob) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	if err := blob.ValidateKey(key); err != nil {
		return nil, "", err
	}
	m.mu.Lock()
	b, ok := m.data[key]
	m.mu.Unlock()
	if !ok {
		return nil, "", blob.ErrNotFound
	}
	return seekCloser{Reader: bytes.NewReader(b)}, "image/jpeg", nil
}

type seekCloser struct {
	*bytes.Reader
}

func (seekCloser) Close() error { return nil }

func (m *memBlob) Delete(_ context.Context, key string) error {
	if err := blob.ValidateKey(key); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; !ok {
		return blob.ErrNotFound
	}
	delete(m.data, key)
	return nil
}

type seqIDs struct {
	n int64
}

func (s *seqIDs) Next(context.Context) (int64, error) {
	s.n++
	return 1_000_000_000_000 + s.n, nil
}

func testMediaService(t *testing.T) (*Service, *memBlob) {
	t.Helper()
	svc, _, acct, _ := testProfileService(t)
	acct.put(Account{ID: 1, Email: "minh@example.com", DisplayName: "Minh"})
	store := newMemBlob()
	svc.WithMedia(store, &seqIDs{})
	return svc, store
}

func TestUploadAvatarJPEG(t *testing.T) {
	t.Parallel()
	svc, store := testMediaService(t)
	me, err := svc.UploadImage(context.Background(), 1, "203.0.113.9", KindAvatar, encodeJPEG(t, 40, 40))
	if err != nil {
		t.Fatal(err)
	}
	if me.AvatarURL == "" {
		t.Fatal("missing avatar URL")
	}
	if me.Email != "minh@example.com" {
		t.Fatalf("email: %s", me.Email)
	}
	key := me.AvatarURL[len(testPublicBase)+1:]
	rc, ct, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(rc)
	_ = rc.Close()
	if ct != "image/jpeg" || !bytes.HasPrefix(raw, []byte{0xff, 0xd8}) {
		t.Fatalf("stored %s prefix %x", ct, raw[:2])
	}
}

func TestUploadRejectsGIF(t *testing.T) {
	t.Parallel()
	svc, _ := testMediaService(t)
	var buf bytes.Buffer
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UploadImage(context.Background(), 1, "203.0.113.9", KindAvatar, buf.Bytes()); !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("gif: %v", err)
	}
}

func TestDeleteAvatarClearsURL(t *testing.T) {
	t.Parallel()
	svc, _ := testMediaService(t)
	if _, err := svc.UploadImage(context.Background(), 1, "203.0.113.9", KindAvatar, encodeJPEG(t, 20, 20)); err != nil {
		t.Fatal(err)
	}
	me, err := svc.DeleteImage(context.Background(), 1, "203.0.113.9", KindAvatar)
	if err != nil {
		t.Fatal(err)
	}
	if me.AvatarURL != "" {
		t.Fatalf("avatar still set: %s", me.AvatarURL)
	}
}

func TestReplaceAvatarDeletesPrevious(t *testing.T) {
	t.Parallel()
	svc, store := testMediaService(t)
	first, err := svc.UploadImage(context.Background(), 1, "203.0.113.9", KindAvatar, encodeJPEG(t, 20, 20))
	if err != nil {
		t.Fatal(err)
	}
	oldKey := first.AvatarURL[len(testPublicBase)+1:]
	second, err := svc.UploadImage(context.Background(), 1, "203.0.113.9", KindAvatar, encodeJPEG(t, 24, 24))
	if err != nil {
		t.Fatal(err)
	}
	newKey := second.AvatarURL[len(testPublicBase)+1:]
	if oldKey == newKey {
		t.Fatal("key not rotated")
	}
	if _, _, err := store.Get(context.Background(), oldKey); !errors.Is(err, blob.ErrNotFound) {
		t.Fatalf("old object kept: %v", err)
	}
	if _, _, err := store.Get(context.Background(), newKey); err != nil {
		t.Fatal(err)
	}
}
