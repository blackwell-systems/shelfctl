package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
)

// Reader is an io.Reader that accumulates sha256 and total byte count in-flight.
type Reader struct {
	r    io.Reader
	h    hash.Hash
	size int64
}

// NewReader wraps r with in-flight sha256 and size tracking.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r, h: sha256.New()}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if n > 0 {
		r.h.Write(p[:n]) //nolint:errcheck — hash.Write never errors
		r.size += int64(n)
	}
	return
}

// SHA256 returns the hex-encoded sha256 of all bytes read so far.
func (r *Reader) SHA256() string {
	return hex.EncodeToString(r.h.Sum(nil))
}

// Size returns the total bytes read so far.
func (r *Reader) Size() int64 { return r.size }
