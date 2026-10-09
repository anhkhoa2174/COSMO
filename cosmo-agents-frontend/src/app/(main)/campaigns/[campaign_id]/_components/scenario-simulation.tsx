'use client';

import { IconRobot } from '@/assets/icons';
import { MainButton } from '@/components/buttons/main-button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { Textarea } from '@/components/ui/textarea';
import { buildEmailContext, getContent, getErrorMessage } from '@/helpers';
import { promptAIReply } from '@/helpers/prompt';
import useOrgByCampaign from '@/hooks/use-org-by-campain';
import { useTypingEffect } from '@/hooks/use-typing-effect';
import { EmailIntent } from '@/models/email';
import { classifyIntent } from '@/network/client/ai';
import { useMutation } from '@tanstack/react-query';
import { Send, TriangleAlert } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import {
  PreviewEmailWithIntent,
  useCampaign,
  useCampaignSupport,
} from '../use-campaign';
import { usePrefetchV3 } from '../use-prefetch-v2';
import { SheetHeader } from './sheet-header';

interface ConversationProps {
  data: PreviewEmailWithIntent;
  previewContact: any;
  currentUser: any;
  agent: any;
  scrollRef: React.RefObject<HTMLDivElement>;
}

function Conversation({
  data,
  previewContact,
  currentUser,
  agent,
  scrollRef,
}: ConversationProps) {
  const [campaign] = useCampaign();
  const { config } = campaign.cmetadata;
  const [content, setContent] = useState('');
  const [isFinished, setIsFinished] = useState(false);
  const configFound = config.find((item) => item.intent_type === data.intent);

  useTypingEffect({
    text: data.content || '',
    isActive: true,
    setText: (value) => value && setContent(value),
    speed: 50,
    step: 10,
    dependencies: [],
    onFinished: () => setIsFinished(true),
  });

  useEffect(() => {
    if (isFinished) {
      scrollRef?.current?.scrollIntoView({
        behavior: 'smooth',
      });
    }
  }, [isFinished]);

  return (
    <Card>
      <CardContent className="space-y-4 pt-4">
        <div className="flex items-center gap-2">
          {data.status === 'send' ? (
            <>
              <Avatar>
                <AvatarFallback>{data.from_email?.slice(0, 2)}</AvatarFallback>
              </Avatar>
            </>
          ) : (
            <span className="relative">
              <Avatar>
                <AvatarImage
                  src={agent?.data.entity.picture}
                  alt={agent?.data.entity.email}
                />
                <AvatarFallback>
                  {agent?.data?.entity?.email?.slice(0, 2)}
                </AvatarFallback>
              </Avatar>
              <span className="absolute -bottom-1 -left-1">
                <IconRobot width={20} height={20} />
              </span>
            </span>
          )}
          <div className="flex-1">
            <p className="text-sm font-medium">{data.from_email}</p>
            <p className="text-sm text-muted-foreground">
              <span className="font-semibold">To:</span> {data.to_email}
            </p>
          </div>
          {data.intent && getIntentLabel(data.intent)}
        </div>
        <div className="text-base leading-loose">
          <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeRaw]}>
            {getContent(content, previewContact, currentUser, agent)}
          </Markdown>
        </div>
        {data.intent && !configFound && (
          <>
            <Separator />
            <p className="text-sm text-destructive">
              No action found. Please configure the action for this intent.
            </p>
          </>
        )}
        {data.intent && configFound?.who === 'Assign to a person' && (
          <>
            <Separator />
            <p className="text-sm text-info">
              This intent is configured to be assigned to a person. Therefore
              this email will not be replied.
            </p>
          </>
        )}
      </CardContent>
    </Card>
  );
}

interface ReplyInputProps {
  onConversationChange: (conversation: PreviewEmailWithIntent) => void;
  selectedContact: any;
  agent: any;
  disabled: boolean;
}

