'use client';

import { IconRobot } from '@/assets/icons';
import { AIWriterV2 } from '@/components/ai-writer-v2';
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable';
import { MainButton } from '@/components/buttons/main-button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { Spinner } from '@/components/ui/spinner';
import {
  buildEmailContext,
  fakeStream,
  getContent,
  getErrorMessage,
} from '@/helpers';
import { promptAIReply } from '@/helpers/prompt';
import useOrgByCampaign from '@/hooks/use-org-by-campain';
import type { CampaignConfig } from '@/models/campaign';
import type { EmailIntent } from '@/models/email';
import { SalesRep } from '@/models/sales-rep';
import chatApi from '@/network/client/ai-chat';
import CampaignApi from '@/network/client/campaign';
import SalesRepApi from '@/network/client/sales-rep';
import { GetTemplateData } from '@/network/client/template';
import { useMutation } from '@tanstack/react-query';
import _ from 'lodash';
import { RefreshCw, Sparkles, TriangleAlert } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { toast } from 'sonner';
import {
  PreviewEmailWithIntent,
  useCampaign,
  useCampaignSupport,
} from '../../use-campaign';
import { usePrefetchV3 } from '../../use-prefetch-v2';
import SalesReps from '../sales-reps';
import { sampleTemplatesHumanReply } from './templates';

interface ConversationProps {
  data: PreviewEmailWithIntent;
  previewContact: any;
  currentUser: any;
  agent: any;
  salesReps: SalesRep[];
}

