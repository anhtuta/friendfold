package profile

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestTranscodeJPEGPNGWebP(t *testing.T) {
	t.Parallel()
	jpegRaw := encodeJPEG(t, 32, 24)
	got, err := transcodeImage(jpegRaw, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 32 || got.Height != 24 || !bytes.HasPrefix(got.JPEG, []byte{0xff, 0xd8}) {
		t.Fatalf("%d x %d prefix=%x", got.Width, got.Height, got.JPEG[:2])
	}

	pngRaw := encodePNG(t, 16, 16)
	if _, err := transcodeImage(pngRaw, 1024); err != nil {
		t.Fatalf("png: %v", err)
	}
	if _, err := transcodeImage(tinyWebP, 1024); err != nil {
		t.Fatalf("webp: %v", err)
	}
}

func TestTranscodeRejectsGIFAndPDF(t *testing.T) {
	t.Parallel()
	var gifBuf bytes.Buffer
	img := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	if err := gif.Encode(&gifBuf, img, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := transcodeImage(gifBuf.Bytes(), 1024); err == nil {
		t.Fatal("gif accepted")
	}
	if _, err := transcodeImage([]byte("%PDF-1.4\n%"), 1024); err == nil {
		t.Fatal("pdf accepted")
	}
}

func TestTranscodeStripsEXIF(t *testing.T) {
	t.Parallel()
	raw := jpegWithGPSAPP1(t)
	if !bytes.Contains(raw, []byte("Exif")) || !bytes.Contains(raw, []byte("GPS")) {
		t.Fatal("fixture missing EXIF GPS")
	}
	got, err := transcodeImage(raw, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(got.JPEG, []byte{0xff, 0xe1}) || bytes.Contains(got.JPEG, []byte("GPS")) {
		t.Fatal("transcoded JPEG still has APP1/GPS")
	}
}

func TestTranscodePixelBombHeader(t *testing.T) {
	t.Parallel()
	raw := patchJPEGSize(encodeJPEG(t, 8, 8), 9000, 9000)
	if _, err := transcodeImage(raw, 1024); err == nil {
		t.Fatal("pixel bomb accepted")
	}
}

func TestTranscodeResizesLongEdge(t *testing.T) {
	t.Parallel()
	raw := encodeJPEG(t, 2000, 1000)
	got, err := transcodeImage(raw, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 1024 || got.Height != 512 {
		t.Fatalf("got %d x %d", got.Width, got.Height)
	}
}

func TestFitLongEdgeNoUpscale(t *testing.T) {
	t.Parallel()
	w, h := fitLongEdge(100, 50, 1024)
	if w != 100 || h != 50 {
		t.Fatalf("%d x %d", w, h)
	}
}

func encodeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 40, G: 80, B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func jpegWithGPSAPP1(t *testing.T) []byte {
	t.Helper()
	raw := encodeJPEG(t, 8, 8)
	app1 := []byte{
		0xff, 0xe1, 0x00, 0x0c,
		'E', 'x', 'i', 'f', 0x00, 0x00,
		'G', 'P', 'S', 0x00,
	}
	out := make([]byte, 0, 2+len(app1)+len(raw)-2)
	out = append(out, 0xff, 0xd8)
	out = append(out, app1...)
	out = append(out, raw[2:]...)
	return out
}

func patchJPEGSize(raw []byte, w, h uint16) []byte {
	out := append([]byte(nil), raw...)
	for i := 0; i < len(out)-8; i++ {
		if out[i] == 0xff && (out[i+1] == 0xc0 || out[i+1] == 0xc1 || out[i+1] == 0xc2) {
			out[i+5] = byte(h >> 8)
			out[i+6] = byte(h)
			out[i+7] = byte(w >> 8)
			out[i+8] = byte(w)
			return out
		}
	}
	return out
}

// tinyWebP is a 1×1 lossy WebP (not a production asset).
var tinyWebP = []byte{
	0x52, 0x49, 0x46, 0x46, 0x24, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
	0x56, 0x50, 0x38, 0x20, 0x18, 0x00, 0x00, 0x00, 0x30, 0x01, 0x00, 0x9d,
	0x01, 0x2a, 0x01, 0x00, 0x01, 0x00, 0x02, 0x00, 0x34, 0x25, 0xa4, 0x00,
	0x03, 0x70, 0x00, 0xfe, 0xfb, 0xfd, 0x80, 0x00,
}
