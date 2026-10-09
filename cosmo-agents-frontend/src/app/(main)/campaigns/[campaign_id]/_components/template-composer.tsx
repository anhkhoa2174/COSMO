import { AIWriterV2 } from '@/components/ai-writer-v2';
import { PaneHeader } from './pane-header';
import { Eye, PencilLine } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import SalesRepApi from '@/network/client/sales-rep';
import type { SalesRep } from '@/models/sales-rep';
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable';
import { PlateEditor } from '@/components/editor/plate-editor';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { Spinner } from '@/components/ui/spinner';
import { getContent } from '@/helpers';
import { useGetAgentQuery } from '@/network/client/agent';
import { useGetContactListQuery } from '@/network/client/contact-list';
import {
  createTemplate,
  useUpdateTemplateMutation,
  type GetTemplateData,
} from '@/network/client/template';
import _ from 'lodash';
import { TriangleAlert } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { toast } from 'sonner';
import { useCampaign, useCampaignSupport } from '../use-campaign';
import { usePrefetch } from '../use-prefetch-v2';
import { SheetHeader } from './sheet-header';

import chatApi from '@/network/client/ai-chat';
import { useUser } from '@/hooks/use-user';

interface TemplateComposerProps {
  closeSheet: () => void;
  onCreateSuccess?: (template_id: string) => void;
  templateId: string;
  templateIndex: number;
}

