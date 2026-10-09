'use client';

import { useCampaign } from '@/app/(main)/campaigns/[campaign_id]/use-campaign';
import { MainButton } from '@/components/buttons/main-button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import {
  cleanText,
  formatAndCapitalize,
  generateId,
  replaceTag,
} from '@/helpers';
import { useTypingEffect } from '@/hooks/use-typing-effect';
import { cn } from '@/lib/utils';
import { EmailIntent } from '@/models/email';
import chatApi, {
  getMessage,
  getMessageError,
  Message,
} from '@/network/client/ai-chat';
import CampaignApi from '@/network/client/campaign';
import { getCustomFields } from '@/network/client/custom-field';
import { KnowledgeApiV2 } from '@/network/client/knowledge';
import { GetTemplateData } from '@/network/client/template';
import { useMutation, useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import _ from 'lodash';
import {
  BookOpenText,
  CheckCircle2,
  Paperclip,
  RotateCcw,
  SendHorizonal,
} from 'lucide-react';
import {
  PropsWithChildren,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { toast } from 'sonner';
import { useUser } from '@/hooks/use-user';
import { CosmoMark } from './nav/cosmo-mark';
import { ChatMessageList } from './chat/ChatMessageList';
import CodeDisplayBlock from './chat/CodeBlock';
import FileDropzone from './file-dropzone';
import { Button } from './ui/button';
import { Skeleton } from './ui/skeleton';
import { Textarea } from './ui/textarea';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';

/**
 * Openers for an empty thread. Each one is something this writer genuinely
 * does to a campaign email: draft from scratch, tighten, add a CTA, reshape
 * the tone. Clicking one fills the composer rather than sending, so the user
 * can adapt it before it costs a round trip.
 */
const STARTER_PROMPTS = [
  'Write a first outreach email for cold leads',
  'Make this shorter and more direct',
  'Add a clear call to action at the end',
  'Rewrite it in a warmer, more personal tone',
] as const;

const suggestionAction = [
  {
    label: 'Improve writing',
    value: 'improveWriting',
  },
  {
    label: 'Emojify',
    value: 'emojify',
  },
  {
    label: 'Make longer',
    value: 'makeLonger',
  },
  {
    label: 'Make shorter',
    value: 'makeShorter',
  },
  {
    label: 'Fix spelling & grammar',
    value: 'fixSpelling',
  },
  {
    label: 'Simplify language',
    value: 'simplifyLanguage',
  },
];

export function AIWriterV2({
  intentType,
  campaignType,
  conversationId,
  onClearConversation,
  template,
  onUpdateTemplate,
  isAIReply,
  knowledgeReply,
  onFinished,
  setLoadingReply,
  onChangeKnowledge,
  onLoadingChat,
  campaignId,
  campaignName,
  organizationId,
}: {
  conversationId?: string;
  campaignType?: string;
  /** Supplied by callers outside the campaign builder (e.g. the AI Inbox),
   *  where the campaign atom is not populated. */
  campaignId?: string;
  campaignName?: string;
  organizationId?: string;
  intentType?: EmailIntent;
  onClearConversation?: () => void;
  template?: GetTemplateData;
  onUpdateTemplate?: (key: keyof GetTemplateData, value: any) => void;
  isAIReply?: boolean;
  knowledgeReply?: Message[];
  onFinished?: (subject: string, content: string) => void;
  setLoadingReply?: (loading: boolean) => void;
  onChangeKnowledge?: (values: string[]) => void;
  onLoadingChat?: React.Dispatch<React.SetStateAction<boolean>>;
}) {
  const [campaign] = useCampaign();
  const { user } = useUser();
  const messagesRef = useRef<HTMLDivElement>(null);
  // The composer text lives here so the empty-state starter prompts can drop
  // a prompt straight into the input instead of sending it blind.
  const composerRef = useRef<HTMLTextAreaElement>(null);
  const [composerText, setComposerText] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [isTyping, setIsTyping] = useState(false);
  const [conversationLoading, setConversationLoading] = useState(true);
  const [textStream, setTextStream] = useState('');
  const [emailContent, setEmailContent] = useState('');
  const [emailSubject, setEmailSubject] = useState('');
  const [isSuggestion, setIsSuggestion] = useState(false);

  const { data: customFields } = useQuery({
    queryKey: ['customFields'],
    queryFn: getCustomFields,
  });

  // Falls back to the campaign atom so the campaign builder keeps working
  // unchanged, but callers can pass the ids explicitly instead.
  const effectiveCampaignId = campaignId || campaign.id;
  const effectiveOrganizationId = organizationId || campaign.organization_id;
  const effectiveCampaignName = campaignName || campaign.name;

  const { data: mergeTags } = useQuery({
    // Keyed by campaign — merge tags differ per campaign, and a bare
    // 'mergeTags' key served the first campaign's tags to every other one.
    queryKey: ['mergeTags', effectiveCampaignId],
    queryFn: () => CampaignApi.mergeTags(effectiveCampaignId),
    enabled: Boolean(effectiveCampaignId),
  });

  const saleRepOptions = useMemo(() => {
    const mergeTagsData = mergeTags?.data as any;
    return (
      mergeTagsData?.sale_rep?.map((item: string) => ({
        id: generateId(),
        group: formatAndCapitalize('sale_rep'),
        label: formatAndCapitalize(item, 'sale_rep_'),
        value: item,
      })) || []
    );
  }, [mergeTags?.data]);

  const foundOrganization = user?.organizations.find(
    (item) => item.id === effectiveOrganizationId
  ) as any;
  // todo add more entity_type
  const customTags = useMemo(() => {
    return [
      ...tags,
      ...saleRepOptions,
      ...(customFields?.data?.list?.map((field) => ({
        id: field.id,
        group: _.capitalize(field.entity_type),
        label: field.name,
        value:
          field?.entity_type === 'contact'
            ? 'contact_' + field.normalized_name
            : field.normalized_name,
      })) || []),
    ].sort((a, b) => (a.group || '').localeCompare(b?.group || ''));
  }, [customFields?.data?.list, tags, saleRepOptions]);

  // Qualify the label with its group. On its own a chip reading "First name"
  // or "Name" does not say whose — the contact's, the sender's, or the
  // organisation's — and the same bare label appears under several groups.
  const tagsOptions = Object.fromEntries(
    customTags.map((tag) => [
      tag.value,
      tag.group ? `${tag.group} ${tag.label.toLowerCase()}` : tag.label,
    ])
  );

  useTypingEffect({
    text: emailSubject || '',
    isActive: true,
    setText: (value) => value && onUpdateTemplate?.('subject', value),
    speed: 50,
    step: 3,
    dependencies: [],
  });

  useTypingEffect({
    text: emailContent || '',
    isActive: true,
    setText: (value) => value && onUpdateTemplate?.('content', value),
    speed: 50,
    step: 10,
    dependencies: [],
  });

  const newChat = async (newUserMessages: Message[]) => {
    onLoadingChat?.(true);
    setIsTyping(true);
    setIsSuggestion(false);
    setLoadingReply?.(true);
    setTextStream('');
    const templateContent = template?.content || '';
    const templateSubject = template?.subject || '';
    const customFieldsData = [
      ...(customFields?.data?.list?.map((item) => item.normalized_name) || []),
      ...saleRepOptions.map((item) => item.value),
    ];
    const organization = {
      company_description: foundOrganization?.company_description || '',
      company_targeting_persona:
        foundOrganization?.company_targeting_persona || '',
      company_url: foundOrganization?.company_url || '',
      name: foundOrganization?.name || '',
    };
    const knowledge = `**Context**: \n\n
    organization: \n\n
     - name: ${organization.name}\n\n
     - description: ${organization.company_description} / ${Array.isArray(organization?.company_targeting_persona) ? organization.company_targeting_persona.join(', ') : organization?.company_targeting_persona || ''} \n\n
     - company_url: ${organization.company_url}\n\n
    ${customFieldsData.length > 0 ? `merge_tags: ${customFieldsData.join(', ')}` : ''}\n\n
    ${campaignType ? `campaign_type: ${campaignType}` : ''}\n\n
    ${intentType ? `intent_type: ${intentType}` : ''}\n\n
    Subject: ${templateSubject}\n\n
    Content: ${templateContent}`;
    if (!conversationId) {
      return;
    }
    try {
      const { subject, content } = await chatApi.newChat(
        conversationId,
        [
          ...(isAIReply
            ? knowledgeReply || []
            : [
                {
                  content: knowledge,
                  content_type: 'text',
                  role: 'user',
                },
              ]),
          ...newUserMessages,
        ],
        (answer, lastItem) => {
          let i = 0;
          const intervalId = setInterval(() => {
            if (i < answer.length) {
              setTextStream(answer.substring(0, i + 1));
              i++;
            } else {
              onLoadingChat?.(false);
              // Done chunk
              clearInterval(intervalId);
              const cleanedSubject = cleanText(subject);
              const cleanedContent = cleanText(content);
              const isChangeSubject = cleanedSubject !== template?.subject;
              const isChangeContent = cleanedContent !== template?.content;
              if (isChangeSubject && isChangeContent) {
                setEmailSubject?.(cleanedSubject);
                setEmailContent?.(cleanedContent);
              } else if (isChangeSubject) {
                setEmailSubject?.(cleanedSubject);
              } else if (isChangeContent) {
                setEmailContent?.(cleanedContent);
              }
              // Presentation only: this reply is typed straight into the
              // template, so remember what it touched and let the thread say
              // so under the bubble. There is nothing for the user to click.
              const wroteInto = isChangeSubject
                ? isChangeContent
                  ? 'both'
                  : 'subject'
                : isChangeContent
                  ? 'content'
                  : null;
              if (wroteInto) {
                lastItem.meta_data = {
                  ...(lastItem.meta_data || {}),
                  wrote_into_draft: wroteInto,
                };
              }
              lastItem.content = answer;
              onFinished?.(cleanedSubject, cleanedContent);
              setMessages((prev) => [...prev, lastItem]);
              setIsTyping(false);
            }
          }, 10);
        }
      );
    } catch (error) {
      console.error('Error during chat:', error);
      const errorMessage = getMessageError(error);
      setMessages((prev) => [...prev, errorMessage]);
      setIsTyping(false);
      onLoadingChat?.(false);
    }
  };

  const handleSendInstruction = async (text: string) => {
    const message = getMessage({
      content: text,
      content_type: 'text',
      role: 'user',
    });
    if (text.trim()) {
      setMessages((prev) => [...prev, message]);
      const cloneMessage = { ...message };
      cloneMessage.content = text + '\n\n **Action**: add to email!';
      await newChat([cloneMessage]);
    }
  };

  const handleSuggestionAction = async (action: string) => {
    setIsSuggestion(false);
    const message = getMessage({
      content: action,
      content_type: 'text',
      role: 'user',
    });
    setMessages((prev) => [...prev, message]);
    await newChat([message]);
  };

  const handleStarterPrompt = (prompt: string) => {
    setComposerText(prompt);
    composerRef.current?.focus();
  };

  const handleClearConversation = async () => {
    setIsSuggestion(false);
    setConversationLoading(true);
    setIsTyping(false);
    setTextStream('');
    setMessages([]);
    setComposerText('');
    onClearConversation?.();
  };

  useEffect(() => {
    (async () => {
      if (!conversationId) return;
      try {
        setIsSuggestion(false);
        setConversationLoading(true);
        const data = await chatApi.getConversation(conversationId);
        setMessages(data || []);
      } finally {
        setConversationLoading(false);
        setIsSuggestion(true);
      }
    })();
  }, [conversationId]);

  useEffect(() => {
    if (messagesRef.current) {
      messagesRef.current.scrollTo({
        top: messagesRef.current.scrollHeight,
        behavior: 'smooth',
      });
    }
  }, [messages]);

  const renderContent = useCallback(
    (message: Message) => {
      return message.content.split('```').map((part: string, index: number) => {
        if (index % 2 === 0) {
          return (
            <Markdown
              key={index}
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw]}
            >
              {replaceTag(part, tagsOptions)}
            </Markdown>
          );
        } else {
          return (
            <pre className="whitespace-pre-wrap pt-2" key={index}>
              <CodeDisplayBlock code={part} lang="" />
            </pre>
          );
        }
      });
    },
    [tagsOptions]
  );

  // The bot opens every fresh conversation with a greeting, so "no messages"
  // is almost never true. What actually means "the user hasn't started" is
  // "no user turn yet" — that is when the starter prompts earn their space.
  const showStarters =
    !conversationLoading &&
    !isTyping &&
    !messages.some((m) => m.role === 'user');

  const quickEdits = isSuggestion ? (
    <div className="flex flex-wrap gap-1.5">
      {suggestionAction.map((action) => (
        <button
          key={action.value}
          type="button"
          onClick={() => handleSuggestionAction(action.label)}
          className="rounded-full border bg-background px-2.5 py-1 text-[0.7rem] font-medium text-muted-foreground transition-colors hover:border-violet-300 hover:bg-violet-500/10 hover:text-violet-700 dark:hover:border-violet-700 dark:hover:text-violet-300"
        >
          {action.label}
        </button>
      ))}
    </div>
  ) : null;

  return (
    <div
      role="complementary"
      aria-label="AI Writer"
      className="flex h-full min-h-0 flex-col border-l bg-card"
    >
      {/* Header — same family as the Ask COSMO widget, one notch lighter
          because this panel is narrow. */}
      <div className="flex shrink-0 items-start gap-2.5 border-b bg-background px-3 py-2.5">
        <span className="mt-px grid size-8 shrink-0 place-items-center rounded-lg border border-border bg-card shadow-sm">
          <CosmoMark size={22} id="ai-writer-header" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold leading-tight">AI Writer</p>
          {/* Which campaign this draft belongs to. Opened from a node the
              panel filled the screen with no reminder of what it was writing
              for, and the merge tags and knowledge base both come from this
              campaign. */}
          {effectiveCampaignName && (
            <p className="mt-0.5 truncate text-[0.7rem] font-medium leading-snug text-foreground/80">
              {effectiveCampaignName}
            </p>
          )}
          <p className="mt-0.5 text-[0.7rem] leading-snug text-muted-foreground">
            Describe the email you want. It writes into the draft on the left,
            grounded in your knowledge base.
          </p>
        </div>
        <Tooltip>
          <TooltipTrigger
            aria-label="Clear conversation"
            className="grid size-7 shrink-0 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            onClick={handleClearConversation}
          >
            <RotateCcw className="size-4" />
          </TooltipTrigger>
          <TooltipContent side="bottom" align="end" className="z-[51]">
            Clear conversation
          </TooltipContent>
        </Tooltip>
      </div>

      {/* Thread */}
      <div className="min-h-0 flex-1 px-3 py-3">
        <ChatMessageList smooth ref={messagesRef}>
          {messages.map((m, idx) => (
            <WriterMessage
              key={m.id || idx}
              index={idx}
              fromUser={m.role === 'user'}
              wroteIntoDraft={
                m.meta_data?.wrote_into_draft as string | undefined
              }
            >
              {renderContent(m)}
            </WriterMessage>
          ))}

          {showStarters ? (
            <div className="mb-3 flex flex-col gap-3 px-0.5">
              {messages.length === 0 && (
                <div className="flex flex-col items-center gap-2 pt-2 text-center">
                  <span className="grid size-11 place-items-center rounded-xl border border-border bg-card shadow-sm">
                    <CosmoMark size={28} id="ai-writer-empty" />
                  </span>
                  <p className="text-sm font-semibold text-foreground">
                    What should this email say?
                  </p>
                  <p className="text-xs leading-relaxed text-muted-foreground">
                    Ask in plain language. Every reply is typed straight into
                    the subject and body on the left.
                  </p>
                </div>
              )}
              <div className="flex flex-col gap-1.5">
                <p className="text-[0.68rem] font-semibold uppercase tracking-wide text-muted-foreground">
                  Try one of these
                </p>
                {STARTER_PROMPTS.map((prompt) => (
                  <button
                    key={prompt}
                    type="button"
                    onClick={() => handleStarterPrompt(prompt)}
                    className="rounded-xl border bg-background px-3 py-2 text-left text-xs font-medium text-muted-foreground transition-colors hover:border-violet-300 hover:bg-violet-500/10 hover:text-violet-700 dark:hover:border-violet-700 dark:hover:text-violet-300"
                  >
                    {prompt}
                  </button>
                ))}
              </div>
              {quickEdits && (
                <div className="flex flex-col gap-1.5">
                  <p className="text-[0.68rem] font-semibold uppercase tracking-wide text-muted-foreground">
                    Quick edits
                  </p>
                  {quickEdits}
                </div>
              )}
            </div>
          ) : (
            quickEdits && <div className="mb-3 pl-8">{quickEdits}</div>
          )}

          {conversationLoading && (
            <div className="flex flex-col gap-2 py-2">
              <Skeleton className="h-10 w-4/5 rounded-2xl" />
              <Skeleton className="ml-auto h-8 w-1/2 rounded-2xl" />
            </div>
          )}

          {isTyping && (
            <div className="mb-3 flex gap-2">
              <span className="mt-0.5 grid size-6 shrink-0 place-items-center rounded-md border border-border bg-card shadow-sm">
                <CosmoMark size={16} id="ai-writer-typing" />
              </span>
              <div className="min-w-0 flex-1">
                <div className="mb-1 flex items-center gap-1.5 text-[0.68rem] font-semibold uppercase tracking-wide text-violet-600 dark:text-violet-400">
                  <span className="flex gap-0.5">
                    <span className="size-1 animate-bounce rounded-full bg-violet-500 [animation-delay:-0.3s] dark:bg-violet-400" />
                    <span className="size-1 animate-bounce rounded-full bg-violet-500 [animation-delay:-0.15s] dark:bg-violet-400" />
                    <span className="size-1 animate-bounce rounded-full bg-violet-500 dark:bg-violet-400" />
                  </span>
                  Writing…
                </div>
                {textStream && (
                  <div className="whitespace-pre-wrap break-words rounded-2xl rounded-tl-sm border border-border bg-muted/60 px-3 py-2 text-sm leading-relaxed text-foreground">
                    {textStream}
                  </div>
                )}
              </div>
            </div>
          )}
        </ChatMessageList>
      </div>

      {/* Composer */}
      <div className="shrink-0 border-t bg-background px-3 py-2.5">
        <MentionInput
          customTags={customTags}
          onSubmit={handleSendInstruction}
          onChangeKnowledge={onChangeKnowledge}
          text={composerText}
          setText={setComposerText}
          inputRef={composerRef}
        />
      </div>
    </div>
  );
}

