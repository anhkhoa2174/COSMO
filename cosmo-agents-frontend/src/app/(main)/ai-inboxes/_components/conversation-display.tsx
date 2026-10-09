'use client';

import { IconRobot } from '@/assets/icons';
import { DeleteButton } from '@/components/buttons/delete-button';
import { MainButton } from '@/components/buttons/main-button';
import { PlateEditor } from '@/components/editor/plate-editor-lazy';
import { AIWriterV2 } from '@/components/ai-writer-v2';
import { CosmoMark } from '@/components/nav/cosmo-mark';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';
import { Agent } from '@/models/agent';
import type { Email } from '@/models/email';
import type { MemberSearchResponseData } from '@/models/organization';
import type { ConversationIntentCorrection } from '@/models/conversation';
import ConversationApi from '@/network/client/conversation';
import EmailApi, { regenerateAIReply } from '@/network/client/email';
import { useUser } from '@/hooks/use-user';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  Check,
  ChevronDown,
  CornerUpLeft,
  CornerUpRight,
  Reply,
  RotateCcw,
  Send,
  Sparkles,
  Trash2,
  Wand2,
  X,
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { toast } from 'sonner';
import { useConversation } from '../use-conversation';
import { getIntentStyle, IntentBadge } from './intent-badge';
import { IntentEditor } from './intent-editor';
import { GroupPicker } from './conversation-groups';

interface ConversationDisplayProps {
  agent: Agent;
  tab: string;
  members: MemberSearchResponseData[];
  assigneeId?: string;
}

