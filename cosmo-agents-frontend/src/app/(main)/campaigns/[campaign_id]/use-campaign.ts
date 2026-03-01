import { atom, useAtom } from 'jotai';
import type { Campaign } from '@/models/campaign';
import type { Contact } from '@/models/contact';
import type { EmailIntent } from '@/models/email';
import type { Member } from '@/models/organization';
import type { SalesRep } from '@/models/sales-rep';
import type { GetDraftTemplateData, GetTemplateData } from '@/network/client/template';

type CampaignSupport = {
  previewContact?: Contact;
  templates: Record<string, GetTemplateData | null>;
  draftTemplates: Record<string, GetDraftTemplateData | null>;
  members: Member[];
  salesReps: Omit<SalesRep, 'created_at' | 'updated_at'>[];
  preview: Record<EmailIntent, PreviewEmailWithIntent[]>;
  conversations: PreviewEmailWithIntent[];
};

export interface PreviewEmail {
  content: string;
  from_email: string;
  to_email: string;
  subject: string;
  status: string;
}

export interface PreviewEmailWithIntent extends PreviewEmail {
  intent?: string;
}

export const campaignInitial: Campaign = {
  id: '',
  user_id: '',
  playbook: 'revive_dormant_leads',
  inbound_lead_forms: [],
  name: '',
  list_contact_id: null,
  schedule: null,
  organization_id: '',
  created_at: '',
  updated_at: '',
  status: 'draft',
  agent_id: null,
  notifications: [],
  templates: [],
  draft_templates: [],
  cmetadata: {
    config: [],
  },
};

export const campaignSupportInitial: CampaignSupport = {
  templates: {},
  draftTemplates: {},
  members: [],
  salesReps: [],
  preview: {
    Interested: [],
    'Not interested': [],
    'Request for pricing': [],
    'Request for information': [],
    'Do not contact': [],
    'Out of office': [],
    'Unknown intent': [],
  },
  conversations: [],
};

const campaignAtom = atom<Campaign>(campaignInitial);
const campaignSupportAtom = atom<CampaignSupport>(campaignSupportInitial);

export function useCampaign() {
  return useAtom(campaignAtom);
}

export function useCampaignSupport() {
  return useAtom(campaignSupportAtom);
}
