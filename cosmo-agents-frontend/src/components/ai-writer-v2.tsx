'use client';

import { useCampaign } from '@/app/(main)/campaigns/[campaign_id]/use-campaign';
import { MainButton } from '@/components/buttons/main-button';
import { Card, CardContent } from '@/components/ui/card';
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
import { BookOpenText, Paperclip, SendHorizonal } from 'lucide-react';
import Image from 'next/image';
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
import { ChatMessage } from './chat/ChatMessage';
import { ChatMessageList } from './chat/ChatMessageList';
import { ChatTypingBubble } from './chat/ChatTypingBubble';
import CodeDisplayBlock from './chat/CodeBlock';
import FileDropzone from './file-dropzone';
import { Button } from './ui/button';
import { Skeleton } from './ui/skeleton';
import { Textarea } from './ui/textarea';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';

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
}: {
  conversationId?: string;
  campaignType?: string;
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

  const { data: mergeTags } = useQuery({
    queryKey: ['mergeTags'],
    queryFn: () => CampaignApi.mergeTags(campaign.id),
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
    (item) => item.id === campaign.organization_id
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

  const tagsOptions = Object.fromEntries(
    customTags.map((tag) => [tag.value, tag.label])
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
     - description: ${organization.company_description} / ${organization?.company_targeting_persona?.join(', ') || ''} \n\n
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

  const handleClearConversation = async () => {
    setIsSuggestion(false);
    setConversationLoading(true);
    setIsTyping(false);
    setTextStream('');
    setMessages([]);
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

  return (
    <div className="flex h-full flex-col border bg-background">
      <div className="border-b px-4 py-2">
        <div className="flex w-full items-center justify-between">
          <p className="text-lg font-semibold">AI Writer</p>
          <Tooltip>
            <TooltipTrigger
              className="inline-flex"
              onClick={handleClearConversation}
            >
              <Image src="/broom.svg" alt="Broom" width={20} height={20} />
            </TooltipTrigger>
            <TooltipContent side="bottom" align="start" className="z-[51]">
              Clear
            </TooltipContent>
          </Tooltip>
        </div>
      </div>
      {/* Chat Interface Section */}
      <div className="flex flex-1 flex-col p-4 pl-1">
        <div className="mb-4 h-52 flex-grow">
          <ChatMessageList smooth ref={messagesRef}>
            {messages.map((m, idx) => {
              if (m.role === 'user') {
                return (
                  <ChatMessage key={idx} message={renderContent(m)} fromUser />
                );
              }
              return (
                <ChatMessage
                  key={idx}
                  message={renderContent(m)}
                  fromUser={false}
                  avatarUrl="/favicon.svg"
                />
              );
            })}
            {isSuggestion && (
              <div className="flex flex-wrap gap-2 px-2">
                {suggestionAction.map((action) => (
                  <Button
                    className="border-1 border-black bg-white text-black hover:text-white"
                    key={action.value}
                    onClick={() => handleSuggestionAction(action.label)}
                  >
                    {action.label}
                  </Button>
                ))}
              </div>
            )}
            {(isTyping || conversationLoading) && (
              <ChatTypingBubble text={textStream} avatarUrl="/favicon.svg" />
            )}
          </ChatMessageList>
        </div>
        <div className="mt-auto">
          <MentionInput
            customTags={customTags}
            onSubmit={handleSendInstruction}
            onChangeKnowledge={onChangeKnowledge}
          />
        </div>
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
}: {
  customTags: Tag[];
  onSubmit?: (text: string) => Promise<void>;
  onChangeKnowledge?: (values: string[]) => void;
}) {
  const [activeSuggestionIndex, setActiveSuggestionIndex] = useState(0);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const [text, setText] = useState('');
  const [cursorPosition, setCursorPosition] = useState(0);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [suggestions, setSuggestions] = useState<Tag[]>([]);
  const [countRefresh, setCountRefresh] = useState(0);

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
    <div className="relative -mr-2 mb-2 ml-0.5 flex flex-col gap-2 rounded-lg border bg-background p-1.5">
      <Textarea
        ref={inputRef}
        placeholder="Type @ to access field"
        value={text}
        onChange={handleOnChange}
        onKeyDown={handleKeyDown}
        className="h-[40px] max-h-[80px] min-h-[50px]"
      />
      <div className="flex items-center pt-0">
        <UploadDocumentDialogV2 setCountRefresh={setCountRefresh}>
          <MainButton
            variant="ghost"
            size="sm"
            text="Upload"
            icon={Paperclip}
          />
        </UploadDocumentDialogV2>
        <GetKnowledgeDialogV2
          countRefresh={countRefresh}
          onChangeKnowledge={onChangeKnowledge}
        >
          <MainButton
            variant="ghost"
            size="sm"
            text="Knowledge"
            icon={BookOpenText}
          />
        </GetKnowledgeDialogV2>

        <MainButton
          size="sm"
          className="mb-1 ml-auto mr-1 gap-1.5"
          onClick={handleSubmit}
          icon={SendHorizonal}
        />
      </div>
      {showSuggestions && (
        <Card className="absolute bottom-full left-0 mb-2 max-h-40 w-full overflow-y-auto rounded-md">
          <CardContent className="p-2">
            {suggestions.length > 0 ? (
              suggestionsGroup.map((group, gIndex) => (
                <div key={group.label}>
                  <p className="font-semibold">{group.label}</p>
                  {group.users.map((tag, tIndex) => {
                    const suggestionIndex = Array.from({
                      length: gIndex,
                    }).reduce(
                      (acc: number, _, i) =>
                        acc + suggestionsGroup[i].users.length,
                      0
                    ) as number;
                    return (
                      <MainButton
                        id={`suggestion-${suggestionIndex + tIndex}`}
                        key={tag.id}
                        variant="ghost"
                        className={cn(
                          'w-full justify-start text-left text-muted-foreground transition-all duration-200',
                          activeSuggestionIndex === suggestionIndex + tIndex &&
                            'bg-accent text-accent-foreground'
                        )}
                        onClick={() => handleMentionClick(tag)}
                        text={tag.label}
                      />
                    );
                  })}
                </div>
              ))
            ) : (
              <div className="text-center text-muted-foreground">
                No matching users
              </div>
            )}
          </CardContent>
        </Card>
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
      <p className="font-semibold">{knowledge.cmetadata.origin.filename}</p>
      <p>{knowledge.summary_pair[0]}</p>
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
}

function GetKnowledgeDialogV2({
  children,
  countRefresh,
  onChangeKnowledge,
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
    return data?.data.filter(
      (knowledge) =>
        knowledge.cmetadata.origin.filename
          .toLowerCase()
          .includes(searchTerm) ||
        knowledge.summary_pair[0].toLowerCase().includes(searchTerm)
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
