'use client';

import { useRef } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import DailyActionsApi from '@/network/client/daily-actions';
import type { Language } from '@/network/client/outreach';
import type { GenerateRequest } from '@/types/daily-actions';


interface UseDailyActionsOptions {
  language?: Language;
  includeCompleted?: boolean;
  enabled?: boolean;
}

export function useDailyActions(options: UseDailyActionsOptions = {}) {
  const { language, includeCompleted, enabled = true } = options;
  const today = new Date().toISOString().split('T')[0];
  const failCountRef = useRef(0);

  const query = useQuery({
    queryKey: ['daily-actions', today, language, includeCompleted],
    queryFn: async () => {
      try {
        const response = await DailyActionsApi.getDailyActions({
          language,
          include_completed: includeCompleted,
        });
        failCountRef.current = 0;
        return response.data;
      } catch (err) {
        failCountRef.current += 1;
        if (failCountRef.current >= 2) {
          toast.error('Không thể tải dữ liệu pipeline', {
            action: {
              label: 'Thử lại',
              onClick: () => query.refetch(),
            },
          });
        }
        throw err;
      }
    },
    enabled,
    refetchInterval: 60_000,
    refetchIntervalInBackground: true,
    refetchOnWindowFocus: true,
    staleTime: 0,
    // Keep previously loaded data visible on poll errors
    placeholderData: (prev) => prev,
  });

  return query;
}

export function useDailyActionsGenerate() {
  const queryClient = useQueryClient();
  const today = new Date().toISOString().split('T')[0];

  return useMutation({
    mutationFn: async (options?: GenerateRequest) => {
      // Try agentic generation first
      try {
        const res = await fetch('/api/ai/daily-actions/generate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ language: options?.language ?? 'vi' }),
        });
        const data = await res.json();

        if (data.status === 'ready') {
          toast.success('AI briefing generated');
          return data;
        }

        // Fallback to rule-based
        console.log('[daily-actions] AI generation fallback:', data.reason);
      } catch (err) {
        console.log('[daily-actions] AI generation failed, falling back:', err);
      }

      // Rule-based fallback
      toast.info('Using standard briefing');
      return DailyActionsApi.generate(options);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['daily-actions', today],
      });
    },
  });
}
