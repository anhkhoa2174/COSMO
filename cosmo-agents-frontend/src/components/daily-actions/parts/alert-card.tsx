'use client';

import type { AlertData } from '@/types/daily-actions';

const alertEmojiMap: Record<string, string> = {
  prospect_replied: '🔔',
  meeting_approaching: '📅',
  snooze_expired: '⏰',
};

const alertStyleMap: Record<string, { bg: string; border: string; text: string }> = {
  prospect_replied: {
    bg: 'bg-emerald-500/[0.1]',
    border: 'border-emerald-500/[0.2]',
    text: 'text-emerald-600 dark:text-emerald-400',
  },
  meeting_approaching: {
    bg: 'bg-violet-500/[0.1]',
    border: 'border-violet-500/[0.2]',
    text: 'text-violet-600 dark:text-violet-400',
  },
  snooze_expired: {
    bg: 'bg-amber-500/[0.1]',
    border: 'border-amber-500/[0.2]',
    text: 'text-amber-600 dark:text-amber-400',
  },
};

interface AlertCardProps {
  alert: AlertData;
}

export function AlertCard({ alert }: AlertCardProps) {
  const emoji = alertEmojiMap[alert.alert_type] || '🔔';
  const styles = alertStyleMap[alert.alert_type] || alertStyleMap.prospect_replied;

  return (
    <div className={`mt-4 overflow-hidden rounded-[10px] border ${styles.border} bg-card animate-in fade-in slide-in-from-top-2`}>
      {/* Colored header */}
      <div className={`flex items-center gap-2.5 px-[14px] py-3 ${styles.bg} ${styles.text} font-bold text-[0.85rem]`}>
        <span className="text-[1rem]">{emoji}</span>
        <span>{alert.alert_type === 'prospect_replied' ? 'New Reply' : alert.alert_type === 'meeting_approaching' ? 'Meeting Approaching' : 'Reminder'}</span>
        <span className="ml-auto font-mono text-[0.65rem] font-medium opacity-70">
          {new Date(alert.timestamp).toLocaleTimeString([], {
            hour: '2-digit',
            minute: '2-digit',
          })}
        </span>
      </div>

      {/* Body */}
      <div className="p-5 space-y-2">
        <p className="text-[0.92rem] font-bold text-foreground">{alert.contact.name}</p>
        <p className="text-[0.85rem] text-muted-foreground leading-relaxed">{alert.content}</p>

        {/* Agent reasoning block */}
        <div className="relative my-2 rounded-lg border border-border bg-muted p-[11px_14px] text-[0.82rem] italic text-muted-foreground leading-relaxed">
          <span className="absolute -top-2 left-3 bg-muted px-1.5 text-[0.56rem] font-bold not-italic uppercase tracking-[0.08em] text-muted-foreground/80 font-mono border border-border rounded-sm">
            reasoning
          </span>
          {alert.agent_reasoning}
        </div>

        <p className="text-[0.85rem] leading-relaxed text-muted-foreground">
          <strong className="text-foreground">Recommended: </strong>
          {alert.recommended_action}
        </p>
      </div>
    </div>
  );
}
