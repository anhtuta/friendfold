package profile

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	avatarMaxBytes = 5 << 20
	coverMaxBytes  = 10 << 20
	multipartSlack = 64 << 10
	avatarMaxLong  = 1024
	coverMaxLong   = 2048
	maxImageEdge   = 8192
	maxImagePixels = 40_000_000
	jpegQuality    = 85
)

type transcoded struct {
	JPEG   []byte
	Width  int
	Height int
}

func transcodeImage(raw []byte, maxLong int) (transcoded, error) {
	if len(raw) < 12 {
		return transcoded{}, fmt.Errorf("%w", ErrInvalidImage)
	}
	sniff := raw
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	switch http.DetectContentType(sniff) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return transcoded{}, fmt.Errorf("%w", ErrInvalidImage)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return transcoded{}, fmt.Errorf("%w", ErrInvalidImage)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return transcoded{}, fmt.Errorf("%w", ErrInvalidImage)
	}
	if cfg.Width > maxImageEdge || cfg.Height > maxImageEdge {
		return transcoded{}, fmt.Errorf("%w", ErrInvalidImage)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return transcoded{}, fmt.Errorf("%w", ErrInvalidImage)
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return transcoded{}, fmt.Errorf("%w", ErrInvalidImage)
	}
	nw, nh := fitLongEdge(cfg.Width, cfg.Height, maxLong)
	out := src
	if nw != cfg.Width || nh != cfg.Height {
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		out = dst
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return transcoded{}, fmt.Errorf("profile jpeg encode: %w", err)
	}
	return transcoded{JPEG: buf.Bytes(), Width: nw, Height: nh}, nil
}

func fitLongEdge(w, h, maxLong int) (int, int) {
	if w <= 0 || h <= 0 || maxLong <= 0 {
		return w, h
	}
	long := w
	if h > long {
		long = h
	}
	if long <= maxLong {
		return w, h
	}
	nw := w * maxLong / long
	nh := h * maxLong / long
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return nw, nh
}
