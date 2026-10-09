package template

import "time"

// AddWorkingDays moves t forward by n working days, skipping Saturdays and
// Sundays. The time of day is kept. n <= 0 returns t unchanged.
func AddWorkingDays(t time.Time, n int) time.Time {
	for n > 0 {
		t = t.AddDate(0, 0, 1)
		if wd := t.Weekday(); wd != time.Saturday && wd != time.Sunday {
			n--
		}
	}
	return t
}

// SequenceSendTimes returns when each email of a campaign sequence is due,
// given the templates in send order (by position) and the campaign's start.
//
// A template's SendAfter is the number of working days to wait after the
// previous email, as the campaign editor says ("Send if contact does not reply
// in N working days"). Each step is therefore offset from the one before it,
// not from the start. The first email waits its own SendAfter from the start,
// normally zero.
func SequenceSendTimes(start time.Time, templates []*Template) []time.Time {
	times := make([]time.Time, len(templates))
	at := start
	for i, tpl := range templates {
		if tpl != nil {
			at = AddWorkingDays(at, tpl.SendAfter)
		}
		times[i] = at
	}
	return times
}
