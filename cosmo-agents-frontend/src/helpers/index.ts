import type { User } from '@/models/auth';
import type { Contact } from '@/models/contact';
import { SalesRep } from '@/models/sales-rep';

function getPreviewData(
  contact: Contact,
  user: User,
  agent: any,
  salesReps?: SalesRep[]
) {
  const previewData: any = {};

  if (contact) {
    Object.entries(contact).map(([key, value]) => {
      previewData[`contact_${key}`] = value;
    });
  }

  if (agent) {
    previewData.sender_name = agent.data.entity.name;
    previewData.sender_email = agent.data.entity.email;
  }

  if (user) {
    previewData.organization_name = user.organizations[0].name;
    previewData.organization_company_url = user.organizations[0].company_url;
  }

  if (salesReps && salesReps?.length > 0) {
    previewData.sale_rep_first_name = salesReps[0]?.first_name || '';
    previewData.sale_rep_last_name = salesReps[0]?.last_name || '';
    previewData.sale_rep_email = salesReps[0]?.email || '';
    previewData.sale_rep_calendar_link = salesReps[0]?.calendar_link || '';
  }

  return previewData;
}

function replaceMergeTags(text: string, data: Record<string, any>) {
  return text.replaceAll(/\{([^}]+)\}/g, (match, p1) => {
    return data[p1]
      ? `<span style="background-color: #FF99001A; color: #FF9900;" title="${match}">${data[p1]}</span>`
      : `<span style="background-color: #FF00001A; color: #FF0000;" title="Not available">${match}</span>`;
  });
}

export function getContent(
  content: string,
  contact: any,
  user: any,
  agent: any,
  salesReps?: SalesRep[]
) {
  const previewData = getPreviewData(contact, user, agent, salesReps);

  const signature = agent?.data.entity.signature;
  if (signature) {
    const signatureContent = content.replace('{agent_signature}', signature);
    return replaceMergeTags(signatureContent, previewData);
  }
  return replaceMergeTags(content, previewData);
}

export function extractEmailParts(input: string) {
  const textMatch = input.match(/^(.+?)\n\n<EmailSubject>/s);
  const subjectMatch = input.match(
    /<EmailSubject>\s*(.*?)\s*<\/EmailSubject>/s
  );
  const contentMatch = input.match(
    /<EmailContent>\s*([\s\S]*?)\s*<\/EmailContent>/s
  );
  return {
    text: textMatch?.[1]?.trim() || '',
    subject: subjectMatch?.[1]?.trim() || '',
    content: contentMatch?.[1]?.trim() || '',
  };
}

export function cleanText(input: string, isDoubleQuoted: boolean = true) {
  const cleaned = input.replace(isDoubleQuoted ? /^"(.*)"$/ : /^`(.*)`$/, '$1');
  return cleaned;
}

export function replaceTag(input: string, tagsOptions: Record<string, string>) {
  return input.replace(/\{([^}]+)\}/g, (match, p1) => {
    const isAgent = p1.startsWith('agent_');
    const value = tagsOptions[p1] || 'Not available';
    return `<span className="mx-0.5 p-1 rounded-sm ${isAgent ? 'bg-[#FFE8E9] text-[#FF0000]' : 'bg-[#FFF5EA] text-[#FF9900]'}" title="${match}">${value}</span>`;
  });
}

export async function getBase64(file: File) {
  const arrayBuffer = await file.arrayBuffer();
  const uint8Array = new Uint8Array(arrayBuffer);
  const chunkSize = 16384; // Adjust the chunk size as needed
  let base64String = '';

  for (let i = 0; i < uint8Array.length; i += chunkSize) {
    const chunk = uint8Array.slice(i, i + chunkSize);
    base64String += String.fromCharCode(...chunk);
  }

  return btoa(base64String);
}

export const fakeStream = ({
  step = 10,
  data,
  callback,
  onFinished,
  time = 50,
}: {
  step?: number;
  data: string;
  callback: (data: string) => void;
  onFinished?: () => void;
  time?: number;
}) => {
  let index = 0;
  const initValueContent = data;
  const intervalContent = setInterval(() => {
    if (index <= initValueContent.length) {
      callback(initValueContent.slice(0, index + step));
      index += step;
    } else {
      clearInterval(intervalContent);
      onFinished?.();
    }
  }, time);
};

export function formatAndCapitalize(str: string, prefix?: string): string {
  let result = str;

  if (prefix && result.startsWith(prefix)) {
    result = result.slice(prefix.length);
  }

  result = result.replace(/_/g, ' ');
  return result.charAt(0).toUpperCase() + result.slice(1);
}

export function buildEmailContext({
  organization = null,
  campaignType = '',
  intentType = '',
  subject,
  content,
  words = 60,
}: {
  organization?: any;
  campaignType?: string;
  intentType?: string;
  subject?: string;
  content?: string;
  words?: number;
}) {
  return `**Context**\n\n
${
  organization
    ? `organization: \n\n
- name: ${organization.name}\n\n
- description: ${organization.company_description} / ${(organization.company_targeting_persona || []).join(', ')}\n\n
- company_url: ${organization.company_url}\n\n`
    : ''
}
${campaignType ? `campaign_type: ${campaignType} \n\n` : ''}
intent_type: ${intentType} \n\n
Subject: ${subject || ''} \n\n
Content: ${content || ''} \n\n
---\n\n
Summarize the content in around ${words} words, keeping the original intent and removing any redundant or duplicate parts. Slightly shorter is okay.
`;
}

export function getErrorMessage(
  error: any,
  defaultMessage: string = 'Oops! Something went wrong'
) {
  return error.error?.message || error.message || defaultMessage;
}

export function generateId(): string {
  return Math.random().toString(36).substring(2, 9);
}

export function toCamelCase(input: string): string {
  return input.toLowerCase().replace(/[-_ ]+(\w)/g, (_, c) => c.toUpperCase());
}

export function markdownToPlainText(markdown: string): string {
  return (
    markdown
      // Remove headers
      .replace(/^#{1,6}\s+/gm, '')
      // Remove bold and italic
      .replace(/\*\*(.*?)\*\*/g, '$1')
      .replace(/\*(.*?)\*/g, '$1')
      .replace(/__(.*?)__/g, '$1')
      .replace(/_(.*?)_/g, '$1')
      // Remove strikethrough
      .replace(/~~(.*?)~~/g, '$1')
      // Remove inline code
      .replace(/`(.*?)`/g, '$1')
      // Remove code blocks
      .replace(/```[\s\S]*?```/g, '')
      // Remove links
      .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
      // Remove images
      .replace(/!\[([^\]]*)\]\([^)]+\)/g, '$1')
      // Remove blockquotes
      .replace(/^>\s+/gm, '')
      // Remove horizontal rules
      .replace(/^-{3,}$/gm, '')
      // Remove list markers
      .replace(/^\s*[-*+]\s+/gm, '')
      .replace(/^\s*\d+\.\s+/gm, '')
      // Remove HTML tags
      .replace(/<[^>]*>/g, '')
      // Clean up extra whitespace
      .replace(/\n\s*\n/g, '\n')
      .trim()
  );
}
