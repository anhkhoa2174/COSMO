'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useAtom } from 'jotai';
import { format } from 'date-fns';
import { Loader2, RefreshCw, Zap } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { MessagePartRenderer } from '@/components/daily-actions/message-part-renderer';
import { ChatInputBar } from '@/components/daily-actions/chat-input-bar';
import { TopBar } from '@/components/daily-actions/top-bar';
import { CosmoMark } from '@/components/nav/cosmo-mark';
import { SkeletonBriefing } from '@/components/daily-actions/skeleton-briefing';
import { QuickCommandChips } from '@/components/daily-actions/quick-command-chips';
import { useDailyActions, useDailyActionsGenerate } from '@/hooks/use-daily-actions';
import { useDailyChat } from '@/hooks/use-daily-chat';
import { useDailyEvents } from '@/hooks/use-daily-events';
import { ContactDetailPanel } from '@/components/daily-actions/contact-detail-panel';
import { ToolUsageIndicator } from '@/components/daily-actions/tool-usage-indicator';
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts';
import { languageAtom, activePanelAtom } from '@/stores/daily-actions';
import type {
  BDChatMessage,
  MessagePart,
  DailyActionsBriefing,
  DailyActionSSEEvent,
} from '@/types/daily-actions';
import { sortCategoriesByPriority } from '@/types/daily-actions';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function parseChatMessageParts(msg: any): MessagePart[] {
  const parts: MessagePart[] = [];

  // Check tool invocations for pipeline summary data
  if (msg.toolInvocations?.length) {
    for (const inv of msg.toolInvocations) {
      if (inv.state === 'result' && inv.result) {
        const result = inv.result;
        // Detect pipeline summary shape
        if (
          typeof result === 'object' &&
          result !== null &&
          'total_active_contacts' in result &&
          'contacts_by_stage' in result
        ) {
          parts.push({
            type: 'pipeline-summary',
            pipeline_summary: result as import('@/types/daily-actions').PipelineSummary,
          });
        }
      }
    }
  }

  // Only show text if no structured parts were found
  if (parts.length === 0 && msg.content) {
    parts.push({ type: 'text', text: msg.content });
  }

  return parts;
}

function getMessageTypeLabel(type: string): string {
  switch (type) {
    case 'briefing':
      return 'Daily Briefing';
    case 'alert':
      return 'Alert';
    case 'update':
      return 'Update';
    default:
      return 'Response';
  }
}

function briefingToMessage(briefing: DailyActionsBriefing): BDChatMessage {
  // Check if any action has agentic strategic reasoning
  const hasAgenticContent = briefing.categories?.some((cat) =>
    cat.actions?.some((a) => a.outreach_data?.strategic_reasoning)
  );

  const parts: MessagePart[] = [];

  // Fallback notice if using rule-based generation
  if (!hasAgenticContent && briefing.categories?.some((c) => c.actions?.length > 0)) {
    parts.push({
      type: 'text',
      text: '📋 *Standard briefing* — AI strategic recommendations are being set up. Actions below use rule-based suggestions.',
    });
  }

  parts.push(
    { type: 'text', text: briefing.agent_briefing.greeting },
    {
      type: 'agent-reasoning',
      text: briefing.agent_briefing.strategic_reasoning,
      memory_refs: briefing.agent_briefing.memory_references,
    },
    {
      type: 'category-badges',
      badges: briefing.agent_briefing.category_counts,
    },
    ...sortCategoriesByPriority(briefing.categories).map((cat) => ({
      type: 'action-category' as const,
      category: cat,
    })),
  );

  return {
    id: `briefing-${briefing.generation_id}`,
    role: 'assistant',
    parts,
    message_type: 'briefing',
    created_at: briefing.generated_at || new Date().toISOString(),
  };
}