export default function TemplateComposer({
  closeSheet,
  onCreateSuccess,
  templateId,
  templateIndex,
}: TemplateComposerProps) {
  const [campaign, setCampaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const { templates } = campaignSupport;
  const [conversationId, setConversationId] = useState('');
  const [template, setTemplate] = useState(templates[templateId] || null);
  const [assistantOpen, setAssistantOpen] = useState(false);

  // The preview resolves {sale_rep_*} tags from this list; without it every
  // one of them renders as unfilled even when a rep is configured.
  const { data: salesRepsRes } = useQuery({
    queryKey: ['sales-reps'],
    queryFn: () => SalesRepApi.search({ filter: {} }),
  });
  // The search endpoint wraps each rep in a row object; only the entity is
  // needed, and getPreviewData reads first_name / calendar_link off it.
  const salesReps = (salesRepsRes?.data?.list ?? []).map(
    (row) => row.entity
  ) as SalesRep[];
  const [templateCreatedId, setTemplateCreatedId] = useState(templateId);
  const [isFocus, setIsFocus] = useState(false);
  const [isGenerating, setIsGenerating] = useState<boolean>(false);

  const { isLoading, error, getGeneratedTemplate } =
    usePrefetch(templateCreatedId);

  const hasChanged = useMemo(
    () => !_.isEqual(templates[templateId], template),
    [templates[templateId], template]
  );

  const {
    mutateAsync: updateTemplate,
    isPending: isUpdateTemplatePending,
    status: updateTemplateStatus,
  } = useUpdateTemplateMutation(templateCreatedId);

  const handleCreateTemplate = async () => {
    if (isGenerating) return;
    setIsGenerating(true);
    setTemplate(null);
    try {
      const generatedTemplate = await getGeneratedTemplate([
        {
          content: `Generate for me the ${campaign.templates.length === 0 ? 'first' : 'follow-up'} outreach email`,
          content_type: 'text',
          role: 'user',
        },
      ]);
      if (!generatedTemplate) {
        return;
      }
      const newItem = {
        type:
          campaign.templates.length === 0
            ? 'First Email'
            : `Follow-up Email ${campaign.templates.length}`,
        subject: generatedTemplate.subject || '',
        content: generatedTemplate.content || '',
        send_after: campaign.templates.length,
        knowledges: [],
      };
      const res = await createTemplate(campaign.id, newItem);
      if (res?.data) {
        setTemplateCreatedId(res.data.id);
        setCampaign((prev) => ({
          ...prev,
          templates: [...prev.templates, { ...res.data }],
        }));
        setCampaignSupport((prev) => ({
          ...prev,
          templates: {
            ...prev.templates,
            [res.data.id]: {
              ...res.data,
            },
          },
        }));
        const templateTemp = {
          ...res.data,
        };
        onCreateSuccess?.(templateTemp.id);
        setTemplate(
          (prev) =>
            ({
              ...prev,
              id: templateTemp.id,
              send_after: templateTemp.send_after,
              knowledges: templateTemp.knowledges,
              type: templateTemp.type,
            }) as GetTemplateData
        );
        // Đặt thẳng nội dung thay vì gõ dần từng nhịp: trình soạn thảo rich
        // text chỉ đọc giá trị lúc khởi tạo, nên nếu gõ dần thì nó chốt lại ở
        // đúng nhịp đầu tiên (10 ký tự — "Hi {contac") và không bao giờ nhận
        // phần còn lại, trong khi khung preview vẫn đọc state nên hiện đủ.
        setTemplate(
          (prev) =>
            ({
              ...prev,
              subject: templateTemp.subject,
              content: templateTemp.content,
            }) as GetTemplateData
        );
      }
    } finally {
      setIsGenerating(false);
    }
  };

  const handleSave = async () => {
    updateTemplate(template as Partial<GetTemplateData>);
  };

  const handleChange = (key: keyof GetTemplateData, value: any) => {
    setTemplate((prev) => ({ ...prev, [key]: value }) as GetTemplateData);
  };

  const handleOnClickAiWriter = async () => {
    setAssistantOpen(!assistantOpen);
    if (!conversationId) {
      const newConversationId = await chatApi.postCreateConversation();
      if (newConversationId) {
        setConversationId(newConversationId);
      }
    } else {
      if (!conversationId) return;
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
  // create template
  useEffect(() => {
    if (!templateId) {
      handleCreateTemplate();
    }
  }, []);

  useEffect(() => {
    if (templateId && templates[templateId]) {
      setTemplate(templates[templateId]);
    }
  }, [templates, templateId]);

  useEffect(() => {
    if (updateTemplateStatus === 'success') {
      setCampaignSupport((prev) => ({
        ...prev,
        templates: { ...prev.templates, [templateId]: template },
      }));
      closeSheet();
    }

    if (updateTemplateStatus === 'error') {
      toast.error('Failed to update template');
    }
  }, [updateTemplateStatus]);

  return (
    <div className="flex h-full flex-col">
      <SheetHeader
        title={template?.type || 'New Template'}
        hasChanged={hasChanged}
        isError={!!error}
        isDraft={campaign.status === 'draft'}
        onSubmit={handleSave}
        onClose={handleOnCloseAiWriter}
        onAssistantOpenChange={handleOnClickAiWriter}
        assistantOpen={assistantOpen}
      />
      <Separator />
      {error ? (
        <p className="inline-flex justify-center gap-2 p-4 text-center text-base text-destructive">
          <TriangleAlert /> {(error as any).error?.message || error.message}
        </p>
      ) : (
        <ResizablePanelGroup
          direction="horizontal"
          // Persisted so a rep's preferred split survives reopening the sheet.
          autoSaveId="cosmo-template-composer-v2"
          className="flex-1"
        >
          <Spinner
            show={isUpdateTemplatePending}
            withOverlay
            label="Saving..."
          />

          <ResizablePanel id="editor" order={1} defaultSize={45} minSize={25}>
            <div className="flex h-full flex-col">
              <PaneHeader
                icon={PencilLine}
                title="Compose"
                hint="Edits appear in the preview as you type"
              />
              <div className="min-h-0 flex-1">
                <ScrollArea className="h-full p-4" type="always">
                  {campaign.status !== 'draft' && (
                    <p className="mb-4 text-orange-500">
                      You can't change the template for a campaign that is not
                      in draft mode
                    </p>
                  )}
                  {isLoading ? (
                    <div className="flex flex-col space-y-4">
                      <Skeleton className="h-6" />
                      <Skeleton className="h-[500px] rounded-lg" />
                      <Skeleton className="h-6" />
                    </div>
                  ) : template ? (
                    <Card className="relative">
                      <CardContent className="space-y-2 p-0 pt-2">
                        {template.type !== 'First Email' && (
                          <>
                            <div className="flex items-center gap-2 px-4">
                              Send if contact does not reply in{' '}
                              <Input
                                type="number"
                                value={template.send_after}
                                onChange={(event) =>
                                  handleChange(
                                    'send_after',
                                    +event.target.value
                                  )
                                }
                                min={1}
                                className="w-16"
                              />{' '}
                              working days
                            </div>
                            <Separator />
                          </>
                        )}
                        <div className="flex items-center gap-2 px-4">
                          <p className="text-muted-foreground">Subject:</p>
                          <Input
                            value={template?.subject || ''}
                            onChange={(event) => {
                              handleChange('subject', event.target.value);
                            }}
                            className="flex-1 border-none bg-transparent shadow-none"
                          />
                        </div>
                        <Separator />
                        <PlateEditor
                          deps={[isGenerating ? template?.content : undefined]}
                          value={template?.content || ''}
                          onChange={(value) => {
                            handleChange('content', value);
                          }}
                          readOnly={campaign.status !== 'draft'}
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
              template={template}
              isLoading={isLoading}
              salesReps={salesReps}
            />
          </ResizablePanel>
          {assistantOpen && !isLoading && (
            <>
              <ResizableHandle withHandle />
              <ResizablePanel
                id="assistant"
                order={3}
                defaultSize={26}
                minSize={16}
                collapsible
                collapsedSize={0}
                onCollapse={() => setAssistantOpen(false)}
              >
                {/* AIWriterV2 carries its own header (mark, title, what it
                    does), so a PaneHeader here would say "AI Writer" twice. */}
                <AIWriterV2
                  template={template || undefined}
                  onUpdateTemplate={handleChange}
                  onClearConversation={handleOnClearConversation}
                  onLoadingChat={setIsGenerating}
                  conversationId={conversationId}
                  campaignType={campaign?.playbook || ''}
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
  salesReps: SalesRep[];
  template: GetTemplateData | null;
  isLoading: boolean;
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
      <PaneHeader
        icon={Eye}
        title="Live preview"
        hint="Exactly what this contact receives"
      />
      <div className="min-h-0 flex-1">
        <ScrollArea className="h-full bg-[#F7F7F7] p-4" type="always">
          {isListContactLoading ? (
            <p className="p-4">Loading contact list...</p>
          ) : (
            <div className="mb-4 flex items-center gap-2">
              <p className="shrink-0 text-[0.85rem] text-muted-foreground">
                Preview as
              </p>
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
                  <Markdown
                    remarkPlugins={[remarkGfm]}
                    rehypePlugins={[rehypeRaw]}
                    className="text-base font-medium"
                  >
                    {getContent(
                      template?.subject || '',
                      previewContact,
                      currentUser,
                      agent,
                      salesReps
                    )}
                  </Markdown>
                  <Separator />
                  <div className="text-base leading-loose">
                    <Markdown
                      remarkPlugins={[remarkGfm]}
                      rehypePlugins={[rehypeRaw]}
                    >
                      {getContent(
                        template?.content || '',
                        previewContact,
                        currentUser,
                        agent,
                        salesReps
                      )}
                    </Markdown>
                  </div>
                </CardContent>
              </Card>
              {/* Two colours with no key left readers guessing what red meant. */}
              <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 px-1 text-[0.8rem] text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  <span
                    className="inline-block size-2.5 rounded-sm"
                    style={{ backgroundColor: '#FF9900' }}
                  />
                  Filled in for this contact
                </span>
                <span className="flex items-center gap-1.5">
                  <span
                    className="inline-block size-2.5 rounded-sm"
                    style={{ backgroundColor: '#FF0000' }}
                  />
                  No data — will send blank
                </span>
              </div>

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
