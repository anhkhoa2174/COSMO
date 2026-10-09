'use client';

import type { MessagePart } from '@/types/daily-actions';
import { TextBlock } from './parts/text-block';
import { AgentReasoningBlock } from './parts/agent-reasoning-block';
import { CategoryBadges } from './parts/category-badges';
import { ActionCategorySection } from './parts/action-category-section';
import { AlertCard } from './parts/alert-card';
import { SummaryBlock } from './parts/summary-block';
import { PipelineSummaryCard } from './pipeline-summary-card';
import { ContactListResult } from './parts/contact-list-result';

interface MessagePartRendererProps {
  part: MessagePart;
  messageId: string;
}

export function MessagePartRenderer({
  part,
  messageId,
}: MessagePartRendererProps) {
  switch (part.type) {
    case 'text':
      return <TextBlock text={part.text} />;
    case 'agent-reasoning':
      return (
        <AgentReasoningBlock text={part.text} memoryRefs={part.memory_refs} />
      );
    case 'category-badges':
      return <CategoryBadges badges={part.badges} />;
    case 'action-category':
      return <ActionCategorySection category={part.category} />;
    case 'alert':
      return <AlertCard alert={part.alert} />;
    case 'summary':
      return <SummaryBlock summary={part.summary} />;
    case 'pipeline-summary':
      return <PipelineSummaryCard summary={part.pipeline_summary} />;
    case 'contact-list':
      return <ContactListResult data={part.contact_list} />;
    default:
      return null;
  }
}