/**
 * One turn in the thread. The user sits right on a solid violet bubble, the
 * assistant sits left behind the COSMO mark, so the two are never confused at
 * this width.
 *
 * There is no "insert into email" action to surface: replies are typed into
 * the template as they arrive (see `useTypingEffect` above), so a finished
 * reply that changed the draft carries a confirmation line instead.
 */
function WriterMessage({
  children,
  fromUser,
  index,
  wroteIntoDraft,
}: {
  children: React.ReactNode;
  fromUser: boolean;
  index: number;
  wroteIntoDraft?: string;
}) {
  if (fromUser) {
    return (
      <div className="mb-3 flex justify-end">
        <div className="max-w-[88%] break-words rounded-2xl rounded-br-sm bg-violet-600 px-3 py-2 text-sm leading-relaxed text-white shadow-sm [&_p]:my-0">
          {children}
        </div>
      </div>
    );
  }

  return (
    <div className="mb-3 flex gap-2">
      <span className="mt-0.5 grid size-6 shrink-0 place-items-center rounded-md border border-border bg-card shadow-sm">
        <CosmoMark size={16} id={`ai-writer-msg-${index}`} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="mb-1 text-[0.68rem] font-semibold uppercase tracking-wide text-muted-foreground">
          AI Writer
        </p>
        <div className="break-words rounded-2xl rounded-tl-sm border border-border bg-muted/60 px-3 py-2 text-sm leading-relaxed text-foreground [&_a]:text-violet-600 [&_a]:underline [&_h1]:text-sm [&_h1]:font-semibold [&_h2]:text-sm [&_h2]:font-semibold [&_h3]:text-sm [&_h3]:font-semibold [&_li]:my-0.5 [&_ol]:my-1 [&_ol]:list-decimal [&_ol]:pl-4 [&_p]:my-1 [&_strong]:font-semibold [&_ul]:my-1 [&_ul]:list-disc [&_ul]:pl-4">
          {children}
        </div>
        {wroteIntoDraft && (
          <div className="mt-1.5 flex items-center gap-1.5 text-[0.7rem] font-medium text-emerald-600 dark:text-emerald-400">
            <CheckCircle2 className="size-3.5 shrink-0" />
            {wroteIntoDraft === 'subject'
              ? 'Subject written into the draft'
              : wroteIntoDraft === 'content'
                ? 'Body written into the draft'
                : 'Subject and body written into the draft'}
          </div>
        )}
      </div>
    </div>
  );
}

