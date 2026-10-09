'use client';

import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import Cookies from 'js-cookie';
import DailyActionsApi from '@/network/client/daily-actions';
import type { DailyActionSSEEvent } from '@/types/daily-actions';

type SSEEventHandler = (event: DailyActionSSEEvent) => void;

interface UseDailyEventsOptions {
  enabled?: boolean;
  onEvent?: SSEEventHandler;
}

/**
 * The stream writer always emits `event: <type>`, so every message is a named
 * event and `onmessage` — which only fires for unnamed ones — never runs.
 * Each type has to be subscribed explicitly.
 */
const SSE_EVENT_TYPES = [
  'generation_complete',
  'action_updated',
  'prospect_replied',
  'meeting_approaching',
  'followup_due',
  'snooze_expired',
] as const;

/** Ignore repeated reconnect failures for this long before refetching again. */
const ERROR_REFETCH_COOLDOWN_MS = 30_000;

export function useDailyEvents(options: UseDailyEventsOptions = {}) {
  const { enabled = true, onEvent } = options;
  const queryClient = useQueryClient();
  const eventSourceRef = useRef<EventSource | null>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;
  const lastErrorRefetchRef = useRef(0);

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const data: DailyActionSSEEvent = JSON.parse(event.data);

        // Notify consumer
        onEventRef.current?.(data);

        // Auto-invalidate queries based on event type
        const today = new Date().toISOString().split('T')[0];
        switch (data.event_type) {
          case 'generation_complete':
          case 'action_updated':
          case 'prospect_replied':
          case 'meeting_approaching':
          case 'followup_due':
          case 'snooze_expired':
            queryClient.invalidateQueries({
              queryKey: ['daily-actions', today],
            });
            break;
        }
      } catch {
        // Ignore malformed events
      }
    },
    [queryClient]
  );

  useEffect(() => {
    if (!enabled) return;

    // The stream is registered before auth middleware so it can bypass Fiber's
    // response buffering; the handler reads the JWT from the query string
    // instead. Without the token every connection is rejected.
    const token = Cookies.get('access_token');
    if (!token) return;

    const eventSource = DailyActionsApi.openEventStream(token);
    eventSourceRef.current = eventSource;

    for (const eventType of SSE_EVENT_TYPES) {
      eventSource.addEventListener(eventType, handleMessage);
    }

    eventSource.onerror = () => {
      // EventSource retries on its own every 3s (the server sends
      // `retry: 3000`). Refetching on each attempt would turn a dropped
      // connection into a request storm, so it is rate-limited.
      const now = Date.now();
      if (now - lastErrorRefetchRef.current < ERROR_REFETCH_COOLDOWN_MS) return;
      lastErrorRefetchRef.current = now;

      const today = new Date().toISOString().split('T')[0];
      queryClient.invalidateQueries({ queryKey: ['daily-actions', today] });
    };

    return () => {
      for (const eventType of SSE_EVENT_TYPES) {
        eventSource.removeEventListener(eventType, handleMessage);
      }
      eventSource.close();
      eventSourceRef.current = null;
    };
  }, [enabled, handleMessage, queryClient]);
}
