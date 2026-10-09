'use client';

import { useState } from 'react';
import _ from 'lodash';
import { formatDistanceToNow } from 'date-fns';
import { useQuery, useQueryClient } from '@tanstack/react-query';

import { cn } from '@/lib/utils';
import ConversationApi from '@/network/client/conversation';
import { useWatchGmailQuery } from '@/network/client/google';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable';

import { useConversation } from '../use-conversation';
import { ConversationDisplay } from './conversation-display';
import { GroupChip, useConversationGroups } from './conversation-groups';
import { IntentBadge } from './intent-badge';
import { Agent } from '@/models/agent';
import OrganizationApi from '@/network/client/organization';
import { RotateCcw, Sparkles } from 'lucide-react';
import { MainButton } from '@/components/buttons/main-button';
import { DeleteButton } from '@/components/buttons/delete-button';
import { toast } from 'sonner';
import { markdownToPlainText } from '@/helpers';
import {
  countActiveFilters,
  EMPTY_FILTERS,
  InboxFilters as FilterBar,
  type InboxFilters,
} from './inbox-filters';
import { Button } from '@/components/ui/button';

interface ConversationListProps {
  agent: Agent;
  tab: string;
}

const fetchConversations = async (
  agentId: string,
  tab: string,
  filters: InboxFilters
) => {
  const payload: { filter: Record<string, unknown> } = {
    filter: { is_deleted: false },
  };
  const params: Record<string, string> = {};

  // Only send the keys that are actually set. An empty string or a null still
  // reaches the server as a filter and would narrow the results to nothing.
  if (filters.q.trim()) payload.filter.q = filters.q.trim();
  if (filters.intent) payload.filter.intent = filters.intent;
  if (filters.hasDraft) payload.filter.has_draft = true;
  if (filters.sinceDays) payload.filter.since_days = filters.sinceDays;
  if (filters.groupId) payload.filter.group_id = filters.groupId;

  if (tab === 'sent') {
    params.conversation_type = 'sent';
  }
  if (tab === 'assign_to_ai') {
    params.conversation_type = 'assign_to_ai';
  }
  if (tab === 'assign_to_human') {
    params.conversation_type = 'assign_to_human';
  }
  if (tab === 'trash') {
    payload.filter.is_deleted = true;
  }
  return ConversationApi.search(agentId, payload as never, params);
};