export interface Tag {
  id: string;
  group: string;
  label: string;
  value: string;
}

export const tags: Tag[] = [
  {
    id: '1',
    group: 'Contact',
    label: 'First name',
    value: 'contact_first_name',
  },
  { id: '2', group: 'Contact', label: 'Last name', value: 'contact_last_name' },
  { id: '3', group: 'Contact', label: 'Email', value: 'contact_email' },
  { id: '4', group: 'Contact', label: 'Company', value: 'contact_company' },
  { id: '5', group: 'Contact', label: 'Job title', value: 'contact_job_title' },
  { id: '6', group: 'Contact', label: 'Address', value: 'contact_address' },
  { id: '7', group: 'Contact', label: 'City', value: 'contact_city' },
  { id: '8', group: 'Contact', label: 'Country', value: 'contact_country' },
  { id: '9', group: 'Contact', label: 'Zip', value: 'contact_zip' },
  {
    id: '19',
    group: 'Organization',
    label: 'Name',
    value: 'organization_name',
  },
  {
    id: '11',
    group: 'Organization',
    label: 'Company url',
    value: 'organization_company_url',
  },
  { id: '12', group: 'Sender', label: 'Email', value: 'sender_email' },
  { id: '13', group: 'Sender', label: 'Name', value: 'sender_name' },
  { id: '14', group: 'Agent', label: 'Signature', value: 'agent_signature' },
];

