import { MainButton } from '@/components/buttons/main-button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Spinner } from '@/components/ui/spinner';
import { CampaignConfig } from '@/models/campaign';
import type { EmailIntent } from '@/models/email';
import CampaignApi from '@/network/client/campaign';
import {
  GetTemplateData,
  useUpdateTemplateMutation,
} from '@/network/client/template';
import { useMutation } from '@tanstack/react-query';
import _ from 'lodash';
import { Sparkles, TriangleAlert } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { useCampaign, useCampaignSupport } from '../../use-campaign';
import { usePrefetchDraft } from '../../use-prefetch-v2';

import { AIWriterV2 } from '@/components/ai-writer-v2';
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable';
import { PlateEditor } from '@/components/editor/plate-editor';
import { Card, CardContent } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { getContent } from '@/helpers';
import { SalesRep } from '@/models/sales-rep';
import { useGetAgentQuery } from '@/network/client/agent';
import chatApi from '@/network/client/ai-chat';
import { useGetContactListQuery } from '@/network/client/contact-list';
import SalesRepApi from '@/network/client/sales-rep';
import { useUser } from '@/hooks/use-user';
import { useRouter } from 'next/navigation';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import SalesReps from '../sales-reps';

type ConfigPayload = {
  sale_rep_ids: string[];
  content: string;
};

