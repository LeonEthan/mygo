package platform

import (
	"io"
	"testing"
)

func TestSchemeBodyLimits(t *testing.T) {
	const mb = 1 << 20
	if err := SetSchemeBodyLimits(SchemeBodyLimits{5 * mb, 32 * mb}); err != nil {
		t.Fatal(err)
	}
	defer SetSchemeBodyLimits(SchemeBodyLimits{})
	var active [7]SchemeBodyReservation
	for i := range active {
		defer active[i].Release()
	}
	for i := 0; i < 6; i++ {
		if status := active[i].Reserve(5 * mb); status != 0 {
			t.Fatal(status)
		}
		if status := active[i].Reserve(1); status != 413 {
			t.Fatal(status)
		}
	}
	if status := active[6].Reserve(2 * mb); status != 0 {
		t.Fatal(status)
	}
	if status := active[6].Reserve(1); status != 503 {
		t.Fatal(status)
	}
	active[0].Release()
	active[0].Release()
	if status := active[6].Reserve(3 * mb); status != 0 {
		t.Fatal(status)
	}
}

func TestSchemeBodyRead(t *testing.T) {
	for _, tc := range []struct {
		name         string
		limit        SchemeBodyLimits
		size, status int
	}{
		{"exact", SchemeBodyLimits{5 << 20, 32 << 20}, 5 << 20, 0},
		{"request", SchemeBodyLimits{5 << 20, 32 << 20}, 5<<20 + 1, 413},
		{"inflight", SchemeBodyLimits{0, 32 << 20}, 32<<20 + 1, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := SetSchemeBodyLimits(tc.limit); err != nil {
				t.Fatal(err)
			}
			defer SetSchemeBodyLimits(SchemeBodyLimits{})
			var r SchemeBodyReservation
			defer r.Release()
			left := tc.size
			body, status := r.Read(func(buf []byte) (int, bool, error) { n := min(len(buf), left); left -= n; return n, left == 0, nil })
			if status != tc.status {
				t.Fatalf("status %d", status)
			}
			if body != nil {
				sized, ok := body.(interface{ Len() int })
				if !ok || sized.Len() != tc.size {
					t.Fatalf("buffered body does not report its length of %d", tc.size)
				}
				n, err := io.Copy(io.Discard, body)
				if err != nil || n != int64(tc.size) {
					t.Fatalf("body %d: %v", n, err)
				}
			}
		})
	}
}
