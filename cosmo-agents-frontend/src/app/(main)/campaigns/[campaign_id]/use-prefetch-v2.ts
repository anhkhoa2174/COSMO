import type { EmailIntent } from '@/models/email';
import { useGetAgentQuery } from '@/network/client/agent';
import chatApi, { Message } from '@/network/client/ai-chat';
import { useUser } from '@/hooks/use-user';

import { useGetContactListQuery } from '@/network/client/contact-list';
import {
  GetTemplateData,
  useGenerateDraftTemplateQuery,
  useGetDraftTemplateQuery,
  useGetTemplateQuery,
} from '@/network/client/template';
import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { useCampaign, useCampaignSupport } from './use-campaign';

export function usePrefetchDraft(draftTemplateId: string, intent: EmailIntent) {
  const [campaign, setCampaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const { draftTemplates } = campaignSupport;

  const {
    data: draftTemplate,
    isLoading: isGetDraftTemplateLoading,
    error: getDraftTemplateError,
  } = useGetDraftTemplateQuery(draftTemplateId);
  useEffect(() => {
    if (draftTemplate && !draftTemplates[draftTemplateId]) {
      setCampaignSupport((prev) => ({
        ...prev,
        draftTemplates: {
          ...prev.draftTemplates,
          [draftTemplateId]: draftTemplate.data,
        },
      }));
    }
  }, [draftTemplate, draftTemplateId]);

  const {
    data: generatedDraftTemplate,
    isLoading: isGenerateDraftTemplateLoading,
    error: generateDraftTemplateError,
  } = useGenerateDraftTemplateQuery(campaign.id, intent, !draftTemplateId);
  useEffect(() => {
    if (
      generatedDraftTemplate &&
      !draftTemplates[generatedDraftTemplate.data.id]
    ) {
      setCampaign((prev) => ({
        ...prev,
        draft_templates: [
          ...prev.draft_templates,
          { id: generatedDraftTemplate.data.id, intent },
        ],
      }));
      setCampaignSupport((prev) => ({
        ...prev,
        draftTemplates: {
          ...prev.draftTemplates,
          [generatedDraftTemplate.data.id]: generatedDraftTemplate.data,
        },
      }));
    }
  }, [generatedDraftTemplate, draftTemplateId]);

  return {
    draftTemplate,
    generatedDraftTemplate,
    isLoading: isGetDraftTemplateLoading || isGenerateDraftTemplateLoading,
    error: getDraftTemplateError || generateDraftTemplateError,
  };
}

export function usePrefetchV3(templateId: string) {
  const [campaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const { templates } = campaignSupport;
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(false);

  // Get list contact
  const {
    data: listContact,
    isLoading: isGetListContactLoading,
    error: getListContactError,
  } = useGetContactListQuery(campaign.list_contact_id);
  useEffect(() => {
    if (
      listContact &&
      listContact.data.contacts.length > 0 &&
      campaignSupport.previewContact === undefined
    ) {
      setCampaignSupport((prev) => ({
        ...prev,
        previewContact: listContact.data.contacts[0],
      }));
    }
  }, [listContact]);

  // Get template
  const {
    data: template,
    isLoading: isGetTemplateLoading,
    error: getTemplateError,
  } = useGetTemplateQuery(templateId);
  useEffect(() => {
    if (template && !templates[templateId]) {
      setCampaignSupport((prev) => ({
        ...prev,
        templates: { ...prev.templates, [templateId]: template.data },
      }));
    }
  }, [template, templateId]);

  const getGeneratedTemplate = async (messages: Message[]) => {
    setLoading(true);
    const newConversationId = await chatApi.postCreateConversation();
    if (newConversationId) {
      try {
        const { subject, content, answer } = await chatApi.newChat(
          newConversationId,
          messages
        );
        return { subject, content, answer };
      } catch (error: any) {
        setError(error);
        console.error('Error during generate template:', error);
        toast.error('Failed to generate template');
        return { subject: '', content: '', answer: '' };
      } finally {
        await chatApi.deleteClearConversation(newConversationId);
        setLoading(false);
      }
    }
    return { subject: '', content: '', answer: '' };
  };

  // Get current user
  const { user: currentUser } = useUser();

  // Get agent
  const {
    data: agent,
    isLoading: isGetAgentLoading,
    error: getAgentError,
  } = useGetAgentQuery(campaign.agent_id);

  return {
    getGeneratedTemplate,
    listContact,
    template,
    currentUser,
    agent,
    isLoading:
      isGetListContactLoading ||
      isGetTemplateLoading ||
      isGetAgentLoading ||
      loading,
    error: getListContactError || getTemplateError || getAgentError || error,
  };
}

export function usePrefetch(templateId: string) {
  const [campaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const { templates } = campaignSupport;
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // Get template
  const {
    data: template,
    isLoading: isGetTemplateLoading,
    error: getTemplateError,
  } = useGetTemplateQuery(templateId);
  useEffect(() => {
    if (template && !templates[templateId]) {
      setCampaignSupport((prev) => ({
        ...prev,
        templates: { ...prev.templates, [templateId]: template.data },
      }));
    }
  }, [template, templateId]);

  const getGeneratedTemplate = async (messages: Message[]) => {
    setLoading(true);
    const newConversationId = await chatApi.postCreateConversation();
    if (newConversationId) {
      const templates = campaign?.templates || [];
      const knowledge = `**Context**:\n\n
      campaign_type: ${campaign.playbook}\n\n
      ${
        templates?.length > 0
          ? `You are writing a follow-up to the previous email.\n\n
      Previous Email:\n
      ${templates
        .slice(0, 30)
        .map((template: GetTemplateData) => {
          if (template.content)
            return `
          ${template.type}:
          - Subject: ${template.subject}\n
          - Content: ${template.content}\n`;
        })
        .join('--- \n')}\n\n
      **Task**:\n
      Write a follow-up email. Make sure to:\n
      - Adjust the subject to reflect it is a follow-up.\n
      - Continue the conversation in a polite and engaging way. ABSOLUTELY DO NOT repeat any words or phrases from the 'Previous Email' section, and avoid common conversational fillers like 'I hope', 'I wanted', 'Just reaching out', 'I’m eager', 'I’m excited' or similar sentiments.\n\n
      Return only the new subject and new content.
      `
          : ''
      }`;

      try {
        const { subject, content, answer } = await chatApi.newChat(
          newConversationId,
          knowledge
            ? [
                {
                  content: knowledge,
                  content_type: 'text',
                  role: 'user',
                },
                ...messages,
              ]
            : [...messages]
        );
        return { subject, content, answer };
      } catch (error: any) {
        setError(error);
        console.error('Error during generate template:', error);
        toast.error('Failed to generate template');
        return { subject: '', content: '', answer: '' };
      } finally {
        await chatApi.deleteClearConversation(newConversationId);
        setLoading(false);
      }
    }
    return { subject: '', content: '', answer: '' };
  };

  return {
    template,
    loading,
    getGeneratedTemplate,
    isLoading: isGetTemplateLoading || loading,
    error: getTemplateError || error,
  };
}