export default function DraftEmail({
  intentType,
  closeSheet,
  onSuccess,
}: {
  intentType: EmailIntent;
  closeSheet: () => void;
  onSuccess: () => void;
}) {
  const defaultOption: CampaignConfig<ConfigPayload> = {
    intent_type: intentType,
    who: 'Draft an email',
    payload: { sale_rep_ids: [], content: '' },
  };
  const [campaign, setCampaign] = useCampaign();
  const router = useRouter();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const draftTemplateId =
    campaign.draft_templates.find((d) => d.intent === intentType)?.id || '';

  const { isLoading, error, generatedDraftTemplate } = usePrefetchDraft(
    draftTemplateId,
    intentType
  );
  const { draftTemplates } = campaignSupport;
  const [conversationId, setConversationId] = useState('');
  const [assistantOpen, setAssistantOpen] = useState(false);
  const generatedTemplateId = generatedDraftTemplate?.data.id || '';
  const [salesRepsData, setSalesRepsData] = useState<SalesRep[]>([]);
  const [loadingSalesReps, setLoadingSalesReps] = useState(false);
  const [isFocus, setIsFocus] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);

  const existOption = campaign.cmetadata.config.find(
    (c) => c.intent_type === intentType
  );
  const [option, setOption] = useState<CampaignConfig<ConfigPayload>>(
    existOption || defaultOption
  );

  const {
    mutateAsync: assignCampaignIntent,
    isPending: isAssignCampaignIntentPending,
  } = useMutation({
    mutationFn: () => {
      const copy = [...(campaign.cmetadata.config || [])];
      const payload = {
        config: copy.filter((c) => c.intent_type !== intentType).concat(option),
      };
      return CampaignApi.assign(campaign.id, payload);
    },
    onSuccess: () => {
      const copy = [...(campaign.cmetadata.config || [])];
      const payload = {
        config: copy.filter((c) => c.intent_type !== intentType).concat(option),
      };
      setCampaign((prev) => ({
        ...prev,
        cmetadata: { ...prev.cmetadata, ...payload },
      }));
      onSuccess();
      // closeSheet();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    },
  });

  const templateId =
    draftTemplates[draftTemplateId]?.template_id ||
    generatedDraftTemplate?.data.template_id ||
    '';
  const hasChanged = useMemo(
    () => !existOption || !_.isEqual(existOption, option),
    [existOption, option]
  );

  useEffect(() => {
    if (draftTemplateId && draftTemplates[draftTemplateId]) {
      setOption({
        ...option,
        payload: {
          ...option.payload,
          content: draftTemplates[draftTemplateId].template.content,
        },
      });
    }
    if (
      !draftTemplateId &&
      generatedTemplateId &&
      draftTemplates[generatedTemplateId]
    ) {
      setOption({
        ...option,
        payload: {
          ...option.payload,
          content: draftTemplates[generatedTemplateId].template.content,
        },
      });
    }
  }, [draftTemplates, draftTemplateId]);

  /* handle update template */
  const {
    mutateAsync: updateTemplate,
    isPending: isUpdateTemplatePending,
    status: updateTemplateStatus,
  } = useUpdateTemplateMutation(templateId);

  useEffect(() => {
    if (updateTemplateStatus === 'success') {
      setCampaignSupport((prev) => ({
        ...prev,
        draftTemplates: {
          ...prev.draftTemplates,
          [draftTemplateId]: prev.draftTemplates[draftTemplateId]
            ? {
                ...prev.draftTemplates[draftTemplateId],
                template: {
                  ...prev.draftTemplates[draftTemplateId]!.template,
                  content: option.payload.content,
                },
              }
            : null,
        },
      }));
      closeSheet();
    }

    if (updateTemplateStatus === 'error') {
      toast.error('Failed to update template');
    }
  }, [updateTemplateStatus]);

  const handleSave = async () => {
    await assignCampaignIntent();
    await updateTemplate({ content: option.payload.content });
  };

  const handleChange = (key: keyof GetTemplateData, value: any) => {
    setOption((prev) => ({
      ...prev,
      payload: { ...prev.payload, [key]: value },
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
    setAssistantOpen(false);
    closeSheet();
    await chatApi.deleteClearConversation(conversationId);
  };

  const handleOnClearConversation = async (isClose = false) => {
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

  const handleOnAddASalesReps = () => {
    router.push(`/sales-reps?isModalOpen=true`);
  };

  const handleOnChangeSalesReps = (values: string[]) => {
    setOption((prev) => ({
      ...prev,
      payload: { ...prev.payload, sale_rep_ids: values },
    }));
  };

  useEffect(() => {
    async function fetchSalesReps() {
      setLoadingSalesReps(true);
      try {
        const response = await SalesRepApi.search({ filter: {} });
        setSalesRepsData(
          response.data.list.map((l) => ({
            first_name: l.entity.first_name,
            last_name: l.entity.last_name,
            calendar_link: l.entity.calendar_link,
            email: l.entity.email,
            id: l.entity.id,
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
    <div className="flex items-center justify-between gap-2 p-4">
      <p className="text-lg font-bold">Draft an email</p>
      <div className="flex items-center gap-2">
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
            data={salesRepsData}
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

  return (
    <div className="flex h-full flex-col">
      {renderHeader()}
      <Separator />
      {error ? (
        <p className="inline-flex justify-center gap-2 p-4 text-center text-base text-destructive">
          <TriangleAlert /> {(error as any).error?.message || error.message}
        </p>
      ) : (
        <ResizablePanelGroup
          direction="horizontal"
          autoSaveId="cosmo-draft-email-v2"
          className="flex-1"
        >
          <Spinner
            show={isAssignCampaignIntentPending || isUpdateTemplatePending}
            withOverlay
            label="Saving..."
          />

          <ResizablePanel id="editor" order={1} defaultSize={45} minSize={25}>
            <div className="flex h-full flex-col">
              <div className="min-h-0 flex-1">
                <ScrollArea className="h-full p-4" type="always">
                  {isLoading ? (
                    <div className="flex flex-col space-y-4">
                      <Skeleton className="h-[500px] rounded-lg" />
                      <Skeleton className="h-6" />
                    </div>
                  ) : option.payload ? (
                    <Card className="relative">
                      <CardContent className="space-y-2 p-0 pt-2">
                        <PlateEditor
                          deps={[
                            isGenerating ? option.payload.content : undefined,
                          ]}
                          value={option.payload.content}
                          onChange={(value) => {
                            handleChange('content', value);
                          }}
                          onFocus={() => setIsFocus(true)}
                          onBlur={() => setIsFocus(false)}
                        />
                      </CardContent>
                    </Card>
                  ) : null}
                </ScrollArea>
              </div>
            </div>
          </ResizablePanel>

          <ResizableHandle withHandle />

          <ResizablePanel id="preview" order={2} defaultSize={31} minSize={20}>
            <TemplatePreview
              template={option.payload as any}
              isLoading={isLoading}
              salesReps={salesRepsData}
            />
          </ResizablePanel>
          {assistantOpen && (
            <>
              <ResizableHandle withHandle />
              <ResizablePanel
                id="assistant"
                order={3}
                defaultSize={24}
                minSize={16}
                collapsible
                collapsedSize={0}
                onCollapse={() => setAssistantOpen(false)}
              >
                <AIWriterV2
                  template={
                    draftTemplates[draftTemplateId]?.template as GetTemplateData
                  }
                  intentType={intentType}
                  onUpdateTemplate={handleChange}
                  onClearConversation={handleOnClearConversation}
                  conversationId={conversationId}
                  campaignType={campaign?.playbook || ''}
                  onLoadingChat={setIsGenerating}
                />
              </ResizablePanel>
            </>
          )}
        </ResizablePanelGroup>
      )}
    </div>
  );
}

function TemplatePreview({
  template,
  isLoading,
  salesReps,
}: {
  template: GetTemplateData | null;
  isLoading: boolean;
  salesReps: SalesRep[];
}) {
  const [campaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const { previewContact } = campaignSupport;

  const {
    data: listContact,
    isLoading: isListContactLoading,
    error: listContactError,
  } = useGetContactListQuery(campaign.list_contact_id);
  const contacts = listContact?.data.contacts || [];
  useEffect(() => {
    if (contacts.length > 0) {
      setCampaignSupport((prev) => ({ ...prev, previewContact: contacts[0] }));
    }
  }, [contacts]);

  const { user: currentUser } = useUser();
  // const _currentUser = currentUser?.data.organizations[0];
  const { data: agent } = useGetAgentQuery(campaign.agent_id);
  // const _agent = agent?.data.entity;

  if (!campaign.list_contact_id) {
    return (
      <p className="p-2 text-sm text-muted-foreground">
        No contact list selected. Please select a contact list to preview the
        template.
      </p>
    );
  }

  if (listContactError) {
    return (
      <p className="text-center text-sm text-destructive">
        Error loading contact list. Please try again.
      </p>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="min-h-0 flex-1">
        <ScrollArea className="h-full bg-[#F7F7F7] p-4" type="always">
          {isListContactLoading ? (
            <p className="p-4">Loading contact list...</p>
          ) : (
            <div className="mb-4 flex items-center gap-2">
              <p className="text-base font-semibold">Preview as</p>
              <Select
                value={previewContact?.id}
                onValueChange={(contact_id) => {
                  const _contact = contacts.find((c) => c.id === contact_id);
                  if (_contact) {
                    setCampaignSupport((prev) => ({
                      ...prev,
                      previewContact: _contact,
                    }));
                  }
                }}
              >
                <SelectTrigger className="flex-1 bg-background">
                  <SelectValue placeholder="Select a contact" />
                </SelectTrigger>
                <SelectContent>
                  {contacts.map((contact) => (
                    <SelectItem key={contact.id} value={contact.id}>
                      <span className="text-info">{contact.name}</span>{' '}
                      <span className="text-muted-foreground">
                        &lt;{contact.profile?.email || contact.email || '-'}&gt;
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
          {isLoading ? (
            <Skeleton className="h-[300px] w-full rounded-lg" />
          ) : template ? (
            <>
              <Card>
                <CardContent className="space-y-2 py-4">
                  <div className="text-base leading-loose">
                    <Markdown
                      remarkPlugins={[remarkGfm]}
                      rehypePlugins={[rehypeRaw]}
                    >
                      {getContent(
                        template.content,
                        previewContact,
                        currentUser,
                        agent,
                        salesReps
                      )}
                    </Markdown>
                  </div>
                </CardContent>
              </Card>
              {!agent && (
                <p className="p-2 text-sm text-orange-500">
                  Select an AI inbox to preview the template with relevant merge
                  tags.
                </p>
              )}
            </>
          ) : null}
        </ScrollArea>
      </div>
    </div>
  );
}