function insertMention(
  text: string,
  cursorPosition: number,
  mention: string
): [string, number] {
  const beforeCursor = text.slice(0, cursorPosition);
  const afterCursor = text.slice(cursorPosition);
  const lastAtSymbol = beforeCursor.lastIndexOf('@');
  const newText = `${beforeCursor.slice(0, lastAtSymbol) + mention} ${afterCursor}`;
  const newCursorPosition = lastAtSymbol + mention.length + 1;
  return [newText, newCursorPosition];
}

function MentionInput({
  customTags = tags,
  onSubmit,
  onChangeKnowledge,
  text,
  setText,
  inputRef,
}: {
  customTags: Tag[];
  onSubmit?: (text: string) => Promise<void>;
  onChangeKnowledge?: (values: string[]) => void;
  /** Owned by the panel so starter prompts can prefill the composer. */
  text: string;
  setText: (value: string) => void;
  inputRef: React.RefObject<HTMLTextAreaElement>;
}) {
  const [activeSuggestionIndex, setActiveSuggestionIndex] = useState(0);
  const [cursorPosition, setCursorPosition] = useState(0);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [suggestions, setSuggestions] = useState<Tag[]>([]);
  const [countRefresh, setCountRefresh] = useState(0);
  // How many knowledge documents the writer is actually grounded in. Comes
  // from the bot-info query the knowledge dialog already runs on mount, so
  // showing it costs no extra request.
  const [knowledgeCount, setKnowledgeCount] = useState<number | null>(null);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (showSuggestions) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveSuggestionIndex((prevIndex) =>
          prevIndex === suggestions.length - 1 ? 0 : prevIndex + 1
        );
        document
          .getElementById(`suggestion-${activeSuggestionIndex}`)
          ?.scrollIntoView({
            block: 'center',
          });
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveSuggestionIndex((prevIndex) =>
          prevIndex === 0 ? suggestions.length - 1 : prevIndex - 1
        );
        document
          .getElementById(`suggestion-${activeSuggestionIndex}`)
          ?.scrollIntoView({
            block: 'center',
          });
      } else if (e.key === 'Enter') {
        e.preventDefault();
        handleMentionClick(suggestions[activeSuggestionIndex]);
      }
    }
    if (e.key === 'Enter' && !e.shiftKey && !showSuggestions) {
      e.preventDefault();
      handleSubmit();
    } else if (e.key === 'Enter' && e.shiftKey) {
      // Allow line break
      return;
    }
  };

  useEffect(() => {
    if (showSuggestions) {
      setActiveSuggestionIndex(0);
    }
  }, [showSuggestions, suggestions]);

  const handleOnChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const newText = e.target.value;
    const newCursorPosition = e.target.selectionStart || 0;
    setText(newText);
    setCursorPosition(newCursorPosition);

    const beforeCursor = newText.slice(0, newCursorPosition);
    const lastAtSymbol = beforeCursor.lastIndexOf('@');

    if (lastAtSymbol !== -1) {
      const searchString = beforeCursor.slice(lastAtSymbol + 1);

      if (searchString.includes(' ')) {
        setShowSuggestions(false);
        return;
      }

      // Check if the searchString ends with a space
      if (searchString.endsWith(' ')) {
        setShowSuggestions(false);
        return;
      }

      const filtered = customTags.filter((user) =>
        user.label
          .toLowerCase()
          .replace(/\s/g, '')
          .includes(searchString.toLowerCase())
      );
      setSuggestions(filtered);
      setShowSuggestions(true);
    } else {
      setShowSuggestions(false);
    }
  };

  const handleMentionClick = (user: Tag) => {
    const [newText, newCursorPosition] = insertMention(
      text,
      cursorPosition,
      `{${user.value}}`
    );
    setText(newText);
    setCursorPosition(newCursorPosition);
    setShowSuggestions(false);
    inputRef.current?.focus();
  };

  const handleSubmit = () => {
    setText('');
    setCursorPosition(0);
    onSubmit?.(text);
  };

  const suggestionsGroup = suggestions.reduce(
    (acc, user) => {
      const group = acc.find((g) => g.label === user.group);
      if (group) {
        group.users.push(user);
      } else {
        acc.push({ label: user.group, users: [user] });
      }
      return acc;
    },
    [] as { label: string; users: Tag[] }[]
  );

  return (
    <div className="relative">
      <div className="flex flex-col gap-1.5 rounded-xl border bg-card p-1.5 focus-within:border-violet-400 focus-within:ring-1 focus-within:ring-violet-400/40">
        <Textarea
          ref={inputRef}
          placeholder="Ask for a rewrite, or type @ to insert a field"
          value={text}
          onChange={handleOnChange}
          onKeyDown={handleKeyDown}
          className="max-h-32 min-h-[54px] resize-none border-0 bg-transparent px-2 py-1.5 text-sm shadow-none focus-visible:ring-0"
        />
        {/* Wraps rather than clipping when the pane is dragged to its
            minimum width — the Send button must never fall off the edge. */}
        <div className="flex flex-wrap items-center gap-1 px-0.5 pb-0.5">
          <UploadDocumentDialogV2 setCountRefresh={setCountRefresh}>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 gap-1.5 px-2 text-xs font-medium text-muted-foreground hover:text-foreground"
              title="Attach a document to your knowledge base"
            >
              <Paperclip className="size-3.5" />
              Attach
            </Button>
          </UploadDocumentDialogV2>
          <GetKnowledgeDialogV2
            countRefresh={countRefresh}
            onChangeKnowledge={onChangeKnowledge}
            onKnowledgeCount={setKnowledgeCount}
          >
            <Button
              variant="ghost"
              size="sm"
              className="h-7 gap-1.5 px-2 text-xs font-medium text-muted-foreground hover:text-foreground"
              title="Browse knowledge base"
            >
              <BookOpenText className="size-3.5" />
              Knowledge
              {knowledgeCount !== null && knowledgeCount > 0 && (
                <span className="rounded-full bg-violet-100 px-1.5 text-[0.65rem] font-semibold leading-4 text-violet-700 dark:bg-violet-500/20 dark:text-violet-300">
                  {knowledgeCount}
                </span>
              )}
            </Button>
          </GetKnowledgeDialogV2>

          <Button
            size="icon"
            aria-label="Send"
            title="Send (Enter)"
            className="ml-auto size-7 shrink-0 rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 hover:opacity-90"
            onClick={handleSubmit}
            disabled={!text.trim()}
          >
            <SendHorizonal className="size-3.5" />
          </Button>
        </div>
      </div>

      <p className="mt-1.5 px-1 text-[0.65rem] leading-snug text-muted-foreground">
        {knowledgeCount !== null && knowledgeCount > 0
          ? `Grounded in ${knowledgeCount} document${knowledgeCount === 1 ? '' : 's'} · `
          : ''}
        Enter to send
      </p>

      {showSuggestions && (
        <div className="absolute bottom-full left-0 z-50 mb-2 max-h-52 w-full overflow-y-auto overflow-x-hidden rounded-lg border bg-popover p-1 text-popover-foreground shadow-md">
          {suggestions.length > 0 ? (
            suggestionsGroup.map((group, gIndex) => (
              <div key={group.label} className="overflow-hidden p-1">
                <div className="px-2 py-1.5 text-[0.7rem] font-medium text-muted-foreground">
                  {group.label}
                </div>
                {group.users.map((tag, tIndex) => {
                  const suggestionIndex = Array.from({
                    length: gIndex,
                  }).reduce(
                    (acc: number, _, i) =>
                      acc + suggestionsGroup[i].users.length,
                    0
                  ) as number;
                  const isActive =
                    activeSuggestionIndex === suggestionIndex + tIndex;
                  return (
                    <button
                      id={`suggestion-${suggestionIndex + tIndex}`}
                      key={tag.id}
                      type="button"
                      onClick={() => handleMentionClick(tag)}
                      className={cn(
                        'relative flex w-full cursor-pointer select-none items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground',
                        isActive && 'bg-accent text-accent-foreground'
                      )}
                    >
                      <span className="truncate">{tag.label}</span>
                      <span className="ml-auto shrink-0 truncate font-mono text-[0.65rem] text-muted-foreground">
                        {`{${tag.value}}`}
                      </span>
                    </button>
                  );
                })}
              </div>
            ))
          ) : (
            <div className="px-2 py-4 text-center text-sm text-muted-foreground">
              No matching field
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function KnowledgeCard({
  knowledge,
  botInfo,
  handleSelect,
}: {
  knowledge: any;
  botInfo: any;
  handleSelect?: (
    knowledge: { coze_dataset_id: string },
    isUsed: boolean
  ) => void;
}) {
  const [loading, setLoading] = useState(false);
  const isUsedDocument = useCallback(
    (knowledge: { coze_dataset_id: string }) => {
      return botInfo?.knowledge?.knowledge_infos
        ?.map((item: { id: string }) => item.id)
        .includes(knowledge?.coze_dataset_id);
    },
    [botInfo?.knowledge?.knowledge_infos]
  );
  const handleOnClick = async () => {
    setLoading(true);
    await handleSelect?.(knowledge, isUsedDocument(knowledge));
    setLoading(false);
  };
  return (
    <div
      className={`space-y-2 rounded-lg border p-4 ${isUsedDocument(knowledge) ? 'border-[#7724FF]' : ''}`}
    >
      <p className="font-semibold">
        {knowledge.cmetadata?.origin?.filename || 'Untitled document'}
      </p>
      <p>{knowledge.summary_pair?.[0] || ''}</p>
      <div className="flex items-center justify-between">
        <p className="text-muted-foreground">
          Updated {format(new Date(knowledge.updated_at), 'P')}
        </p>
        <MainButton
          loading={loading}
          text={isUsedDocument(knowledge) ? 'Remove article' : 'Use article'}
          variant={isUsedDocument(knowledge) ? 'outline' : 'ghost'}
          onClick={handleOnClick}
          className={
            isUsedDocument(knowledge) ? 'bg-red-400 text-white' : 'text-info'
          }
        />
      </div>
    </div>
  );
}

interface GetKnowledgeDialogV2Props extends PropsWithChildren {
  countRefresh: number;
  onChangeKnowledge?: (values: string[]) => void;
  /** Reports how many documents the bot is currently grounded in. */
  onKnowledgeCount?: (count: number) => void;
}

function GetKnowledgeDialogV2({
  children,
  countRefresh,
  onChangeKnowledge,
  onKnowledgeCount,
}: GetKnowledgeDialogV2Props) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  const {
    data,
    isLoading,
    refetch: refetchKnowledge,
  } = useQuery({
    queryKey: ['knowledge', `knowledge-${countRefresh}`],
    queryFn: () => KnowledgeApiV2.list({ limit: 10, offset: 0 }),
  });

  const {
    data: botInfo,
    isLoading: botInfoLoading,
    refetch: refetchBotInfo,
  } = useQuery({
    queryKey: ['bot-info', `bot-info-${countRefresh}`],
    queryFn: () => chatApi.botInfo(),
  });

  const handleSelect = async (
    knowledge: { coze_dataset_id: string },
    isUsed: boolean
  ) => {
    const datasetId = knowledge?.coze_dataset_id;
    const knowledgeIds = botInfo?.knowledge?.knowledge_infos?.map(
      (item: { id: string }) => item.id
    );

    if (!datasetId) {
      return;
    }
    await chatApi.updateBot(
      isUsed
        ? knowledgeIds?.filter((id: string) => id !== datasetId) || []
        : [...(knowledgeIds || []), datasetId]
    );
    await chatApi.publicBot();
    refetchKnowledge();
    refetchBotInfo();
    setIsDialogOpen(false);
    toast.success(
      isUsed ? 'Remove article successfully' : 'Use article successfully'
    );
  };

  const onSearch = (search: string) => {
    setSearchTerm(search.toLowerCase());
  };

  const filteredData = useMemo(() => {
    // Records uploaded before the metadata shape settled can be missing
    // cmetadata.origin or summary_pair entirely; reading through them blindly
    // took the whole writer panel down with a TypeError.
    return data?.data.filter(
      (knowledge) =>
        (knowledge.cmetadata?.origin?.filename || '')
          .toLowerCase()
          .includes(searchTerm) ||
        (knowledge.summary_pair?.[0] || '').toLowerCase().includes(searchTerm)
    );
  }, [data?.data, searchTerm]);

  useEffect(() => {
    let tempKnowledgeIds: string[] = [];
    if (botInfo?.knowledge?.knowledge_infos?.length) {
      filteredData?.forEach((item: { coze_dataset_id: string; id: string }) => {
        if (
          botInfo?.knowledge?.knowledge_infos
            ?.map((item: { id: string }) => item.id)
            ?.includes(item.coze_dataset_id)
        ) {
          tempKnowledgeIds.push(item.id);
        }
      });
    }
    if (tempKnowledgeIds.length > 0) {
      onChangeKnowledge?.(tempKnowledgeIds);
    }
  }, [botInfo?.knowledge?.knowledge_infos, filteredData]);

  useEffect(() => {
    if (botInfoLoading) return;
    onKnowledgeCount?.(botInfo?.knowledge?.knowledge_infos?.length ?? 0);
  }, [botInfo?.knowledge?.knowledge_infos, botInfoLoading]);

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="z-[1000] sm:max-w-[768px]" key={countRefresh}>
        <DialogHeader>
          <DialogTitle>Knowledge Base</DialogTitle>
          <DialogDescription>
            Search through helpful resources
          </DialogDescription>
        </DialogHeader>
        <Separator />
        <Input
          placeholder="Search knowledge"
          onChange={(e) => onSearch(e.target.value)}
        />
        <ScrollArea className="h-[400px]">
          {isLoading || botInfoLoading ? (
            <div className="flex flex-col gap-2">
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className="h-32" />
              ))}
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {filteredData?.map((knowledge) => (
                <KnowledgeCard
                  key={knowledge.id}
                  knowledge={knowledge}
                  botInfo={botInfo}
                  handleSelect={handleSelect}
                />
              ))}
            </div>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}

interface UploadDocumentDialogV2Props extends PropsWithChildren {
  setCountRefresh: (count: any) => void;
}

function UploadDocumentDialogV2({
  children,
  setCountRefresh,
}: UploadDocumentDialogV2Props) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);

  const handleUpload = async (data) => {
    const datasetId = data?.coze_dataset_id;
    if (datasetId) {
      await chatApi.updateBot([datasetId]);
      await chatApi.publicBot();
      setIsDialogOpen(false);
      setFile(null);
      setCountRefresh((count) => count + 1);
      toast.success('Document uploaded successfully');
    }
  };

  const { mutateAsync, isPending } = useMutation({
    mutationFn: () => KnowledgeApiV2.upload([file as File]),
    onSuccess: async (data) => {
      if (data?.data?.[0]?.result?.knowledge) {
        await handleUpload(data?.data?.[0]?.result?.knowledge);
      }
    },
    onError: (error: any) => {
      toast.error(
        error.error?.message || error.message || 'Oops! Something went wrong'
      );
    },
  });

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="z-[1000] sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>Upload Document</DialogTitle>
          <DialogDescription>
            Upload a document to personalize your email
          </DialogDescription>
        </DialogHeader>
        <Separator />
        <FileDropzone
          onDrop={(files) => setFile(files[0] ?? null)}
          maxSize={5 * 1024 * 1024}
          accept={{
            'application/pdf': ['.pdf'],
          }}
          multiple={false}
        />
        <div className="mt-2 flex justify-end gap-2">
          <MainButton
            variant="outline"
            text="Cancel"
            onClick={() => {
              setIsDialogOpen(false);
              setFile(null);
            }}
          />
          <MainButton
            text="Upload"
            onClick={() => mutateAsync()}
            disabled={!file}
            loading={isPending}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
