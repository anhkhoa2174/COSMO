'use client';

import { useAtom } from 'jotai';
import { AlertTriangle, ExternalLink, Pencil } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { EnrichmentActionData } from '@/types/daily-actions';
import { activePanelAtom } from '@/stores/daily-actions';

interface EnrichmentCardContentProps {
  data: EnrichmentActionData;
  contactId?: string;
}

export function EnrichmentCardContent({
  data,
  contactId,
}: EnrichmentCardContentProps) {
  const [, setActivePanel] = useAtom(activePanelAtom);

  const handleEnrich = () => {
    if (contactId) {
      setActivePanel({ contactId });
    }
  };

  return (
    <div className="space-y-2">
      {/* Missing fields */}
      <div className="flex flex-wrap gap-1.5">
        {data.missing_fields.map((field) => (
          <Badge
            key={field}
            variant="outline"
            className="border-amber-500/[0.2] bg-amber-500/[0.1] text-xs text-amber-600 dark:text-amber-400"
          >
            <AlertTriangle className="mr-1 h-3 w-3" />
            {field}
          </Badge>
        ))}
      </div>

      {/* Quality impact */}
      <p className="text-xs text-muted-foreground/80">{data.quality_impact}</p>

      {/* Suggested sources */}
      {data.suggested_sources && data.suggested_sources.length > 0 && (
        <div className="space-y-1">
          <p className="text-xs font-medium text-muted-foreground">Suggested Sources</p>
          {data.suggested_sources.map((source, i) => (
            <div key={i} className="flex items-center gap-2 text-xs">
              <span className="text-muted-foreground/80">{source.field}:</span>
              {source.url ? (
                <a
                  href={source.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-indigo-600 dark:text-indigo-400 hover:underline"
                >
                  {source.source}
                  <ExternalLink className="h-3 w-3" />
                </a>
              ) : (
                <span className="text-muted-foreground">{source.source}</span>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Enrich button */}
      {contactId && (
        <Button
          variant="outline"
          size="sm"
          onClick={handleEnrich}
          className="mt-1 border-border bg-transparent text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          <Pencil className="mr-1.5 h-3.5 w-3.5" />
          Bổ sung thông tin
        </Button>
      )}
    </div>
  );
}
