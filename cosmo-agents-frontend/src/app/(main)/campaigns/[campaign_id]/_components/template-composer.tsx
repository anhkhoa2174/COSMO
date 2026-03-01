import { AIWriterV2 } from '@/components/ai-writer-v2';
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
import { fakeStream, getContent } from '@/helpers';
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
    setTemplate(null);
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
      fakeStream({
        data: templateTemp.subject,
        callback: (data) => {
          setTemplate(
            (prev) =>
              ({
                ...prev,
                subject: data,
              }) as GetTemplateData
          );
        },
      });
      fakeStream({
        data: templateTemp.content,
        callback: (data) => {
          setTemplate(
            (prev) =>
              ({
                ...prev,
                content: data,
              }) as GetTemplateData
          );
        },
      });
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
      />
      <Separator />
      {error ? (
        <p className="inline-flex justify-center gap-2 p-4 text-center text-base text-destructive">
          <TriangleAlert /> {(error as any).error?.message || error.message}
        </p>
      ) : (
        <div className="flex flex-1">
          <Spinner
            show={isUpdateTemplatePending}
            withOverlay
            label="Saving..."
          />
          {assistantOpen && !isLoading && (
            <div className="w-80 flex-shrink-0">
              <AIWriterV2
                template={template || undefined}
                onUpdateTemplate={handleChange}
                onClearConversation={handleOnClearConversation}
                onLoadingChat={setIsGenerating}
                conversationId={conversationId}
                campaignType={campaign?.playbook || ''}
              />
            </div>
          )}
          <div className="flex flex-1 flex-col">
            <div className="h-96 flex-grow">
              <ScrollArea className="h-full p-4" type="always">
                {campaign.status !== 'draft' && (
                  <p className="mb-4 text-orange-500">
                    You can't change the template for a campaign that is not in
                    draft mode
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
                                handleChange('send_after', +event.target.value)
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
                        deps={[
                          isFocus && !isGenerating ? '' : template?.content,
                        ]}
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
          <TemplatePreview template={template} isLoading={isLoading} />
        </div>
      )}
    </div>
  );
}

function TemplatePreview({
  template,
  isLoading,
}: {
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
    <div className="flex flex-1 flex-col">
      <div className="h-96 flex-grow">
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
                      <span className="text-info">
                        {contact.name}
                      </span>{' '}
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
                      agent
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
                        agent
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