function sseEventToMessage(event: DailyActionSSEEvent): BDChatMessage | null {
  switch (event.event_type) {
    case 'prospect_replied':
      return {
        id: `alert-${event.event_id}`,
        role: 'assistant',
        parts: [
          {
            type: 'alert',
            alert: {
              alert_type: 'prospect_replied',
              contact: event.contact,
              content: event.reply_preview,
              agent_reasoning: event.agent_reasoning,
              recommended_action: event.recommended_action,
              action_id: event.action_id,
              timestamp: event.reply_timestamp,
            },
          },
        ],
        message_type: 'alert',
        created_at: event.timestamp,
      };
    case 'meeting_approaching':
      return {
        id: `alert-${event.event_id}`,
        role: 'assistant',
        parts: [
          {
            type: 'alert',
            alert: {
              alert_type: 'meeting_approaching',
              contact: event.contact,
              content: `${event.meeting_title} — in ${event.hours_until} hours`,
              agent_reasoning: event.agent_message,
              recommended_action: event.has_prep
                ? 'Review your meeting prep briefing'
                : 'Prepare talking points before the meeting',
              action_id: event.action_id,
              timestamp: event.timestamp,
            },
          },
        ],
        message_type: 'alert',
        created_at: event.timestamp,
      };
    case 'followup_due':
      return {
        id: `alert-${event.event_id}`,
        role: 'assistant',
        parts: [
          {
            type: 'alert',
            alert: {
              alert_type: 'prospect_replied',
              contact: event.contact,
              content: `Follow-up #${event.followup_number} due — ${event.days_since} days since last contact${event.is_final ? ' (FINAL)' : ''}`,
              agent_reasoning: event.agent_message,
              recommended_action: event.is_final
                ? 'Send final follow-up or mark as dropped'
                : 'Send follow-up to maintain engagement',
              action_id: event.action_id,
              timestamp: event.timestamp,
            },
          },
        ],
        message_type: 'alert',
        created_at: event.timestamp,
      };
    case 'snooze_expired':
      return {
        id: `alert-${event.event_id}`,
        role: 'assistant',
        parts: [
          {
            type: 'text',
            text: event.agent_message,
          },
          ...event.actions
            .filter(
              (a) =>
                a.type === 'respond' ||
                a.type === 'outreach' ||
                a.type === 'followup'
            )
            .slice(0, 5)
            .map((action) => ({
              type: 'action-category' as const,
              category: {
                id: 'followup' as const,
                label: 'Snoozed actions — time to revisit',
                icon: 'Clock',
                color: 'yellow',
                description: 'These actions were snoozed earlier today.',
                actions: [action],
                total_count: 1,
                has_more: false,
              },
            })),
        ],
        message_type: 'update',
        created_at: event.timestamp,
      };
    default:
      return null;
  }
}

/**
 * Full-height chrome shared by the briefing's loading / error / empty states so
 * they sit inside the same shell as the chat itself instead of floating.
 */
function StateShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-dvh flex-col overflow-hidden bg-muted/40">
      <TopBar />
      <div className="flex min-h-0 flex-1 items-center justify-center p-6">
        {children}
      </div>
    </div>
  );
}

