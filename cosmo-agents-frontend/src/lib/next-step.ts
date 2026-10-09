import { format } from 'date-fns';

import type { NextAction } from '@/network/client/outreach';

/** How each next-step action reads in the UI. */
export const nextActionLabels: Record<NextAction, string> = {
  SEND_INTRO: 'Send introduction',
  SEND_FOLLOW_UP: 'Send follow-up',
  ANSWER_REPLY: 'Answer reply',
  PROPOSE_MEETING: 'Propose meeting',
  MEETING_FOLLOW_UP: 'Meeting follow-up',
  CONTACT_STAKEHOLDER: 'Contact stakeholder',
  SWITCH_CHANNEL: 'Switch channel',
  WAIT: 'Wait',
  NURTURE: 'Nurture',
  FIX_DATA: 'Fix data',
  SUPPRESS: 'Suppressed',
  DISQUALIFY: 'Disqualify',
  ESCALATE: 'Escalate to a person',
};

/** Colour by what the action does: send, hold, close, or hand over. */
export const nextActionColors: Record<NextAction, string> = {
  SEND_INTRO: 'bg-blue-600',
  SEND_FOLLOW_UP: 'bg-blue-600',
  ANSWER_REPLY: 'bg-emerald-600',
  PROPOSE_MEETING: 'bg-emerald-600',
  MEETING_FOLLOW_UP: 'bg-emerald-600',
  CONTACT_STAKEHOLDER: 'bg-violet-600',
  SWITCH_CHANNEL: 'bg-violet-600',
  WAIT: 'bg-amber-500',
  NURTURE: 'bg-amber-600',
  FIX_DATA: 'bg-orange-600',
  SUPPRESS: 'bg-gray-600',
  DISQUALIFY: 'bg-gray-600',
  ESCALATE: 'bg-red-600',
};

/**
 * The Outreach filter chips. Thirteen actions are too many to filter by one at
 * a time, so they are grouped by what the rep does next.
 */
export const NEXT_ACTION_GROUPS: {
  id: string;
  label: string;
  actions: NextAction[];
}[] = [
  { id: 'intro', label: 'Introduce', actions: ['SEND_INTRO'] },
  {
    id: 'follow-up',
    label: 'Follow up',
    actions: ['SEND_FOLLOW_UP', 'SWITCH_CHANNEL'],
  },
  { id: 'reply', label: 'Answer reply', actions: ['ANSWER_REPLY'] },
  {
    id: 'meeting',
    label: 'Meeting',
    actions: ['PROPOSE_MEETING', 'MEETING_FOLLOW_UP'],
  },
  { id: 'stakeholder', label: 'Stakeholder', actions: ['CONTACT_STAKEHOLDER'] },
  { id: 'waiting', label: 'Waiting', actions: ['WAIT', 'NURTURE'] },
  {
    id: 'review',
    label: 'Needs review',
    actions: ['FIX_DATA', 'ESCALATE', 'SUPPRESS', 'DISQUALIFY'],
  },
];

export function nextActionGroup(action?: string | null): string | undefined {
  if (!action) return undefined;
  return NEXT_ACTION_GROUPS.find((g) =>
    g.actions.includes(action as NextAction)
  )?.id;
}

/**
 * Most urgent first: contacts with a decision, by when it falls due (overdue
 * at the top), then contacts the engine has not decided on yet.
 */
export function compareByNextActionDue(
  a: { next_action?: string | null; next_action_due_at?: string | null },
  b: { next_action?: string | null; next_action_due_at?: string | null }
): number {
  const rank = (c: typeof a) => {
    if (!c.next_action) return Number.POSITIVE_INFINITY;
    const t = c.next_action_due_at ? Date.parse(c.next_action_due_at) : NaN;
    // A decision with no due date is actionable now.
    return Number.isNaN(t) ? 0 : t;
  };
  const ra = rank(a);
  const rb = rank(b);
  if (ra === rb) return 0;
  return ra < rb ? -1 : 1;
}

const fmtDay = (v?: string) => {
  if (!v) return '';
  const d = new Date(v.length === 10 ? `${v}T00:00:00` : v);
  return Number.isNaN(d.getTime()) ? v : format(d, 'd MMM yyyy');
};

/** One line describing the action with its arguments. */
export function describeNextAction(
  action: NextAction,
  args?: Record<string, any>,
  dueAt?: string
): string {
  const a = args ?? {};
  const label = nextActionLabels[action] ?? action;
  switch (action) {
    case 'WAIT':
      return `${label} until ${fmtDay(a.until ?? dueAt)}${a.reason ? ` (${a.reason})` : ''}`;
    case 'NURTURE':
      return `${label}${a.reason ? ` (${String(a.reason).replace('_', ' ')})` : ''}, revisit ${fmtDay(a.revisit ?? dueAt)}`;
    case 'ANSWER_REPLY':
      return a.kind && a.kind !== 'answer'
        ? `${label}: ${a.kind}${a.objection ? ` (${a.objection})` : ''}`
        : label;
    case 'MEETING_FOLLOW_UP':
      return a.kind ? `${label}: ${a.kind}` : label;
    case 'CONTACT_STAKEHOLDER':
      return `${label}: ${a.person || a.email || 'someone else'}${a.referrer ? ` (referred by ${a.referrer})` : ''}`;
    case 'SWITCH_CHANNEL':
      return `${label} to ${a.channel === 'call' ? 'a call' : 'LinkedIn'}`;
    case 'SEND_INTRO':
    case 'SEND_FOLLOW_UP':
      return a.angle ? `${label}: ${a.angle}` : label;
    default:
      return label;
  }
}

/**
 * The same description without the action's own name, for places that already
 * show the name in a badge beside it ("objection (price)" rather than
 * "Answer reply: objection (price)").
 */
export function nextActionDetail(
  action: NextAction,
  args?: Record<string, any>,
  dueAt?: string
): string {
  const full = describeNextAction(action, args, dueAt);
  const label = nextActionLabels[action] ?? action;
  if (!full.startsWith(label)) return full;
  return full.slice(label.length).replace(/^[:\s]+/, '');
}
