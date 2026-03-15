package worker

import (
	"testing"
	"time"
)

func TestCalculateFlushBackoff(t *testing.T) {
	t.Parallel()

	cases := []struct {
		failures int
		want     time.Duration
	}{
		{failures: 1, want: 200 * time.Millisecond},
		{failures: 2, want: 500 * time.Millisecond},
		{failures: 3, want: 1 * time.Second},
		{failures: 4, want: 2 * time.Second},
		{failures: 5, want: 5 * time.Second},
		{failures: 20, want: 5 * time.Second},
	}

	for _, tc := range cases {
		t.Run(time.Duration(tc.failures).String(), func(t *testing.T) {
			got := calculateFlushBackoff(tc.failures)
			if got != tc.want {
				t.Fatalf("calculateFlushBackoff(%d) = %s, want %s", tc.failures, got, tc.want)
			}
		})
	}
}
