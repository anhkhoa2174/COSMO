'use client';

import { Badge } from '@/components/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { ContactStatus } from '@/models/contact';
import { CheckCircle2, AlertCircle } from 'lucide-react';

interface ContactStatusBadgeProps {
  status?: ContactStatus;
  missingFields?: string[];
  showTooltip?: boolean;
}

const FIELD_LABELS: Record<string, string> = {
  name: 'Name',
  company: 'Company',
  job_title: 'Job Title',
  industry: 'Industry',
  contact_channel: 'Contact Channel',
};

export function ContactStatusBadge({
  status,
  missingFields,
  showTooltip = true,
}: ContactStatusBadgeProps) {
  const isReady = status === 'ready';

  const badge = (
    <Badge
      variant={isReady ? 'success' : 'warning'}
      className="gap-1 capitalize"
    >
      {isReady ? (
        <CheckCircle2 className="h-3 w-3" />
      ) : (
        <AlertCircle className="h-3 w-3" />
      )}
      {status || 'pending'}
    </Badge>
  );

  if (!showTooltip || isReady || !missingFields?.length) {
    return badge;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{badge}</TooltipTrigger>
        <TooltipContent>
          <div className="text-sm">
            <p className="mb-1 font-medium">Missing fields:</p>
            <ul className="list-inside list-disc">
              {missingFields.map((field) => (
                <li key={field}>{FIELD_LABELS[field] || field}</li>
              ))}
            </ul>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
