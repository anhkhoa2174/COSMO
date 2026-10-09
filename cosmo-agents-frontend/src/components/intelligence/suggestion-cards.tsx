'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { AlertCircle, Lightbulb, ArrowRight, X } from 'lucide-react';
import type { ResearchSuggestion } from '@/network/client/intelligence';
import { useState } from 'react';

interface SuggestionCardsProps {
  suggestions: ResearchSuggestion[];
  onAction: (suggestion: ResearchSuggestion) => void;
  onDismiss?: (suggestion: ResearchSuggestion) => void;
}

export function SuggestionCards({
  suggestions,
  onAction,
  onDismiss,
}: SuggestionCardsProps) {
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());

  const handleDismiss = (suggestion: ResearchSuggestion) => {
    const key = `${suggestion.category}_${suggestion.data_point}`;
    setDismissed(new Set([...dismissed, key]));
    onDismiss?.(suggestion);
  };

  const visibleSuggestions = suggestions.filter((s) => {
    const key = `${s.category}_${s.data_point}`;
    return !dismissed.has(key);
  });

  if (visibleSuggestions.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Lightbulb className="h-5 w-5 text-yellow-500" />
        <h3 className="font-semibold">Suggested Research</h3>
        <Badge variant="secondary">{visibleSuggestions.length}</Badge>
      </div>

      <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
        {visibleSuggestions.map((suggestion, idx) => {
          const priorityColor =
            suggestion.priority === 'high'
              ? 'text-red-500 border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950'
              : suggestion.priority === 'medium'
                ? 'text-yellow-500 border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-950'
                : 'text-blue-500 border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950';

          return (
            <Card
              key={idx}
              className={`relative transition-all hover:shadow-md ${priorityColor}`}
            >
              {/* Dismiss button */}
              {onDismiss && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute right-2 top-2 h-6 w-6"
                  onClick={() => handleDismiss(suggestion)}
                >
                  <X className="h-4 w-4" />
                </Button>
              )}

              <CardHeader className="pb-3">
                <div className="flex items-start gap-2">
                  <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
                  <div className="flex-1 space-y-1">
                    <CardTitle className="text-sm font-medium">
                      {suggestion.category}
                    </CardTitle>
                    <p className="text-xs opacity-80">
                      Missing: {suggestion.data_point}
                    </p>
                  </div>
                </div>
              </CardHeader>

              <CardContent className="space-y-3 pt-0">
                <div className="space-y-2 text-xs">
                  <div>
                    <span className="font-semibold">Why Important:</span>
                    <p className="mt-1 opacity-90">
                      {suggestion.why_important}
                    </p>
                  </div>
                  <div>
                    <span className="font-semibold">Where to Find:</span>
                    <p className="mt-1 opacity-90">
                      {suggestion.where_to_find}
                    </p>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <Badge
                    variant={
                      suggestion.priority === 'high'
                        ? 'destructive'
                        : suggestion.priority === 'medium'
                          ? 'default'
                          : 'secondary'
                    }
                    className="text-xs"
                  >
                    {suggestion.priority} priority
                  </Badge>

                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onAction(suggestion)}
                    className="h-7 gap-1 text-xs"
                  >
                    Add Finding
                    <ArrowRight className="h-3 w-3" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
