package productivity

import "testing"

func TestParsePeriod_DefaultsToThirtyDays(t *testing.T) {
	// An unreadable or absent period must still produce a report; the nightly
	// rollup only writes 7d and 30d, so anything else is 30d.
	for _, in := range []string{"", "30d", "90d", "week", "7D"} {
		want := Period30d
		if in == "7d" {
			want = Period7d
		}
		if got := ParsePeriod(in); got != want {
			t.Fatalf("ParsePeriod(%q) = %q, want %q", in, got, want)
		}
	}
	if ParsePeriod("7d") != Period7d {
		t.Fatal("7d should be honoured")
	}
}

func TestPeriodDays(t *testing.T) {
	if Period7d.Days() != 7 || Period30d.Days() != 30 {
		t.Fatalf("windows wrong: %d, %d", Period7d.Days(), Period30d.Days())
	}
}

func TestScopeFor(t *testing.T) {
	if scopeFor(true) != "team" {
		t.Fatal("an admin looks at the team")
	}
	if scopeFor(false) != "self" {
		t.Fatal("a member looks only at themselves")
	}
}

func TestMetricsPending_TrueOnlyWhenNobodyHasARollup(t *testing.T) {
	none := []Member{{Name: "A"}, {Name: "B"}}
	if !metricsPending(none) {
		t.Fatal("no rollup anywhere should read as pending")
	}

	// One member computed is enough: the page can show real numbers and must
	// not claim the job has not run.
	some := []Member{{Name: "A"}, {Name: "B", Metrics: &Metrics{EmailsSent: 3}}}
	if metricsPending(some) {
		t.Fatal("a single computed member means the job has run")
	}
}

func TestMetricsPending_ZeroRollupIsStillARollup(t *testing.T) {
	// A member who genuinely sent nothing has a row of zeroes. That is a real
	// answer, not a missing one, and must not be reported as pending.
	real := []Member{{Name: "A", Metrics: &Metrics{}}}
	if metricsPending(real) {
		t.Fatal("an all-zero rollup is a computed rollup")
	}
}

func TestRank_BusiestFirst(t *testing.T) {
	members := []Member{
		{Name: "Chi", ActionsCompleted: 2},
		{Name: "An", ActionsCompleted: 9},
		{Name: "Binh", ActionsCompleted: 5},
	}
	rank(members)

	want := []string{"An", "Binh", "Chi"}
	for i, name := range want {
		if members[i].Name != name {
			t.Fatalf("position %d = %q, want %q", i, members[i].Name, name)
		}
	}
}

func TestRank_TiesAreStableByName(t *testing.T) {
	// Everyone idle is the common case on a fresh account. Without the name
	// tiebreak the list would reshuffle on every refresh.
	members := []Member{
		{Name: "Chi"},
		{Name: "An"},
		{Name: "Binh"},
	}
	rank(members)

	for i, name := range []string{"An", "Binh", "Chi"} {
		if members[i].Name != name {
			t.Fatalf("position %d = %q, want %q", i, members[i].Name, name)
		}
	}
}

func TestRank_SkipsDoNotCountAsWorkDone(t *testing.T) {
	// A rep who skipped forty actions has not out-produced one who sent three.
	members := []Member{
		{Name: "Skipper", ActionsCompleted: 0, ActionsSkipped: 40},
		{Name: "Sender", ActionsCompleted: 3},
	}
	rank(members)

	if members[0].Name != "Sender" {
		t.Fatalf("ranked on skips, not completions: %q first", members[0].Name)
	}
}
