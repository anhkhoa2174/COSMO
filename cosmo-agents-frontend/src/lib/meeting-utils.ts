import type { Meeting } from '@/network/client/outreach';

/**
 * Suggests a meeting time that doesn't overlap with existing meetings.
 *
 * Algorithm:
 * 1. Start with 2 days from now at 10:00 AM local time
 * 2. Check for overlaps with existing scheduled meetings
 * 3. If overlap, shift by 1 hour
 * 4. If no slot found in business hours (10-17), shift to next business day at 10:00
 * 5. Repeat until a free slot is found
 */
export function suggestMeetingTime(
  existingMeetings: Meeting[],
  durationMinutes: number = 30
): string {
  const now = new Date();
  const suggested = new Date(now);
  suggested.setDate(suggested.getDate() + 2);
  suggested.setHours(10, 0, 0, 0);

  // Skip weekends for initial date
  skipToBusinessDay(suggested);

  const scheduledMeetings = existingMeetings.filter(
    (m) => m.status === 'scheduled'
  );

  // Try up to 30 business days to find a free slot
  for (let dayAttempt = 0; dayAttempt < 30; dayAttempt++) {
    for (let hour = 10; hour <= 17; hour++) {
      suggested.setHours(hour, 0, 0, 0);

      if (!hasOverlap(suggested, durationMinutes, scheduledMeetings)) {
        return formatForDatetimeLocal(suggested);
      }
    }

    // No slot found on this day, move to next business day
    suggested.setDate(suggested.getDate() + 1);
    skipToBusinessDay(suggested);
  }

  // Fallback: return the original suggestion (2 days from now at 10 AM)
  const fallback = new Date(now);
  fallback.setDate(fallback.getDate() + 2);
  fallback.setHours(10, 0, 0, 0);
  return formatForDatetimeLocal(fallback);
}

function skipToBusinessDay(date: Date): void {
  const day = date.getDay();
  if (day === 0) date.setDate(date.getDate() + 1); // Sunday -> Monday
  if (day === 6) date.setDate(date.getDate() + 2); // Saturday -> Monday
}

function hasOverlap(
  suggestedStart: Date,
  durationMinutes: number,
  meetings: Meeting[]
): boolean {
  const suggestedEnd = new Date(
    suggestedStart.getTime() + durationMinutes * 60_000
  );

  return meetings.some((meeting) => {
    const meetingStart = new Date(meeting.time);
    const meetingEnd = new Date(
      meetingStart.getTime() + (meeting.duration_minutes || 30) * 60_000
    );

    // Overlap: suggestedStart < meetingEnd AND suggestedEnd > meetingStart
    return suggestedStart < meetingEnd && suggestedEnd > meetingStart;
  });
}

function formatForDatetimeLocal(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}