export function ConversationDisplay({
  agent,
  tab,
  members,
  assigneeId,
}: ConversationDisplayProps) {
  const { user } = useUser();
  const queryClient = useQueryClient();
  const [conversation, setConversation] = useConversation();

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['conversations', conversation.selected?.id],
    queryFn: () =>
      EmailApi.search({
        filter: { conversation_id: conversation.selected?.id },
      }),
    enabled: !!conversation.selected?.id,
  });

  const [repliedEmail, setRepliedEmail] = useState<Email | null>(null);
  const [aiDraftDismissed, setAiDraftDismissed] = useState(false);
  // Set when a rep corrects the intent while a draft written for the old label
  // is still on screen. The draft is not touched — it may already carry their
  // edits — so this only warns and offers the rewrite.
  const [draftStale, setDraftStale] = useState(false);

  // Sentences the intent classifier quoted as evidence — highlighted in the
  // prospect's email body. Old conversations have no intent_detail; both the
  // phrase list and the plugin then stay empty and the body renders unchanged.
  const intentDetail = conversation.selected?.cmetadata?.intent_detail;
  const highlightPhrases = useMemo(
    () =>
      intentDetail?.reasoning
        ? extractQuotedPhrases(intentDetail.reasoning)
        : [],
    [intentDetail?.reasoning]
  );
  const highlightPlugin = useMemo(
    () =>
      highlightPhrases.length > 0
        ? rehypeHighlightPhrases(
            highlightPhrases,
            getIntentStyle(intentDetail?.intent).textClass
          )
        : null,
    [highlightPhrases, intentDetail?.intent]
  );

  /**
   * Every tab is a separate query, so refreshing only the active one leaves the
   * others stale — a conversation deleted from the inbox would not show up in
   * Trash until a full reload. Invalidating the prefix refreshes them all.
   */
  const refreshAllTabs = () =>
    queryClient.invalidateQueries({ queryKey: ['conversations', agent.id] });

  const runConversationAction = (
    action: () => Promise<unknown>,
    labels: { loading: string; success: string; error: string }
  ) => {
    toast.promise(action, {
      loading: labels.loading,
      success: () => {
        setConversation((prev) => ({ ...prev, selected: null }));
        refreshAllTabs();
        return labels.success;
      },
      // An empty message here used to swallow failures entirely — the toast
      // closed and the user assumed it worked.
      error: (err: any) => err?.error?.message || err?.message || labels.error,
    });
  };

  const handleDeleteConversation = async () => {
    runConversationAction(
      () => ConversationApi.delete(conversation.selected?.id as string),
      {
        loading: 'Moving to trash...',
        success: 'Conversation moved to trash',
        error: 'Could not move the conversation to trash',
      }
    );
  };

  const handleRestoreConversation = async () => {
    runConversationAction(
      () => ConversationApi.restore(conversation.selected?.id as string),
      {
        loading: 'Restoring...',
        success: 'Conversation restored to the inbox',
        error: 'Could not restore the conversation',
      }
    );
  };

  const handlePurgeConversation = async () => {
    runConversationAction(
      () => ConversationApi.purge(conversation.selected?.id as string),
      {
        loading: 'Deleting permanently...',
        success: 'Conversation permanently deleted',
        error: 'Could not delete the conversation',
      }
    );
  };

  useEffect(() => {
    if (repliedEmail) {
      const textEditor = document.getElementById('email-reply-box');
      if (textEditor) {
        textEditor.scrollIntoView({ behavior: 'smooth' });
      }
    }
  }, [repliedEmail]);

  useEffect(() => {
    setRepliedEmail(null);
    setAiDraftDismissed(false);
    setDraftStale(false);
  }, [conversation.selected]);

  /**
   * Mirrors the backend's correction into the selected conversation so the
   * thread, the list row and the highlight all agree immediately. The quoted
   * evidence is dropped along with the old label: those sentences argued for
   * the intent the rep just rejected, and re-colouring them under the new one
   * would pass the classifier's reasoning off as support for a conclusion it
   * never reached.
   */
  const handleIntentCorrected = (result: ConversationIntentCorrection) => {
    setConversation((prev) =>
      prev.selected
        ? {
            ...prev,
            selected: {
              ...prev.selected,
              intents: [result.intent] as typeof prev.selected.intents,
              cmetadata: {
                ...prev.selected.cmetadata,
                intent_detail: {
                  ...prev.selected.cmetadata?.intent_detail,
                  intent: result.intent,
                  corrected_by_user: true,
                  confidence: undefined,
                  reasoning: undefined,
                },
              },
            },
          }
        : prev
    );
    refetch();
    refreshAllTabs();
  };

  // if (isRefetching) {
  //   toast.loading('Refreshing thread...', {
  //     id: 'refresh-thread',
  //     description: 'Refreshing thread...',
  //   });
  // } else {
  //   toast.dismiss('refresh-thread');
  // }

  if (!conversation.selected) {
    return (
      <div className="flex h-full flex-1 items-center justify-center bg-zinc-100">
        <div className="flex flex-col items-center gap-4">
          <CosmoMark size={48} />
          <p className="text-sm text-muted-foreground">
            Enjoy your journey in Cosmo
          </p>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex h-full flex-1 items-center justify-center bg-zinc-100">
        <div className="flex flex-col items-center gap-4">
          <CosmoMark size={48} />
          <p className="text-sm text-muted-foreground">Email is loading...</p>
        </div>
      </div>
    );
  }

  // Narrowed once here: inside the email map the closure loses the guard above.
  const selectedId = conversation.selected.id;

  return (
    <div className="flex h-full flex-col">
      {/* The controls wrap under the subject rather than squeezing it: with
          intent, groups and assignee side by side, a long subject was pressed
          into a column one word wide. */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2">
        <p className="text-md min-w-[12rem] flex-1 truncate font-semibold">
          {data?.data.list[0].entity.subject}
        </p>
        <div className="ml-auto flex flex-wrap items-center gap-2">
          {/* The intent belongs to the conversation, not to any one email, and
              that is where the back end keeps it. Hanging the control off an
              email row meant it vanished whenever the classifier had written
              the label to the thread but not to the message — which is most of
              the older data. It also sat below the fold, so the badge a user
              actually looks at was not the badge they could change. */}
          {conversation.selected.intents?.[0] && tab !== 'trash' && (
            <span className="flex shrink-0 items-center gap-1.5 text-sm text-muted-foreground">
              Intent:
              <IntentEditor
                conversationId={selectedId}
                intent={conversation.selected.intents[0]}
                corrected={intentDetail?.corrected_by_user}
                size="md"
                onCorrected={handleIntentCorrected}
                onDraftStale={() => setDraftStale(true)}
              />
            </span>
          )}
          {tab !== 'trash' && (
            <GroupPicker
              conversationId={conversation.selected.id}
              groupIds={conversation.selected.group_ids ?? []}
              onChange={(conversationId, groupIds) =>
                setConversation((prev) => ({
                  ...prev,
                  selected:
                    prev.selected?.id === conversationId
                      ? { ...prev.selected, group_ids: groupIds }
                      : prev.selected,
                }))
              }
            />
          )}
          {tab !== 'trash' && user?.roles[0].name === 'admin' && (
            <>
              <AssignMember
                onSuccess={refreshAllTabs}
                conversationId={conversation.selected.id}
                members={members}
                selectedMemberId={assigneeId}
              />
              <DeleteButton
                variant="destructive-secondary"
                onConfirm={handleDeleteConversation}
                // The backend soft-deletes: the row keeps its data and lands in
                // Trash. Claiming it is permanent was simply untrue.
                description="This moves the conversation to Trash. You can restore it from there, or delete it permanently later."
              />
            </>
          )}

          {tab === 'trash' && user?.roles[0].name === 'admin' && (
            <>
              <MainButton
                variant="outline"
                text="Restore"
                icon={RotateCcw}
                onClick={handleRestoreConversation}
              />
              <DeleteButton
                variant="destructive-secondary"
                onConfirm={handlePurgeConversation}
                description="This cannot be undone. The conversation and its emails are removed from our servers for good."
              />
            </>
          )}
          <MainButton
            variant="ghost"
            onClick={() => refetch()}
            icon={RotateCcw}
            size="icon"
          />
        </div>
      </div>
      <Separator />
      <div className="h-96 flex-grow">
        <ScrollArea className="h-full bg-zinc-100">
          <div className="flex flex-col gap-4 p-4">
            {data?.data.list.map((email, index) => (
              <div
                key={`conversion-${index}`}
                className="flex items-start gap-2"
              >
                {email.entity.from_email === agent.email ? (
                  <span className="relative">
                    <Avatar>
                      <AvatarFallback className="bg-orange-50 font-semibold uppercase text-orange-500">
                        {email.entity.from_email.slice(0, 2)}
                      </AvatarFallback>
                    </Avatar>
                    <span className="absolute -bottom-1 -left-1">
                      <IconRobot width={20} height={20} />
                    </span>
                  </span>
                ) : (
                  <Avatar>
                    <AvatarFallback className="bg-blue-50 font-semibold uppercase text-blue-500">
                      {email.entity.from_email.slice(0, 2)}
                    </AvatarFallback>
                  </Avatar>
                )}
                <div
                  className={cn(
                    'relative mb-1 flex-1 rounded-md p-3',
                    email.entity.from_email !== agent.email
                      ? 'bg-[#EEFAFF]'
                      : 'bg-white'
                  )}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <div className="flex-1">
                        <p className="text-sm font-medium">
                          {email.entity.from_email}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          <span className="font-semibold">To:</span>{' '}
                          {email.entity.to_email}
                        </p>
                      </div>
                      {email.entity.intents.length > 0 && (
                        <span className="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
                          Intent:
                          {/* Read-only here. This shows what the classifier
                              recorded against this particular message; the
                              conversation's label, and the control that
                              changes it, live in the header. */}
                          <IntentBadge
                            intent={email.entity.intents[0]}
                            size="md"
                          />
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <p className="text-xs text-muted-foreground">
                        {format(new Date(email.entity.created_at), 'PPpp')}
                      </p>
                      <MainButton
                        variant="ghost"
                        size="icon"
                        icon={Reply}
                        onClick={() => setRepliedEmail(email.entity)}
                      />
                    </div>
                  </div>
                  <Markdown
                    remarkPlugins={[remarkGfm]}
                    rehypePlugins={
                      // Highlight the intent-bearing sentence only in the
                      // prospect email the classifier looked at (when it
                      // recorded an email_id, respect it).
                      highlightPlugin &&
                      email.entity.from_email !== agent.email &&
                      (!intentDetail?.email_id ||
                        intentDetail.email_id === email.entity.id)
                        ? [rehypeRaw, highlightPlugin]
                        : [rehypeRaw]
                    }
                  >
                    {email.entity.content}
                  </Markdown>
                </div>
              </div>
            ))}
            {!aiDraftDismissed &&
              conversation.selected?.cmetadata?.ai_reply?.draft_content &&
              data && (
                <AISuggestedReply
                  draft={conversation.selected.cmetadata.ai_reply}
                  emails={data.data.list.map((e) => e.entity)}
                  agentEmail={agent.email}
                  conversationId={conversation.selected.id}
                  campaignId={conversation.selected.campaign_id}
                  correctedIntent={
                    draftStale ? conversation.selected.intents[0] : undefined
                  }
                  onDismiss={() => setAiDraftDismissed(true)}
                  onSuccess={() => {
                    setAiDraftDismissed(true);
                    refetch();
                  }}
                />
              )}
            {repliedEmail && (
              <ReplyInput
                onClose={() => setRepliedEmail(null)}
                email={repliedEmail}
                onSuccess={() => {
                  setRepliedEmail(null);
                  refetch();
                }}
              />
            )}
          </div>
        </ScrollArea>
      </div>
      <Separator />
      <div className="flex justify-end p-2">
        <a
          href={`/campaigns/${conversation.selected.campaign_id}`}
          className="text-xs font-semibold text-blue-600 hover:underline"
        >
          Go to campaign
        </a>
      </div>
    </div>
  );
}

interface ReplyInputProps {
  onClose: () => void;
  email: Email;
  onSuccess?: () => void;
}

function ReplyInput({ onClose, email, onSuccess }: ReplyInputProps) {
  const [content, setContent] = useState('');
  const [sendFormat, setSendFormat] = useState<'reply' | 'forward'>('reply');

  const ReplyEmailMutation = useMutation({
    mutationFn: () => EmailApi.reply(email.id, { cc: [], content }),
    onSuccess: ({ data }) => {
      onSuccess?.();
      toast.success(data);
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    },
  });

  return (
    <div
      className="overflow-hidden rounded-lg border bg-white"
      id="email-reply-box"
    >
      <div className="flex items-center gap-2 border-b px-3 py-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm text-muted-foreground hover:bg-muted">
              {sendFormat === 'reply' ? (
                <CornerUpLeft className="h-4 w-4" />
              ) : (
                <CornerUpRight className="h-4 w-4" />
              )}
              <ChevronDown className="h-3 w-3" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem onClick={() => setSendFormat('reply')}>
              <CornerUpLeft className="mr-2 h-4 w-4" />
              Reply
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setSendFormat('forward')}>
              <CornerUpRight className="mr-2 h-4 w-4" />
              Forward
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <span className="text-sm text-muted-foreground">
          {email.from_email}
        </span>
      </div>
      <PlateEditor
        value={content}
        onChange={setContent}
        readOnly={ReplyEmailMutation.isPending}
      />
      <div className="flex items-center justify-between border-t px-3 py-2">
        <MainButton
          size="sm"
          loading={ReplyEmailMutation.isPending}
          onClick={() => ReplyEmailMutation.mutate()}
          text="Send"
          rightIcon={Send}
        />
        <MainButton
          variant="destructive-secondary"
          size="icon"
          icon={Trash2}
          onClick={onClose}
          disabled={ReplyEmailMutation.isPending}
        />
      </div>
    </div>
  );
}

interface AISuggestedReplyProps {
  draft: {
    draft_content: string;
    intent: string;
    generated_by: string;
    generated_at: string;
  };
  emails: Email[];
  agentEmail: string;
  conversationId: string;
  campaignId: string;
  /**
   * The label a person picked after this draft was generated, when it differs
   * from the one the draft was written for. Present only to warn: the draft is
   * never rewritten on its own, because it may already hold the rep's edits.
   */
  correctedIntent?: string;
  onDismiss: () => void;
  onSuccess: () => void;
}

function AISuggestedReply({
  draft,
  emails,
  agentEmail,
  conversationId,
  campaignId,
  correctedIntent,
  onDismiss,
  onSuccess,
}: AISuggestedReplyProps) {
  const { user } = useUser();
  const [isEditing, setIsEditing] = useState(false);
  const [assistantOpen, setAssistantOpen] = useState(false);
  const [editedContent, setEditedContent] = useState(draft.draft_content);

  // Editing and the assistant are the same mode: the campaign builder pairs the
  // editor with the AI Writer chat, and this panel mirrors it.
  const openAssistant = () => {
    setIsEditing(true);
    setAssistantOpen(true);
  };

  const latestProspectEmail = [...emails]
    .reverse()
    .find((e) => e.from_email !== agentEmail);

  // Regenerating is offered, never automatic. The draft on screen may already
  // carry the representative's own edits, so replacing it is their decision and
  // the confirmation names what would be lost.
  const regenerateMutation = useMutation({
    mutationFn: () => regenerateAIReply(conversationId),
    onSuccess: (reply) => {
      setEditedContent(reply.content);
      setIsEditing(true);
      toast.success(`Draft rewritten for "${correctedIntent}"`);
    },
    onError: (err: any) => {
      toast.error(
        err?.error?.message || err?.message || 'Could not rewrite the draft'
      );
    },
  });

  const approveMutation = useMutation({
    mutationFn: () => {
      if (!latestProspectEmail)
        throw new Error('No prospect email to reply to');
      return EmailApi.reply(latestProspectEmail.id, {
        cc: [],
        content: isEditing ? editedContent : draft.draft_content,
      });
    },
    onSuccess: ({ data }) => {
      onSuccess();
      toast.success(typeof data === 'string' ? data : 'Reply sent');
    },
    onError: (err: any) => {
      toast.error(err.error?.message || err.message || 'Failed to send reply');
    },
  });

  return (
    <div className="overflow-hidden rounded-xl border border-indigo-200 bg-gradient-to-br from-indigo-50 via-white to-blue-50 shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-indigo-100 bg-gradient-to-r from-indigo-500 to-blue-500 px-4 py-2.5">
        <div className="flex items-center gap-2 text-white">
          <Sparkles className="h-4 w-4" />
          <span className="text-sm font-semibold">AI Suggested Reply</span>
          {draft.intent && <IntentBadge intent={draft.intent} size="md" />}
        </div>
        <button
          onClick={onDismiss}
          className="rounded-md p-1 text-white/70 transition-colors hover:bg-white/10 hover:text-white"
          aria-label="Dismiss"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* The intent changed under this draft. Say so and offer the rewrite —
          overwriting the text automatically would discard any edits the rep
          has already made to it. */}
      {correctedIntent && (
        <div className="flex flex-wrap items-center gap-2 border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-800">
          <span>
            Written for <span className="font-medium">{draft.intent}</span>. You
            changed the intent to{' '}
            <span className="font-medium">{correctedIntent}</span>, so this
            draft may no longer fit.
          </span>
          <span className="ml-auto flex items-center gap-1.5">
            <button
              type="button"
              disabled={regenerateMutation.isPending}
              onClick={() => {
                const edited =
                  isEditing && editedContent !== draft.draft_content;
                if (
                  edited &&
                  !window.confirm(
                    'Rewriting replaces the draft, including the edits you have ' +
                      'made to it. Continue?'
                  )
                )
                  return;
                regenerateMutation.mutate();
              }}
              className="rounded-md border border-amber-300 bg-white px-2 py-1 font-medium text-amber-800 transition-colors hover:bg-amber-100 disabled:opacity-60"
            >
              {regenerateMutation.isPending ? 'Rewriting…' : 'Rewrite draft'}
            </button>
            {/* The chat is the other way to fix a draft: describe the change
                rather than start over. Kept separate because the button above
                promises a rewrite and this one only opens an editor. */}
            <button
              type="button"
              onClick={openAssistant}
              className="rounded-md px-2 py-1 font-medium text-amber-800 underline-offset-2 hover:underline"
            >
              Edit with AI
            </button>
          </span>
        </div>
      )}

      {/* Reply context */}
      {latestProspectEmail && (
        <div className="flex items-center gap-2 border-b border-indigo-100 bg-indigo-50/50 px-4 py-2 text-xs text-indigo-600">
          <CornerUpLeft className="h-3 w-3" />
          <span>
            Replying to{' '}
            <span className="font-medium">
              {latestProspectEmail.from_email}
            </span>
          </span>
        </div>
      )}

      {/* Content — editor on the left, AI Writer chat on the right, the same
          split the campaign builder uses to refine a draft. */}
      <div className="flex gap-0 p-4">
        <div className="min-w-0 flex-1">
          {isEditing ? (
            <PlateEditor
              value={editedContent}
              onChange={setEditedContent}
              readOnly={approveMutation.isPending}
            />
          ) : (
            <div className="rounded-lg border border-gray-100 bg-white p-3 text-sm text-gray-700">
              <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeRaw]}>
                {draft.draft_content}
              </Markdown>
            </div>
          )}
        </div>

        {assistantOpen && (
          <div className="ml-4 w-80 flex-shrink-0 border-l border-indigo-100 pl-4">
            <AIWriterV2
              isAIReply
              conversationId={conversationId}
              campaignId={campaignId}
              organizationId={user?.organizations?.[0]?.id}
              // The corrected label, when there is one. Passing draft.intent
              // here meant the rewrite offered after a correction was composed
              // for the very label the rep had just rejected — the banner
              // above says the draft no longer fits, then the rewrite makes
              // the same mistake again.
              intentType={(correctedIntent || draft.intent) as any}
              template={{
                content: editedContent,
                type: 'email',
                subject: '',
                send_after: 0,
                knowledges: [],
                id: '',
              }}
              onUpdateTemplate={(key, value) => {
                if (key === 'content') setEditedContent(value as string);
              }}
              onClearConversation={() => setEditedContent(draft.draft_content)}
              onFinished={(_subject: string, content: string) => {
                if (content) setEditedContent(content);
              }}
            />
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex flex-wrap items-center gap-2 border-t border-indigo-100 bg-indigo-50/30 px-4 py-2.5">
        <MainButton
          size="sm"
          loading={approveMutation.isPending}
          disabled={!latestProspectEmail}
          onClick={() => approveMutation.mutate()}
          text={isEditing ? 'Send Edited Reply' : 'Send'}
          rightIcon={Send}
        />
        <MainButton
          variant="outline"
          size="sm"
          onClick={() => {
            if (isEditing) {
              setEditedContent(draft.draft_content);
              setAssistantOpen(false);
            }
            setIsEditing(!isEditing);
          }}
          text={isEditing ? 'Cancel Edit' : 'Edit Draft'}
          disabled={approveMutation.isPending}
        />
        <MainButton
          variant={assistantOpen ? 'secondary' : 'outline'}
          size="sm"
          icon={Wand2}
          onClick={() =>
            assistantOpen ? setAssistantOpen(false) : openAssistant()
          }
          text={assistantOpen ? 'Hide AI Writer' : 'AI Writer'}
          disabled={approveMutation.isPending}
        />
        {!isEditing && (
          <MainButton
            variant="ghost"
            size="sm"
            onClick={onDismiss}
            text="Dismiss"
            disabled={approveMutation.isPending}
          />
        )}
      </div>
    </div>
  );
}

