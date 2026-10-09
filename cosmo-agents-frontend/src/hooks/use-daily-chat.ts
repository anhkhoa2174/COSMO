'use client';

import { useChat } from 'ai/react';
import { toast } from 'sonner';

export function useDailyChat() {
  return useChat({
    id: 'daily-actions',
    api: '/api/ai/daily-actions',
    onError: (err) => {
      console.error('[useDailyChat] Error:', err);
      const msg = err.message || 'Chat request failed';
      if (msg.includes('credit balance') || msg.includes('too low')) {
        toast.error('Anthropic API credit exhausted. Please top up at console.anthropic.com');
      } else {
        toast.error(msg);
      }
    },
    onResponse: (res) => {
      if (!res.ok) {
        console.error('[useDailyChat] Response error:', res.status, res.statusText);
      }
    },
  });
}
