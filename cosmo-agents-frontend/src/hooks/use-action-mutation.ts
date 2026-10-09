'use client';

import {
  useMutation,
  useQueryClient,
  type QueryKey,
} from '@tanstack/react-query';
import { toast } from 'sonner';
import DailyActionsApi from '@/network/client/daily-actions';
import type { ApiResponse } from '@/models/response';
import type {
  DailyActionsBriefing,
  UpdateActionRequest,
  UpdateActionResponseData,
  ActionStatus,
} from '@/types/daily-actions';

interface ActionMutationVariables {
  actionId: string;
  request: UpdateActionRequest;
}

const transitionToStatus: Record<string, ActionStatus> = {
  mark_sent: 'completed',
  skip: 'skipped',
  snooze: 'snoozed',
  snooze_custom: 'snoozed',
  mark_completed: 'completed',
  reopen: 'suggested',
};

/**
 * `useDailyActions` caches the unwrapped briefing (its queryFn returns
 * `response.data`), so the cache entry is a `DailyActionsBriefing` — not the
 * `ApiResponse` envelope.
 */
type CachedBriefing = DailyActionsBriefing | undefined;

export function useActionMutation() {
  const queryClient = useQueryClient();

  return useMutation<
    ApiResponse<UpdateActionResponseData>,
    Error,
    ActionMutationVariables,
    { snapshots: [QueryKey, CachedBriefing][] }
  >({
    mutationFn: ({ actionId, request }) =>
      DailyActionsApi.updateAction(actionId, request),

    onMutate: async ({ actionId, request }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['daily-actions'] });

      // The real key is ['daily-actions', date, language, includeCompleted],
      // so several variants can be cached at once. Snapshot every one of them
      // and update each from its own data — writing a single object across the
      // whole prefix would copy one variant's briefing over all the others.
      const snapshots = queryClient.getQueriesData<DailyActionsBriefing>({
        queryKey: ['daily-actions'],
      });

      const newStatus = transitionToStatus[request.transition] || 'in_progress';

      queryClient.setQueriesData<DailyActionsBriefing>(
        { queryKey: ['daily-actions'] },
        (current) => {
          if (!current?.categories) return current;

          const updated = structuredClone(current);
          for (const category of updated.categories) {
            const action = category.actions.find((a) => a.id === actionId);
            if (action) {
              action.status = newStatus;
              action.status_changed_at = new Date().toISOString();
              if (request.snooze_until) {
                action.snooze_until = request.snooze_until;
              }
              break;
            }
          }
          return updated;
        }
      );

      return { snapshots };
    },

    onError: (err, variables, context) => {
      // Rollback each variant to the value it actually had
      for (const [queryKey, snapshot] of context?.snapshots ?? []) {
        queryClient.setQueryData(queryKey, snapshot);
      }
      console.error('Action mutation failed:', err, 'variables:', variables);
      toast.error(`Failed to update action: ${err.message}`);
    },

    onSettled: () => {
      // Always refetch to ensure consistency
      queryClient.invalidateQueries({ queryKey: ['daily-actions'] });
    },
  });
}