/**
 * Pulls the phrases the intent classifier quoted in its reasoning, e.g.
 * `The prospect said "would love to meet the team" ...`. Straight and curly
 * quotes, single and double. Short fragments (< 4 words) are dropped — they
 * are usually quoted words, not the intent-bearing sentence.
 */
function extractQuotedPhrases(reasoning: string): string[] {
  const phrases: string[] = [];
  const pattern = /"([^"]+)"|'([^']+)'|“([^”]+)”|‘([^’]+)’/g;
  let match: RegExpExecArray | null;
  while ((match = pattern.exec(reasoning)) !== null) {
    const phrase = (match[1] || match[2] || match[3] || match[4] || '').trim();
    if (phrase.split(/\s+/).length >= 4 && !phrases.includes(phrase)) {
      phrases.push(phrase);
    }
  }
  return phrases;
}

const HIGHLIGHT_SKIP_TAGS = new Set(['code', 'pre', 'script', 'style']);

/**
 * Rehype plugin factory: wraps case-insensitive verbatim occurrences of the
 * given phrases (within single text nodes, so raw-HTML bodies stay safe) in a
 * bold, intent-colored span with a transparent background. Phrases that never
 * appear in the body simply match nothing — the output is then identical to
 * the input.
 */
function rehypeHighlightPhrases(phrases: string[], textClass: string) {
  const classNames = [
    'font-semibold',
    'bg-transparent',
    ...textClass.split(' '),
  ];
  const lowered = phrases.map((p) => p.toLowerCase());

  const splitTextNode = (value: string) => {
    const lowerValue = value.toLowerCase();
    const parts: any[] = [];
    let pos = 0;
    while (pos < value.length) {
      let best = -1;
      let bestLen = 0;
      for (const phrase of lowered) {
        const idx = lowerValue.indexOf(phrase, pos);
        if (
          idx !== -1 &&
          (best === -1 ||
            idx < best ||
            (idx === best && phrase.length > bestLen))
        ) {
          best = idx;
          bestLen = phrase.length;
        }
      }
      if (best === -1) break;
      if (best > pos)
        parts.push({ type: 'text', value: value.slice(pos, best) });
      parts.push({
        type: 'element',
        tagName: 'span',
        properties: { className: classNames },
        children: [{ type: 'text', value: value.slice(best, best + bestLen) }],
      });
      pos = best + bestLen;
    }
    if (parts.length === 0) return null;
    if (pos < value.length)
      parts.push({ type: 'text', value: value.slice(pos) });
    return parts;
  };

  const visit = (node: any) => {
    if (!node?.children || HIGHLIGHT_SKIP_TAGS.has(node.tagName)) return;
    const next: any[] = [];
    for (const child of node.children) {
      if (child.type === 'text' && typeof child.value === 'string') {
        const parts = splitTextNode(child.value);
        next.push(...(parts ?? [child]));
      } else {
        visit(child);
        next.push(child);
      }
    }
    node.children = next;
  };

  return () => (tree: any) => {
    try {
      visit(tree);
    } catch {
      // Highlighting is decoration — never let it break the email body.
    }
  };
}

