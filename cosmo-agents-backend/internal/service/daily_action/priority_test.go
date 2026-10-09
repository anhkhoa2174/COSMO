package daily_action

import (
	"testing"
	"time"

	contact "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	outreach "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
)

func ago(d time.Duration) *time.Time {
	t := time.Now().Add(-d)
	return &t
}

// completeContact has every field GetMissingFields checks, so completeness
// contributes nothing and the other factors can be read in isolation.
func completeContact(stage string) *contact.Contact {
	return &contact.Contact{
		Name: "An", Company: "Acme", JobTitle: "CTO", Source: "csv",
		ContactInformation: "an@acme.test", BusinessStage: stage,
	}
}

func TestScoreRecency(t *testing.T) {
	tests := []struct {
		name  string
		state *outreach.OutreachState
		want  int
	}{
		{"no state", nil, 0},
		// Only a reply is a warm signal; an outgoing touch an hour ago is not.
		{"not replied", &outreach.OutreachState{ConversationState: "NO_REPLY", LastInteractionAt: ago(time.Minute)}, 0},
		{"replied without timestamp", &outreach.OutreachState{ConversationState: "REPLIED"}, 0},
		{"30m", &outreach.OutreachState{ConversationState: "REPLIED", LastInteractionAt: ago(30 * time.Minute)}, 100},
		{"2h", &outreach.OutreachState{ConversationState: "REPLIED", LastInteractionAt: ago(2 * time.Hour)}, 80},
		{"10h", &outreach.OutreachState{ConversationState: "REPLIED", LastInteractionAt: ago(10 * time.Hour)}, 60},
		{"48h", &outreach.OutreachState{ConversationState: "REPLIED", LastInteractionAt: ago(48 * time.Hour)}, 40},
		{"5d", &outreach.OutreachState{ConversationState: "REPLIED", LastInteractionAt: ago(120 * time.Hour)}, 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := scoreRecency(tc.state); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestScoreCadence(t *testing.T) {
	st := func(step string, days int) *outreach.OutreachState {
		return &outreach.OutreachState{NextStep: step, DaysSinceLastInteraction: days}
	}
	tests := []struct {
		name  string
		state *outreach.OutreachState
		want  int
	}{
		{"no state", nil, 10},
		// The 4-5 day window is the cadence the product promises; the edges
		// decide whether a rep is nudged on the right day.
		{"fu too early", st("FOLLOW_UP_1", 3), 10},
		{"fu window start", st("FOLLOW_UP_1", 4), 100},
		{"fu window end", st("FOLLOW_UP_2", 5), 100},
		{"fu slightly overdue", st("FOLLOW_UP", 6), 70},
		{"fu overdue edge", st("FOLLOW_UP", 7), 70},
		{"fu well overdue", st("FOLLOW_UP", 8), 40},
		{"send", st("SEND", 0), 50},
		{"drop", st("DROP", 30), 20},
		{"wait", st("WAIT", 2), 10},
		{"unknown", st("SET_MEETING", 2), 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := scoreCadence(tc.state); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestScoreMeetingProximity(t *testing.T) {
	m := func(status string, in time.Duration) *outreach.Meeting {
		return &outreach.Meeting{Status: status, Time: time.Now().Add(in)}
	}
	tests := []struct {
		name    string
		meeting *outreach.Meeting
		want    int
	}{
		{"none", nil, 0},
		// A cancelled meeting in an hour must not look urgent.
		{"cancelled", m("cancelled", time.Hour), 0},
		{"passed", m("scheduled", -time.Hour), 0},
		{"1h", m("scheduled", time.Hour), 100},
		{"3h", m("scheduled", 3*time.Hour), 90},
		{"6h", m("scheduled", 6*time.Hour), 75},
		{"12h", m("scheduled", 12*time.Hour), 50},
		{"36h", m("scheduled", 36*time.Hour), 30},
		{"4d", m("scheduled", 96*time.Hour), 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := scoreMeetingProximity(tc.meeting); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestScoreCompleteness(t *testing.T) {
	tests := []struct {
		name string
		c    *contact.Contact
		want int
	}{
		{"complete", completeContact("LEAD"), 0},
		{"one missing", &contact.Contact{Name: "An", Company: "Acme", JobTitle: "CTO", Source: "csv"}, 10},
		{"two missing", &contact.Contact{Name: "An", Company: "Acme", Source: "csv"}, 30},
		{"three missing", &contact.Contact{Name: "An", Source: "csv"}, 50},
		{"empty", &contact.Contact{}, 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := scoreCompleteness(tc.c); got != tc.want {
				t.Fatalf("got %d, want %d (missing=%v)", got, tc.want, tc.c.GetMissingFields())
			}
		})
	}
}

// Migration 000050 rewrote every stored business_stage to the lifecycle names
// (PRE_SALES→LEAD, SALES→OPPORTUNITY, POST_SALES→CUSTOMER) and made LEAD the
// default. Scoring only the legacy names would give every real contact the
// "unknown" 50 and switch the lifecycle factor off for the whole product.
func TestScoreLifecycle(t *testing.T) {
	tests := []struct {
		stage string
		want  int
	}{
		{"PRE_SALES", 100}, {"LEAD", 100}, {"SUBSCRIBER", 100},
		{"SALES", 50}, {"OPPORTUNITY", 50}, {"QUALIFIED", 50},
		{"POST_SALES", 10}, {"CUSTOMER", 10}, {"ADVOCATE", 10},
		{"", 50}, {"MYSTERY", 50},
	}
	for _, tc := range tests {
		t.Run(tc.stage, func(t *testing.T) {
			if got, _ := scoreLifecycle(&contact.Contact{BusinessStage: tc.stage}); got != tc.want {
				t.Fatalf("stage %q: got %d, want %d", tc.stage, got, tc.want)
			}
		})
	}
}

func TestComputePriority(t *testing.T) {
	hot := &outreach.OutreachState{ConversationState: "REPLIED", LastInteractionAt: ago(10 * time.Minute),
		NextStep: "FOLLOW_UP_1", DaysSinceLastInteraction: 4}
	soon := &outreach.Meeting{Status: "scheduled", Time: time.Now().Add(time.Hour)}

	t.Run("everything urgent clamps to 1", func(t *testing.T) {
		score, factors := ComputePriority(&contact.Contact{BusinessStage: "LEAD"}, hot, soon)
		if score != 1 {
			t.Fatalf("score = %d, want 1", score)
		}
		if len(factors) != 5 {
			t.Fatalf("want 5 factors, got %d", len(factors))
		}
	})

	t.Run("nothing urgent", func(t *testing.T) {
		// cadence 10 → 2.5, lifecycle customer 10 → 0.8: urgency 3.3 → 97.
		score, _ := ComputePriority(completeContact("CUSTOMER"), nil, nil)
		if score != 97 {
			t.Fatalf("score = %d, want 97", score)
		}
	})

	t.Run("fresh reply outranks stale follow-up", func(t *testing.T) {
		replied, _ := ComputePriority(completeContact("LEAD"),
			&outreach.OutreachState{ConversationState: "REPLIED", LastInteractionAt: ago(30 * time.Minute)}, nil)
		stale, _ := ComputePriority(completeContact("LEAD"),
			&outreach.OutreachState{ConversationState: "NO_REPLY", NextStep: "FOLLOW_UP", DaysSinceLastInteraction: 20}, nil)
		if replied >= stale {
			t.Fatalf("reply %d should rank ahead of stale follow-up %d", replied, stale)
		}
	})

	t.Run("lead outranks customer, all else equal", func(t *testing.T) {
		lead, _ := ComputePriority(completeContact("LEAD"), nil, nil)
		cust, _ := ComputePriority(completeContact("CUSTOMER"), nil, nil)
		if lead >= cust {
			t.Fatalf("lead %d should rank ahead of customer %d", lead, cust)
		}
	})
}
