package template

import (
	"testing"
	"time"
)

func TestAddWorkingDays(t *testing.T) {
	fri := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC) // a Friday
	tests := []struct {
		name string
		from time.Time
		n    int
		want time.Time
	}{
		{"zero keeps the time", fri, 0, fri},
		{"Friday plus one skips the weekend", fri, 1, time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)},
		{"Friday plus five is the next Friday", fri, 5, time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC)},
		{"Saturday plus one is Monday", fri.AddDate(0, 0, 1), 1, time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddWorkingDays(tt.from, tt.n); !got.Equal(tt.want) {
				t.Fatalf("got %s, want %s", got.Format(time.RFC1123), tt.want.Format(time.RFC1123))
			}
		})
	}
}

// Each follow-up waits its days after the previous email, in working days.
// Previously send_after was added to the start as hours.
func TestSequenceSendTimes(t *testing.T) {
	mon := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	got := SequenceSendTimes(mon, []*Template{{SendAfter: 0}, {SendAfter: 2}, {SendAfter: 3}})
	want := []time.Time{
		mon, // first email at the start
		time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC), // Wed: 2 working days later
		time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC), // Mon: 3 working days after Wed
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("step %d: got %s, want %s", i, got[i].Format(time.RFC1123), want[i].Format(time.RFC1123))
		}
	}
}