function Conversation({
  data,
  previewContact,
  currentUser,
  agent,
  salesReps,
}: ConversationProps) {
  const [campaign] = useCampaign();
  const { config } = campaign.cmetadata;

  const configFound = config.find((item) => item.intent_type === data.intent);

  return (
    <Card>
      <CardContent className="space-y-4 pt-4">
        <div className="flex items-center gap-2">
          {data.status === 'send' ? (
            <>
              <Avatar>
                <AvatarFallback>{data.from_email.slice(0, 2)}</AvatarFallback>
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
                  {agent?.data.entity.email.slice(0, 2)}
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
            {getContent(
              data.content,
              previewContact,
              currentUser,
              agent,
              salesReps
            )}
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

type ConfigPayload = {
  sale_rep_ids: string[];
  content?: string;
  knowledge_ids?: string[];
};

interface AIReplyProps {
  intent: EmailIntent;
  closeSheet: () => void;
  onSuccess: () => void;
}

export default function AIReplyV2({
  intent,
  closeSheet,
  onSuccess,
}: AIReplyProps) {
  const [campaign] = useCampaign();

  if (campaign.agent_id) {
    return (
      <AIReplyWithAgent
        intent={intent}
        closeSheet={closeSheet}
        onSuccess={onSuccess}
      />
    );
  }

  return (
    <div className="flex h-full flex-col p-4">
      <p className="flex items-center justify-center gap-2 p-4 text-base text-yellow-600">
        <TriangleAlert /> Please choose an agent (AI Inbox) to simulate the AI
        reply
      </p>
      <MainButton text="Close" onClick={closeSheet} />
    </div>
  );
}

function AIReplyWithAgent({ intent, closeSheet, onSuccess }: AIReplyProps) {
  const defaultOption: CampaignConfig<ConfigPayload> = {
    intent_type: intent,
    who: 'Let AI reply',
    payload: { sale_rep_ids: [], knowledge_ids: [] },
  };
  const scrollRef = useRef<HTMLDivElement>(null);
  const [campaign, setCampaign] = useCampaign();
  const router = useRouter();
  const [organization] = useOrgByCampaign();
  const [loadingSample, setLoadingSample] = useState(false);
  const [loadingReply, setLoadingReply] = useState(false);
  const [salesReps, setSalesReps] = useState<SalesRep[]>([]);
  const [loadingSalesReps, setLoadingSalesReps] = useState(false);
  const [assistantOpen, setAssistantOpen] = useState(false);
  const [conversationId, setConversationId] = useState<string | null>(null);

  const existOption = (campaign.cmetadata.config || []).find(
    (c) => c.intent_type === intent
  );
  const [option, setOption] = useState<CampaignConfig<ConfigPayload>>(
    existOption || defaultOption
  );

  const templateId = campaign.templates[0]?.id || '';
  const { currentUser, agent, error, getGeneratedTemplate } =
    usePrefetchV3(templateId);

  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const { templates, previewContact, preview } = campaignSupport;

  const hasChanged = useMemo(
    () =>
      !existOption ||
      !_.isEqual(
        existOption?.payload?.sale_rep_ids,
        option?.payload?.sale_rep_ids
      ) ||
      !_.isEqual(
        existOption?.payload?.knowledge_ids,
        option?.payload?.knowledge_ids
      ),
    [existOption, option]
  );

  // Generate Sample Response
  const generateSampleResponse = useCallback(async () => {
    setLoadingSample(true);
    setTimeout(() => {
      const contentSample =
        sampleTemplatesHumanReply[intent][
          Math.floor(Math.random() * sampleTemplatesHumanReply[intent].length)
        ];
      const initialEmail: PreviewEmailWithIntent = {
        from_email: previewContact?.email || '',
        to_email: agent?.data.entity.email || '',
        subject: '',
        content: contentSample || '',
        status: 'send',
      };
      fakeStream({
        data: contentSample,
        callback: (data) => {
          setCampaignSupport((prev) => ({
            ...prev,
            preview: {
              ...prev.preview,
              [intent]: [{ ...initialEmail, content: data }],
            },
          }));
        },
      });
      setLoadingSample(false);
    }, 300);
  }, [intent, previewContact, agent, getGeneratedTemplate, setCampaignSupport]);

  useEffect(() => {
    if (
      !!templates[templateId] &&
      !!previewContact &&
      !preview[intent].length
    ) {
      generateSampleResponse();
    }
  }, [templates, previewContact, preview]);

  const conversation = useMemo(() => {
    return campaignSupport.preview[intent];
  }, [campaignSupport.preview, intent]);

  // Generate Email Reply
  const generateEmailReply = useCallback(async () => {
    setLoadingReply(true);
    const context = buildEmailContext({
      organization,
      intentType: intent,
      subject: '',
      content: conversation[0]?.content || '',
      words: 60,
    });
    const { subject: replySubject, content: replyContent } =
      await getGeneratedTemplate([
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
      subject: replySubject || '',
      content: replyContent || '',
      status: 'reply',
    };
    fakeStream({
      data: replyContent,
      callback: (data) => {
        setCampaignSupport((prev) => ({
          ...prev,
          preview: {
            ...prev.preview,
            [intent]: [
              prev.preview[intent]?.[0] ?? null,
              { ...replyEmail, content: data },
            ],
          },
        }));
      },
    });
    setLoadingReply(false);
  }, [intent, previewContact, agent, getGeneratedTemplate, setCampaignSupport]);

  const aiReplyData = useMemo(() => {
    const context = buildEmailContext({
      organization,
      intentType: intent,
      subject: '',
      content: conversation[1]?.content || '',
      words: 60,
    });
    return [{ content: context, content_type: 'text', role: 'user' }];
  }, [conversation]);

  useEffect(() => {
    if (conversation?.length === 1) {
      generateEmailReply();
    }
  }, [conversation?.length]);

  // Assign Campaign Intent
  const {
    mutate: assignCampaignIntent,
    isPending: isAssignCampaignIntentPending,
  } = useMutation({
    mutationFn: () => {
      const copy = [...(campaign.cmetadata.config || [])];
      const payload = {
        config: copy.filter((c) => c.intent_type !== intent).concat(option),
      };
      return CampaignApi.assign(campaign.id, payload);
    },
    onSuccess: () => {
      const copy = [...(campaign.cmetadata.config || [])];
      const payload = {
        config: copy.filter((c) => c.intent_type !== intent).concat(option),
      };
      setCampaign((prev) => ({
        ...prev,
        cmetadata: { ...prev.cmetadata, ...payload },
      }));
      onSuccess();
      closeSheet();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    },
  });

  const handleSave = () => {
    assignCampaignIntent();
  };

  const handleOnAddASalesReps = () => {
    router.push(`/sales-reps?isModalOpen=true`);
  };

  const handleOnChangeSalesReps = (values: string[]) => {
    setOption((prev) => ({
      ...prev,
      payload: { ...prev.payload, sale_rep_ids: values },
    }));
  };

  const handleOnChangeKnowledge = (values: string[]) => {
    setOption((prev) => ({
      ...prev,
      payload: { ...prev.payload, knowledge_ids: values },
    }));
  };

  const handleChange = (key: keyof GetTemplateData, value: any) => {
    setCampaignSupport((prev) => ({
      ...prev,
      preview: {
        ...prev.preview,
        [intent]: prev.preview[intent].map((c) => {
          if (c.status === 'reply') {
            return { ...c, [key]: value };
          }
          return c;
        }),
      },
    }));
  };

  const handleOnClickAiWriter = async () => {
    setAssistantOpen(!assistantOpen);
    if (!conversationId) {
      const newConversationId = await chatApi.postCreateConversation();
      if (newConversationId) {
        setConversationId(newConversationId);
      }
    } else {
      await handleOnClearConversation(true);
    }
  };

  const handleOnCloseAiWriter = async () => {
    if (!conversationId) return;
    setAssistantOpen(false);
    closeSheet();
    await chatApi.deleteClearConversation(conversationId);
  };

  const handleOnClearConversation = async (isClose?: boolean) => {
    if (!conversationId) return;
    await chatApi.deleteClearConversation(conversationId);
    if (isClose) {
      setConversationId('');
      return;
    }
    const newConversationId = await chatApi.postCreateConversation();
    if (newConversationId) {
      setConversationId(newConversationId);
    }
  };

  useEffect(() => {
    async function fetchSalesReps() {
      setLoadingSalesReps(true);
      try {
        const response = await SalesRepApi.search({ filter: {} });
        setSalesReps(
          response.data.list.map((l) => ({
            id: l.entity.id,
            calendar_link: l.entity.calendar_link,
            first_name: l.entity.first_name,
            last_name: l.entity.last_name,
            email: l.entity.email,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          }))
        );
      } catch (error) {
        toast.error('Failed to fetch sales reps');
      } finally {
        setLoadingSalesReps(false);
      }
    }
    fetchSalesReps();
  }, []);

  const renderHeader = () => (
    <div className="flex items-center gap-2 p-4">
      <p className="text-lg font-bold">Let AI reply</p>
      <div className="ml-auto flex gap-2">
        {!error && (
          <MainButton
            className="bg-gradient-to-r from-[#41AEFD] via-[#D543F5] to-[#5A57F2]"
            text="AI Writer"
            icon={Sparkles}
            onClick={handleOnClickAiWriter}
          />
        )}
        {!error && !loadingSalesReps && (
          <SalesReps
            data={salesReps}
            value={option.payload.sale_rep_ids}
            onClick={handleOnAddASalesReps}
            onChange={handleOnChangeSalesReps}
          />
        )}
        {!error && hasChanged && (
          <MainButton onClick={handleSave} text="Save" />
        )}
        {!error && hasChanged && (
          <MainButton
            onClick={handleOnCloseAiWriter}
            variant="ghost"
            text="Discard"
          />
        )}
        {(!!error || !hasChanged) && (
          <MainButton onClick={handleOnCloseAiWriter} text="Close" />
        )}
      </div>
    </div>
  );

  const commonError = error;

  return (
    <div className="flex h-full flex-col">
      {renderHeader()}
      <Separator />
      {commonError ? (
        <p className="inline-flex justify-center gap-2 p-4 text-center text-base text-destructive">
          <TriangleAlert /> {getErrorMessage(commonError)}
        </p>
      ) : (
        <ResizablePanelGroup
          direction="horizontal"
          autoSaveId="cosmo-ai-reply-v2"
          className="h-96 flex-grow"
        >
          <Spinner
            show={isAssignCampaignIntentPending}
            withOverlay
            label="Saving..."
          />

          <ResizablePanel
            id="conversation"
            order={1}
            defaultSize={70}
            minSize={30}
          >
            <ScrollArea
              className="h-full w-full bg-[#F7F7F7] p-4"
              type="always"
            >
              {loadingSample ? (
                <div className="flex flex-col space-y-4">
                  <Skeleton className="h-[300px] rounded-lg" />
                </div>
              ) : (
                <div className="flex flex-col gap-4">
                  {!!conversation[0] && (
                    <div className="space-y-4">
                      <p className="text-sm">Sample response from lead</p>
                      <Conversation
                        data={conversation[0]}
                        previewContact={previewContact}
                        currentUser={currentUser}
                        agent={agent}
                        salesReps={salesReps}
                      />
                    </div>
                  )}
                  {loadingReply ? (
                    <div className="flex flex-col space-y-4">
                      <Skeleton className="h-[300px] rounded-lg" />
                    </div>
                  ) : (
                    <>
                      {!!conversation[1] && (
                        <div className="space-y-4">
                          <p className="text-sm">AI generates a reply</p>
                          <Conversation
                            data={conversation[1]}
                            previewContact={previewContact}
                            currentUser={currentUser}
                            agent={agent}
                            salesReps={salesReps}
                          />
                          {!assistantOpen && (
                            <div className="flex justify-end">
                              <MainButton
                                size="sm"
                                icon={RefreshCw}
                                onClick={() => generateEmailReply()}
                                text="Re-generate"
                              />
                            </div>
                          )}
                        </div>
                      )}
                    </>
                  )}
                  <div ref={scrollRef} />
                </div>
              )}
            </ScrollArea>
          </ResizablePanel>
          {assistantOpen && (
            <>
              <ResizableHandle withHandle />
              <ResizablePanel
                id="assistant"
                order={2}
                defaultSize={30}
                minSize={18}
                collapsible
                collapsedSize={0}
                onCollapse={() => setAssistantOpen(false)}
              >
                <AIWriterV2
                  template={{
                    content: conversation[1]?.content || '',
                    type: 'email',
                    subject: '',
                    send_after: 0,
                    knowledges: [],
                    id: '',
                  }}
                  onUpdateTemplate={handleChange}
                  onClearConversation={handleOnClearConversation}
                  conversationId={conversationId || ''}
                  campaignType={campaign?.playbook || ''}
                  isAIReply
                  setLoadingReply={setLoadingReply}
                  knowledgeReply={aiReplyData}
                  onChangeKnowledge={handleOnChangeKnowledge}
                  onFinished={(subject: string, content: string) => {
                    handleChange('subject', subject);
                    handleChange('content', content);
                    const replyEmail: PreviewEmailWithIntent = {
                      from_email: agent?.data.entity.email || '',
                      to_email: previewContact?.email || '',
                      subject: subject || '',
                      content: content || '',
                      status: 'reply',
                    };
                    setCampaignSupport((prev) => {
                      return {
                        ...prev,
                        preview: {
                          ...prev.preview,
                          [intent]: [
                            prev.preview[intent]?.[0] ?? null,
                            replyEmail,
                          ],
                        },
                      };
                    });
                    setLoadingReply(false);
                  }}
                />
              </ResizablePanel>
            </>
          )}
        </ResizablePanelGroup>
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
