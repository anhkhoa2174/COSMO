'use client';

import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import DailyActionsApi from '@/network/client/daily-actions';
import Cookies from 'js-cookie';
import type { DailyActionSSEEvent } from '@/types/daily-actions';

/**
 * Global SSE listener that shows toast notifications for pipeline events
 * regardless of which page the user is on.
 *
 * Mounted in the main layout — always active while app is open.
 */
export function GlobalNotifications() {
  const queryClient = useQueryClient();
  const eventSourceRef = useRef<EventSource | null>(null);

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const data: DailyActionSSEEvent = JSON.parse(event.data);
        const today = new Date().toISOString().split('T')[0];

        switch (data.event_type) {
          case 'prospect_replied':
            toast.info(
              `${data.contact?.name || 'A prospect'} replied`,
              {
                description: data.reply_preview
                  ? data.reply_preview.slice(0, 100)
                  : 'New reply received — AI has drafted a response',
                action: {
                  label: 'View',
                  onClick: () => {
                    window.location.href = '/daily-actions';
                  },
                },
                duration: 10000,
              }
            );
            queryClient.invalidateQueries({ queryKey: ['daily-actions', today] });
            break;

          case 'meeting_approaching':
            toast.info(
              `Meeting with ${data.contact?.name || 'contact'} approaching`,
              {
                description: 'Review meeting prep and talking points',
                action: {
                  label: 'Prepare',
                  onClick: () => {
                    window.location.href = '/daily-actions';
                  },
                },
                duration: 10000,
              }
            );
            break;

          case 'followup_due':
            toast.info(
              `Follow-up due for ${data.contact?.name || 'contact'}`,
              {
                description: 'Time to send a follow-up message',
                duration: 8000,
              }
            );
            break;

          case 'generation_complete':
            toast.success('Daily briefing ready', {
              description: 'AI has prepared your action plan for today',
              action: {
                label: 'View',
                onClick: () => {
                  window.location.href = '/daily-actions';
                },
              },
              duration: 5000,
            });
            queryClient.invalidateQueries({ queryKey: ['daily-actions', today] });
            break;

          case 'snooze_expired':
            toast.info('Snoozed action is now due', { duration: 8000 });
            queryClient.invalidateQueries({ queryKey: ['daily-actions', today] });
            break;

          case 'action_updated':
            queryClient.invalidateQueries({ queryKey: ['daily-actions', today] });
            break;
        }
      } catch {
        // Ignore malformed events
      }
    },
    [queryClient]
  );

  useEffect(() => {
    let cancelled = false;

    function connect() {
      const token = Cookies.get('access_token');
      if (cancelled || !token) return;

      const eventSource = DailyActionsApi.openEventStream(token);
      eventSourceRef.current = eventSource;
      // Named SSE events need addEventListener (onmessage only handles unnamed events)
      const eventTypes = ['prospect_replied', 'meeting_approaching', 'followup_due', 'generation_complete', 'snooze_expired', 'action_updated'];
      for (const eventType of eventTypes) {
        eventSource.addEventListener(eventType, handleMessage);
      }
      // Also handle unnamed events
      eventSource.onmessage = handleMessage;
      eventSource.onerror = () => {
        // SSE reconnects automatically
      };
    }

    connect();

    return () => {
      cancelled = true;
      eventSourceRef.current?.close();
      eventSourceRef.current = null;
    };
  }, [handleMessage]);

  return null; // No UI — just listens and shows toasts
}
