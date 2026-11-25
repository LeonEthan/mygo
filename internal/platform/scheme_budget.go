package platform

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

type SchemeBodyLimits struct {
	MaxRequestBodyBytes         int64
	MaxInFlightRequestBodyBytes int64
}

var schemeBudget struct {
	sync.Mutex
	SchemeBodyLimits
	used int64
}

func SetSchemeBodyLimits(l SchemeBodyLimits) error {
	if l.MaxRequestBodyBytes < 0 || l.MaxInFlightRequestBodyBytes < 0 {
		return errors.New("negative scheme body limit")
	}
	schemeBudget.Lock()
	defer schemeBudget.Unlock()
	schemeBudget.SchemeBodyLimits = l
	return nil
}

type SchemeBodyReservation struct{ size int64 }

func (r *SchemeBodyReservation) Reserve(n int64) int {
	schemeBudget.Lock()
	defer schemeBudget.Unlock()
	if n < 0 || schemeBudget.MaxRequestBodyBytes > 0 && n > schemeBudget.MaxRequestBodyBytes-r.size {
		return 413
	}
	if schemeBudget.MaxInFlightRequestBodyBytes > 0 && n > schemeBudget.MaxInFlightRequestBodyBytes-schemeBudget.used {
		return 503
	}
	schemeBudget.used += n
	r.size += n
	return 0
}
func (r *SchemeBodyReservation) Release() {
	schemeBudget.Lock()
	defer schemeBudget.Unlock()
	schemeBudget.used -= r.size
	r.size = 0
}

type bufferedSchemeBody struct {
	io.ReadCloser
	length int
}

func (b bufferedSchemeBody) Len() int { return b.length }

// Read collects fixed-size chunks so a stream returning one byte at a time
// consumes the same storage as one returning full chunks. Only the current
// native read uses scratch space outside the payload reservation.
func (r *SchemeBodyReservation) Read(read func([]byte) (int, bool, error)) (io.ReadCloser, int) {
	buf := make([]byte, 64<<10)
	var chunks []io.Reader
	used := 0
	for {
		n, eof, err := read(buf[used:])
		if err != nil || n < 0 || n > len(buf)-used || n == 0 && !eof {
			return nil, 500
		}
		if status := r.Reserve(int64(n)); status != 0 {
			return nil, status
		}
		used += n
		if used == len(buf) || eof {
			if used > 0 {
				chunks = append(chunks, bytes.NewReader(bytes.Clone(buf[:used])))
			}
			used = 0
		}
		if eof {
			return bufferedSchemeBody{io.NopCloser(io.MultiReader(chunks...)), int(r.size)}, 0
		}
	}
}