export function DailyActionsChat() {
  const [language, setLanguage] = useAtom(languageAtom);
  const {
    data: briefing,
    isLoading,
    error,
    refetch,
  } = useDailyActions({ language });
  const { mutate: generate, isPending: isGenerating } =
    useDailyActionsGenerate();
  const {
    messages: chatMessages,
    input: chatInput,
    handleInputChange,
    handleSubmit: handleChatSubmit,
    isLoading: isChatLoading,
    append,
    data: chatStreamData,
  } = useDailyChat();

  // Build a map of messageId → tools used (FR-031)
  // useChat data[] accumulates stream annotations across all responses.
  // We track which new annotations arrived while each message was streaming.
  const [messageToolsMap, setMessageToolsMap] = useState<Record<string, string[]>>({});
  const prevDataLengthRef = useRef(0);
  const prevLoadingRef = useRef(false);

  useEffect(() => {
    const isNowLoading = isChatLoading;
    const wasLoading = prevLoadingRef.current;

    if (wasLoading && !isNowLoading) {
      // Streaming just finished — collect tools used since last message
      const allData = (chatStreamData ?? []) as Array<{ toolsUsed?: string[] }>;
      const newEntries = allData.slice(prevDataLengthRef.current);
      const toolsUsed = newEntries.flatMap((d) => d.toolsUsed ?? []);

      if (toolsUsed.length > 0) {
        const lastMsg = chatMessages[chatMessages.length - 1];
        if (lastMsg && lastMsg.role === 'assistant') {
          setMessageToolsMap((prev) => ({ ...prev, [lastMsg.id]: toolsUsed }));
        }
      }
      prevDataLengthRef.current = allData.length;
    }
    prevLoadingRef.current = isNowLoading;
  }, [isChatLoading, chatStreamData, chatMessages]);
  const [alertMessages, setAlertMessages] = useState<BDChatMessage[]>([]);
  const [, setActivePanel] = useAtom(activePanelAtom);
  const scrollRef = useRef<HTMLDivElement>(null);
  const chatInputRef = useRef<HTMLInputElement>(null);

  // FR-049: Keyboard shortcuts
  useKeyboardShortcuts({
    onCommandPalette: () => {
      chatInputRef.current?.focus();
      chatInputRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
    },
    onEscape: () => {
      setActivePanel(null);
    },
  });

  // Handle SSE events — convert to alert messages in the chat flow
  const handleSSEEvent = useCallback((event: DailyActionSSEEvent) => {
    const message = sseEventToMessage(event);
    if (message) {
      setAlertMessages((prev) => [...prev, message]);
    }
  }, []);

  useDailyEvents({
    enabled: !!briefing && briefing.generation_status === 'ready',
    onEvent: handleSSEEvent,
  });

  // Check if pipeline is empty (no actions at all)
  const isEmptyPipeline = useMemo(() => {
    if (!briefing || briefing.generation_status !== 'ready') return false;
    return (
      briefing.categories.length === 0 ||
      briefing.categories.every((c) => c.total_count === 0)
    );
  }, [briefing]);

  // Combine all messages: briefing + alerts + conversational
  const allMessages = useMemo(() => {
    const result: BDChatMessage[] = [];

    // 1. Briefing message (or empty pipeline message)
    if (briefing && briefing.generation_status === 'ready') {
      if (isEmptyPipeline) {
        result.push({
          id: 'empty-pipeline',
          role: 'assistant',
          parts: [
            {
              type: 'text',
              text:
                language === 'vi'
                  ? 'Pipeline hiện tại không có contacts nào cần action. Hãy import contacts hoặc bắt đầu một campaign mới để BD Agent có thể hỗ trợ bạn.'
                  : 'Your pipeline is currently empty — no contacts need action today. Import contacts or start a new campaign to get started with BD Agent.',
            },
          ],
          message_type: 'conversational',
          created_at: new Date().toISOString(),
        });
      } else {
        result.push(briefingToMessage(briefing));
      }
    }

    // 2. Alert messages from SSE events (sorted by time)
    result.push(...alertMessages);

    // 3. Conversational messages from useChat
    for (const msg of chatMessages) {
      const parts: MessagePart[] = parseChatMessageParts(msg);
      result.push({
        id: msg.id,
        role: msg.role as 'user' | 'assistant',
        parts,
        message_type: msg.role === 'user' ? 'user_message' : 'conversational',
        created_at: msg.createdAt?.toISOString() || new Date().toISOString(),
      });
    }

    return result;
  }, [briefing, alertMessages, chatMessages]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [allMessages.length]);

  const handleSendMessage = (message: string) => {
    append({ role: 'user', content: message });
  };

  // Loading state — skeleton shimmer
  if (isLoading) {
    return (
      <div className="flex h-dvh flex-col overflow-hidden bg-muted/40">
        <TopBar />
        <div className="cosmo-thin-scrollbar min-h-0 flex-1 overflow-y-auto">
          <div className="mx-auto max-w-[860px] px-7 pt-6 pb-5">
            <SkeletonBriefing />
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <StateShell>
        <div className="flex max-w-md flex-col items-center gap-4 rounded-2xl border border-dashed bg-card px-6 py-12 text-center">
          <p className="text-sm text-muted-foreground">
            Failed to load daily actions
          </p>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Retry
          </Button>
        </div>
      </StateShell>
    );
  }

  // Not generated state
  if (briefing?.generation_status === 'not_generated') {
    const pipelineCount = briefing?.pipeline_summary?.total_active_contacts;

    return (
      <StateShell>
        <div className="flex w-full max-w-md flex-col items-center gap-4 rounded-2xl border border-dashed bg-card px-6 py-12 text-center">
          <span className="grid size-14 place-items-center rounded-2xl bg-white shadow-sm ring-1 ring-border">
            <CosmoMark size={30} id="daily-actions-empty" />
          </span>
          <div>
            <p className="text-lg font-semibold text-foreground">
              No briefing for today yet
            </p>
            <p className="mt-1.5 text-[0.9rem] leading-relaxed text-muted-foreground">
              BD Agent analyses your pipeline and proposes who to contact today
              and why.
            </p>
          </div>
          <Button
            disabled={isGenerating}
            onClick={() => generate({ language, force_refresh: true })}
          >
            {isGenerating ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="mr-2 h-4 w-4" />
            )}
            {isGenerating ? 'Starting...' : 'Generate Briefing'}
          </Button>
          {typeof pipelineCount === 'number' && pipelineCount > 0 && (
            <p className="text-xs text-muted-foreground/80">
              {pipelineCount} active {pipelineCount === 1 ? 'contact' : 'contacts'}{' '}
              in your pipeline
            </p>
          )}
        </div>
      </StateShell>
    );
  }

  // Generating state (started = enqueued, generating = worker running)
  if (
    briefing?.generation_status === 'started' ||
    briefing?.generation_status === 'generating'
  ) {
    return (
      <StateShell>
        <div className="flex max-w-md flex-col items-center gap-4 rounded-2xl border border-dashed bg-card px-6 py-12 text-center">
          <Loader2 className="h-8 w-8 animate-spin text-violet-600 dark:text-violet-400" />
          <p className="text-sm text-muted-foreground">
            Generating your daily briefing...
          </p>
        </div>
      </StateShell>
    );
  }

  return (
    <div className="flex h-dvh flex-col overflow-hidden bg-muted/40">
      {/* Top bar */}
      <TopBar pipelineSummary={briefing?.pipeline_summary} />

      {/* Chat area */}
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {/* Chat header with progress pill + language toggle */}
        <div className="flex items-center justify-between border-b border-border bg-card px-5 py-3 sm:px-7 sm:py-4">
          <div className="flex items-center gap-2">
            {briefing?.progress && (
              <span className="rounded-[20px] border border-emerald-500/[0.2] bg-emerald-500/[0.1] px-3.5 py-1 text-[0.8rem] font-semibold text-emerald-600 dark:text-emerald-400">
                {briefing.progress.completed} / {briefing.progress.total}{' '}
                completed
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              className="h-8 border-border bg-transparent text-xs text-muted-foreground hover:bg-muted"
              disabled={isGenerating}
              onClick={() => generate({ language, force_refresh: true })}
              title="Regenerate briefing (force refresh)"
            >
              {isGenerating ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <RefreshCw className="h-3.5 w-3.5" />
              )}
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-8 border-border bg-transparent text-xs text-muted-foreground hover:bg-muted"
              onClick={() =>
                setLanguage((lang) => (lang === 'vi' ? 'en' : 'vi'))
              }
            >
              {language === 'vi' ? 'VI' : 'EN'}
            </Button>
          </div>
        </div>

        {/* Messages */}
        <div className="cosmo-thin-scrollbar min-h-0 flex-1 overflow-y-auto">
          <div className="mx-auto max-w-[860px] space-y-5 px-7 pt-6 pb-5">
            {allMessages.map((message) => (
              <div key={message.id} className="space-y-4">
                {/* User message bubble */}
                {message.role === 'user' && (
                  <div className="flex justify-end">
                    <div
                      className="max-w-[75%] bg-indigo-600 text-white text-sm leading-relaxed"
                      style={{
                        padding: '10px 16px',
                        borderRadius: '14px 14px 4px 14px',
                      }}
                    >
                      {message.parts[0]?.type === 'text' &&
                        message.parts[0].text}
                      <div className="mt-1 text-[0.65rem] font-mono opacity-40">
                        {format(new Date(message.created_at), 'HH:mm')}
                      </div>
                    </div>
                  </div>
                )}

                {/* Assistant message — agent wrapper */}
                {message.role === 'assistant' && (
                  <div>
                    <div className="mb-2 flex items-center gap-2 text-[0.65rem] font-semibold uppercase tracking-[0.4px] text-muted-foreground/80">
                      <div className="flex h-[34px] w-[34px] items-center justify-center rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 text-white">
                        <Zap className="h-4 w-4" />
                      </div>
                      BD Agent &middot;{' '}
                      {format(new Date(message.created_at), 'HH:mm')}{' '}
                      &middot; {getMessageTypeLabel(message.message_type)}
                    </div>
                    <div className="space-y-4">
                      {message.parts.map((part, i) => (
                        <MessagePartRenderer
                          key={`${message.id}-${i}`}
                          part={part}
                          messageId={message.id}
                        />
                      ))}
                    </div>
                    {/* FR-031: tool usage transparency */}
                    {messageToolsMap[message.id] && (
                      <ToolUsageIndicator toolsUsed={messageToolsMap[message.id]} />
                    )}
                  </div>
                )}
              </div>
            ))}

            {/* Quick command chips — shown after briefing, before any chat messages */}
            {briefing && briefing.generation_status === 'ready' && chatMessages.length === 0 && (
              <QuickCommandChips
                onSendCommand={handleSendMessage}
                disabled={isChatLoading}
              />
            )}

            {/* Typing indicator */}
            {isChatLoading && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin text-indigo-600 dark:text-indigo-400" />
                <span>BD Agent is thinking...</span>
              </div>
            )}

            <div ref={scrollRef} />
          </div>
        </div>

        {/* Chat input */}
        <ChatInputBar ref={chatInputRef} onSend={handleSendMessage} isLoading={isChatLoading} />
      </div>

      {/* Contact detail panel (slide-over) */}
      <ContactDetailPanel />
    </div>
  );
}