interface AssignMemberProps {
  conversationId: string;
  selectedMemberId?: string;
  members: MemberSearchResponseData[];
  onSuccess?: () => void;
}

function AssignMember({
  conversationId,
  selectedMemberId,
  members,
  onSuccess,
}: AssignMemberProps) {
  const [open, setOpen] = useState(false);

  const handleSaveAssignee = (assignee_id: string) => {
    setOpen(false);
    const myPromise = async () => {
      return await ConversationApi.assign(conversationId, {
        conversation_type: 'assign_to_human',
        assignee_id,
      });
    };

    toast.promise(myPromise, {
      loading: 'Assigning...',
      success: () => {
        onSuccess?.();
        return `Conversation has been assigned`;
      },
      error: 'Failed to assign conversation',
    });
  };

  const selectedMember = members.find(
    (member) => member.entity.id === selectedMemberId
  );

  return (
    <div className="flex items-center space-x-4">
      <p className="text-sm text-muted-foreground">Assignee:</p>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <MainButton
            variant="outline"
            text={selectedMember?.entity.name || '+ Set assignee'}
          />
        </PopoverTrigger>
        <PopoverContent className="p-0" side="bottom" align="end">
          <Command defaultValue={selectedMemberId}>
            <CommandInput placeholder="Search assignee..." />
            <CommandList>
              <CommandEmpty>No results found.</CommandEmpty>
              <CommandGroup>
                {members.map((member) => (
                  <CommandItem
                    key={member.entity.id}
                    value={member.entity.id}
                    onSelect={handleSaveAssignee}
                    keywords={[member.entity.name]}
                  >
                    {member.entity.name}
                    <Check
                      className={cn(
                        'ml-auto',
                        member.entity.id === selectedMemberId
                          ? 'opacity-100'
                          : 'opacity-0'
                      )}
                    />
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}