function ReplyInput({
  onConversationChange,
  selectedContact,
  agent,
  disabled,
}: ReplyInputProps) {
  const [value, setValue] = useState('');

  const classifyIntentMutation = useMutation({
    mutationFn: classifyIntent,
    onSuccess: ({ data }) => {
      onConversationChange({
        subject: '',
        to_email: agent?.data.entity.email,
        content: value,
        intent: data.intent,
        from_email: selectedContact?.email,
        status: 'send',
      });
      setValue('');
    },
  });

  return (
    <Card>
      <CardContent className="space-y-4 pt-4">
        <div className="flex items-center gap-2">
          <Avatar>
            <AvatarFallback>
              {selectedContact?.email?.slice(0, 2)}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1">
            <p className="text-sm font-medium">{selectedContact?.email}</p>
            <p className="text-sm text-muted-foreground">
              <span className="font-semibold">To:</span>{' '}
              {agent?.data.entity.email}
            </p>
          </div>
        </div>
        <Separator />
        <Textarea
          placeholder="Test the AI with a sample reply..."
          rows={4}
          value={value}
          onChange={(event) => setValue(event.target.value)}
          disabled={classifyIntentMutation.isPending || disabled}
          className="resize-none border-none shadow-none"
        />
        {classifyIntentMutation.error && (
          <p className="text-sm text-destructive">
            {getErrorMessage(classifyIntentMutation.error)}
          </p>
        )}
        <div className="flex justify-end">
          <MainButton
            disabled={!value.length || disabled}
            loading={classifyIntentMutation.isPending}
            text="Send"
            rightIcon={Send}
            onClick={() => classifyIntentMutation.mutate({ content: value })}
            size="sm"
          />
        </div>
      </CardContent>
    </Card>
  );
}

interface ScenarioSimulationProps {
  closeSheet: () => void;
}

export default function ScenarioSimulation({
  closeSheet,
}: ScenarioSimulationProps) {
  const [campaign] = useCampaign();

  if (campaign.agent_id) {
    return <ScenarioSimulationWithAgent closeSheet={closeSheet} />;
  }

  return (
    <div className="flex h-full flex-col p-4">
      <p className="flex items-center justify-center gap-2 p-4 text-base text-yellow-600">
        <TriangleAlert /> Please choose an agent (AI Inbox) to simulate the
        preview
      </p>
      <MainButton text="Close" onClick={closeSheet} />
    </div>
  );
}