export function ConversationList({ agent, tab }: ConversationListProps) {
  useWatchGmailQuery();

  const [conversation, setConversation] = useConversation();
  const [filters, setFilters] = useState<InboxFilters>(EMPTY_FILTERS);

  const { data, isLoading, refetch } = useQuery({
    // Filters belong in the key: without them a search reuses the cached
    // unfiltered page and the list appears not to react.
    queryKey: ['conversations', agent.id, tab, filters],
    queryFn: () => fetchConversations(agent.id, tab, filters),
    enabled: !!agent.id && !!tab,
    placeholderData: (prev) => prev,
  });

  const { data: _members } = useQuery({
    queryKey: ['members', agent.id],
    queryFn: () => OrganizationApi.searchMember('me', { filter: {} }),
    enabled: !!agent.id,
  });

  const conversations = data?.data.list || [];
  const { data: groups = [] } = useConversationGroups();
  const groupsById = new Map(groups.map((g) => [g.id, g]));
  const members = _members?.data.list || [];

  const assigneeId = conversations.find(
    (c) => c.entity.id === conversation.selected?.id
  )?.entity.assignee_id;

  // if (isRefetching) {
  //   toast.loading('Refreshing inbox...', {
  //     id: 'refresh-inbox',
  //     description: 'Refreshing inbox...',
  //   });
  // } else {
  //   toast.dismiss('refresh-inbox');
  // }

  const queryClient = useQueryClient();

  const handleEmptyTrash = async () => {
    toast.promise(() => ConversationApi.emptyTrash(), {
      loading: 'Emptying trash...',
      success: (res: any) => {
        setConversation((prev) => ({ ...prev, selected: null }));
        queryClient.invalidateQueries({
          queryKey: ['conversations', agent.id],
        });
        return typeof res?.data === 'string' ? res.data : 'Trash emptied';
      },
      error: (err: any) =>
        err?.error?.message || err?.message || 'Could not empty the trash',
    });
  };

  return (
    <ResizablePanelGroup direction="horizontal" className="rounded-lg border">
      <ResizablePanel defaultSize={35} minSize={35}>
        <div className="flex h-10 items-center justify-between border-b bg-background px-4 shadow-sm">
          <div className="font-semibold">
            {tab === 'trash' ? 'Trash' : 'Inbox'}
          </div>
          <div className="flex items-center gap-1">
            {tab === 'trash' && conversations.length > 0 && (
              <DeleteButton
                variant="ghost"
                size="sm"
                text="Empty trash"
                title="Empty trash"
                description={`This cannot be undone. All ${conversations.length} conversation(s) in the trash will be removed from our servers for good.`}
                onConfirm={handleEmptyTrash}
              />
            )}
            <MainButton
              aria-label="Refresh"
              variant="ghost"
              onClick={() => refetch()}
              icon={RotateCcw}
              size="icon"
            />
          </div>
        </div>

        <FilterBar value={filters} onChange={setFilters} />

        <ScrollArea className="h-[calc(100vh-13.5em)]">
          <div className="flex flex-col gap-2 p-4">
            {isLoading ? (
              <div className="flex flex-col space-y-2">
                <Skeleton className="h-[150px] rounded-lg" />
                <Skeleton className="h-[150px] rounded-lg" />
                <Skeleton className="h-[150px] rounded-lg" />
              </div>
            ) : (
              <>
                {conversations.length === 0 &&
                  // Say which of the two it is. "Inbox is empty" under an
                  // active search reads as data loss rather than no match.
                  (filters.q.trim() || countActiveFilters(filters) > 0 ? (
                    <div className="space-y-2 p-6 text-center">
                      <p className="font-medium">No conversation matches</p>
                      <p className="text-sm text-muted-foreground">
                        Try a different term, or clear the filters.
                      </p>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setFilters(EMPTY_FILTERS)}
                      >
                        Clear search and filters
                      </Button>
                    </div>
                  ) : (
                    <div className="p-4 text-center font-medium">
                      AI inbox is empty
                    </div>
                  ))}
                {_.orderBy(
                  conversations,
                  ['latest_email.updated_at'],
                  ['desc']
                ).map((c) => {
                  const { latest_email: item } = c;
                  const isRead = c.entity.status === 'read';
                  const hasAiDraft =
                    !!c.entity.cmetadata?.ai_reply?.draft_content;
                  return (
                    <button
                      key={c.entity.id}
                      className={cn(
                        'flex flex-col items-start gap-2 rounded-lg p-3 text-left text-sm transition-all hover:bg-zinc-100',
                        !isRead && 'border-l-2 border-l-blue-500 bg-blue-50/30',
                        conversation.selected?.id === c.entity.id &&
                          'bg-zinc-100'
                      )}
                      onClick={() =>
                        setConversation((prev) => ({
                          ...prev,
                          selected: c.entity,
                        }))
                      }
                    >
                      <div className="flex w-full flex-col gap-2">
                        <div className="flex items-center gap-2">
                          <div
                            className={cn(
                              'line-clamp-1',
                              isRead
                                ? 'font-medium text-muted-foreground'
                                : 'font-semibold text-foreground'
                            )}
                          >
                            {item.from_email}
                          </div>
                          <div
                            className={cn(
                              'ml-auto shrink-0 text-xs',
                              conversation.selected?.id === c.entity.id
                                ? 'text-foreground'
                                : 'text-muted-foreground'
                            )}
                          >
                            {formatDistanceToNow(new Date(item.updated_at), {
                              addSuffix: true,
                            })}
                          </div>
                        </div>
                        <div className="line-clamp-1 font-medium">
                          {item.subject}
                        </div>
                      </div>
                      <Separator />
                      <div
                        className={cn(
                          'line-clamp-2',
                          isRead && 'text-muted-foreground'
                        )}
                      >
                        {markdownToPlainText(item.content.substring(0, 300))}
                      </div>
                      <div className="flex flex-wrap items-center gap-2">
                        {(c.entity.group_ids ?? []).map((id) => {
                          const group = groupsById.get(id);
                          return group ? (
                            <GroupChip key={id} group={group} />
                          ) : null;
                        })}
                        {hasAiDraft && (
                          <span className="inline-flex items-center gap-1 rounded-full bg-indigo-50 px-2 py-0.5 text-xs text-indigo-600">
                            <Sparkles className="h-3 w-3" />
                            AI Draft
                          </span>
                        )}
                        {item.intents.map((label) => (
                          <IntentBadge key={label} intent={label} size="sm" />
                        ))}
                      </div>
                    </button>
                  );
                })}
              </>
            )}
          </div>
        </ScrollArea>
      </ResizablePanel>
      <ResizableHandle />
      <ResizablePanel defaultSize={65} minSize={35}>
        <ConversationDisplay
          agent={agent}
          tab={tab}
          members={members}
          assigneeId={assigneeId}
        />
      </ResizablePanel>
    </ResizablePanelGroup>
  );
}