function ScenarioSimulationWithAgent({ closeSheet }: ScenarioSimulationProps) {
  const [campaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const [organization] = useOrgByCampaign();
  const { conversations, previewContact, templates } = campaignSupport;
  const templateId = campaign.templates[0]?.id || '';
  const scrollRef = useRef<HTMLDivElement>(null);
  const { currentUser, agent, error, isLoading, getGeneratedTemplate } =
    usePrefetchV3(templateId);

  useEffect(() => {
    if (
      templates[templateId] &&
      previewContact &&
      agent &&
      !conversations.length
    ) {
      const toEmail =
        previewContact.profile?.email ?? previewContact.email ?? '';
      const initialEmail: PreviewEmailWithIntent = {
        from_email: agent.data.entity.email,
        to_email: toEmail,
        subject: templates[templateId].subject || '',
        content: templates[templateId].content || '',
        status: 'reply',
      };
      setCampaignSupport((prev) => ({
        ...prev,
        conversations: [initialEmail],
      }));
    }
  }, [templates, previewContact, agent]);

  const handleSaveAndClose = async () => {
    setCampaignSupport((prev) => ({ ...prev, conversations }));
    closeSheet();
  };

  const handleConversationChange = (conversation: PreviewEmailWithIntent) => {
    setCampaignSupport((prev) => ({
      ...prev,
      conversations: [...prev.conversations, conversation],
    }));
  };

  // Generate Email Reply
  const generateEmailReply = useCallback(
    async ({
      conversation,
      intent,
    }: {
      conversation: PreviewEmailWithIntent[];
      intent: EmailIntent;
    }) => {
      const context = buildEmailContext({
        organization,
        intentType: intent,
        subject: '',
        content: conversation[0]?.content || '',
        words: 60,
      });
      const { content: replyContent } = await getGeneratedTemplate([
        { content: context, content_type: 'text', role: 'user' },
        {
          content: promptAIReply(),
          content_type: 'text',
          role: 'user',
        },
      ]);
      const replyEmail: PreviewEmailWithIntent = {
        from_email: agent?.data.entity.email || '',
        to_email: previewContact?.email || '',
        subject: '',
        content: replyContent,
        status: 'reply',
      };
      setCampaignSupport((prev) => ({
        ...prev,
        conversations: [...prev.conversations, replyEmail],
      }));
    },
    [previewContact, agent, getGeneratedTemplate, setCampaignSupport]
  );

  useEffect(() => {
    const lastConversation = conversations[conversations.length - 1];
    if (lastConversation?.status === 'send') {
      const { config } = campaign.cmetadata;

      const configFound = config.find(
        (item) => item.intent_type === lastConversation.intent
      );

      if (configFound?.who === 'Assign to a person') {
        return;
      }

      if (configFound?.who === 'Draft an email') {
        const email: PreviewEmailWithIntent = {
          from_email: agent?.data.entity.email || '',
          to_email: previewContact?.email || '',
          subject: '',
          content: configFound.payload.content || '',
          status: 'reply',
        };
        setCampaignSupport((prev) => ({
          ...prev,
          conversations: [...prev.conversations, email],
        }));
      }

      if (configFound?.who === 'Let AI reply') {
        generateEmailReply({
          conversation: [lastConversation],
          intent: lastConversation.intent as EmailIntent,
        });
      }
    }
  }, [conversations.length]);

  return (
    <div className="flex h-full flex-col">
      <SheetHeader
        title="Preview"
        isError={!!error}
        hasChanged={false}
        onSubmit={() => {}}
        onClose={handleSaveAndClose}
      />
      <Separator />
      {error ? (
        <p className="inline-flex justify-center gap-2 p-4 text-center text-base text-destructive">
          <TriangleAlert /> {getErrorMessage(error)}
        </p>
      ) : (
        <div className="h-96 flex-grow">
          <ScrollArea className="h-full bg-[#F7F7F7] p-4" type="always">
            <div className="flex flex-col gap-4">
              {conversations.map((conversation, index) => (
                <Conversation
                  key={`conversation-${index}`}
                  data={conversation}
                  previewContact={previewContact}
                  currentUser={currentUser}
                  agent={agent}
                  scrollRef={scrollRef}
                />
              ))}
              {isLoading && (
                <div className="flex flex-col space-y-4">
                  <p className="text-sm text-muted-foreground">
                    AI is generating a reply...
                  </p>
                  <Skeleton className="h-[300px] rounded-lg" />
                </div>
              )}
              <ReplyInput
                disabled={isLoading}
                onConversationChange={handleConversationChange}
                selectedContact={previewContact}
                agent={agent}
              />
            </div>
            <div ref={scrollRef} />
          </ScrollArea>
        </div>
      )}
    </div>
  );
}

function getIntentLabel(intent: string) {
  let className = '';
  switch (intent) {
    case 'Interested':
      className = 'text-blue-500 bg-blue-500/10 hover:bg-blue-500/20';
      break;
    case 'Not interested':
      className = 'text-green-500 bg-green-500/10 hover:bg-green-500/20';
      break;
    case 'Request for pricing':
      className = 'text-yellow-500 bg-yellow-500/10 hover:bg-yellow-500/20';
      break;
    case 'Request for information':
      className = 'text-purple-500 bg-purple-500/10 hover:bg-purple-500/20';
      break;
    case 'Do not contact':
      className = 'text-red-500 bg-red-500/10 hover:bg-red-500/20';
      break;
    case 'Out of office':
      className = 'text-orange-500 bg-orange-500/10 hover:bg-orange-500/20';
      break;
    default:
      className = 'text-gray-500 bg-gray-500/10 hover:bg-gray-500/20';
  }

  return <Badge className={`uppercase ${className}`}>{intent}</Badge>;
}
