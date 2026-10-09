'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Calendar,
  Lightbulb,
  AlertTriangle,
  MessageSquare,
  Loader2,
} from 'lucide-react';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';

interface MeetingBrief {
  last_interactions: Array<{
    type: string;
    date: string;
    summary: string;
    sentiment?: string;
  }>;
  confirmed_facts: Array<{
    category: string;
    facts: string[];
  }>;
  segments: Array<{
    name: string;
    fit_score: number;
  }>;
  talking_points: string[];
  discovery_questions: string[];
  risk_flags: string[];
}

interface MeetingIntelligenceProps {
  contactId: string;
  contactName: string;
  onGenerateBrief?: () => Promise<MeetingBrief>;
}

export function MeetingIntelligence({
  contactId,
  contactName,
  onGenerateBrief,
}: MeetingIntelligenceProps) {
  const [brief, setBrief] = useState<MeetingBrief | null>(null);
  const [activeTab, setActiveTab] = useState<
    'overview' | 'talking-points' | 'questions' | 'risks'
  >('overview');

  const generateBriefMutation = useMutation({
    mutationFn: async () => {
      if (onGenerateBrief) {
        return await onGenerateBrief();
      }
      // Default: call API endpoint (to be implemented)
      throw new Error('Meeting brief generation not implemented');
    },
    onSuccess: (data) => {
      setBrief(data);
      toast.success('Meeting brief generated!');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to generate meeting brief');
    },
  });

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            <CardTitle>Meeting Intelligence</CardTitle>
          </div>
          <Button
            onClick={() => generateBriefMutation.mutate()}
            disabled={generateBriefMutation.isPending}
            size="sm"
          >
            {generateBriefMutation.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <Lightbulb className="mr-2 h-4 w-4" />
                Generate Pre-Meeting Brief
              </>
            )}
          </Button>
        </div>
      </CardHeader>

      {brief && (
        <CardContent className="space-y-4">
          {/* Tabs */}
          <div className="flex gap-2 border-b">
            {[
              { key: 'overview', label: 'Overview' },
              { key: 'talking-points', label: 'Talking Points' },
              { key: 'questions', label: 'Discovery Questions' },
              { key: 'risks', label: 'Risk Flags' },
            ].map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key as typeof activeTab)}
                className={`px-4 py-2 text-sm font-medium transition-colors ${
                  activeTab === tab.key
                    ? 'border-b-2 border-primary text-primary'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          <ScrollArea className="h-[400px]">
            {/* Overview Tab */}
            {activeTab === 'overview' && (
              <div className="space-y-4 pr-4">
                {/* Contact Name */}
                <div>
                  <h3 className="mb-2 font-semibold">Meeting With</h3>
                  <p className="text-lg">{contactName}</p>
                </div>

                <Separator />

                {/* Last Interactions */}
                <div>
                  <h3 className="mb-3 font-semibold">
                    Recent Interactions ({brief.last_interactions.length})
                  </h3>
                  <div className="space-y-2">
                    {brief.last_interactions.map((interaction, idx) => (
                      <div
                        key={idx}
                        className="rounded-lg border bg-card p-3 text-sm"
                      >
                        <div className="mb-1 flex items-center justify-between">
                          <Badge variant="outline">{interaction.type}</Badge>
                          <span className="text-xs text-muted-foreground">
                            {new Date(interaction.date).toLocaleDateString()}
                          </span>
                        </div>
                        <p className="text-muted-foreground">
                          {interaction.summary}
                        </p>
                        {interaction.sentiment && (
                          <Badge variant="secondary" className="mt-2 text-xs">
                            Sentiment: {interaction.sentiment}
                          </Badge>
                        )}
                      </div>
                    ))}
                  </div>
                </div>

                <Separator />

                {/* Confirmed Facts */}
                <div>
                  <h3 className="mb-3 font-semibold">Confirmed Facts</h3>
                  <div className="space-y-3">
                    {brief.confirmed_facts.map((category, idx) => (
                      <div key={idx}>
                        <p className="mb-2 text-sm font-medium">
                          {category.category}
                        </p>
                        <ul className="list-inside list-disc space-y-1 text-sm text-muted-foreground">
                          {category.facts.map((fact, factIdx) => (
                            <li key={factIdx}>{fact}</li>
                          ))}
                        </ul>
                      </div>
                    ))}
                  </div>
                </div>

                <Separator />

                {/* Segments */}
                <div>
                  <h3 className="mb-3 font-semibold">Segment Fit</h3>
                  <div className="space-y-2">
                    {brief.segments.map((segment, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between rounded-lg border p-2"
                      >
                        <span className="text-sm">{segment.name}</span>
                        <Badge
                          variant={
                            segment.fit_score >= 80
                              ? 'default'
                              : segment.fit_score >= 60
                                ? 'secondary'
                                : 'outline'
                          }
                        >
                          {segment.fit_score}% fit
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* Talking Points Tab */}
            {activeTab === 'talking-points' && (
              <div className="space-y-3 pr-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <MessageSquare className="h-4 w-4" />
                  <p className="text-sm">
                    Key topics to discuss based on profile and interactions
                  </p>
                </div>
                <div className="space-y-2">
                  {brief.talking_points.map((point, idx) => (
                    <div
                      key={idx}
                      className="flex gap-3 rounded-lg border bg-card p-3"
                    >
                      <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-primary text-xs text-primary-foreground">
                        {idx + 1}
                      </span>
                      <p className="text-sm">{point}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Discovery Questions Tab */}
            {activeTab === 'questions' && (
              <div className="space-y-3 pr-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Lightbulb className="h-4 w-4" />
                  <p className="text-sm">
                    Questions to uncover gaps and confirm assumptions
                  </p>
                </div>
                <div className="space-y-2">
                  {brief.discovery_questions.map((question, idx) => (
                    <div
                      key={idx}
                      className="flex gap-3 rounded-lg border bg-card p-3"
                    >
                      <span className="text-lg">❓</span>
                      <p className="text-sm">{question}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Risk Flags Tab */}
            {activeTab === 'risks' && (
              <div className="space-y-3 pr-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <AlertTriangle className="h-4 w-4" />
                  <p className="text-sm">
                    Potential concerns or objections to be aware of
                  </p>
                </div>
                {brief.risk_flags.length > 0 ? (
                  <div className="space-y-2">
                    {brief.risk_flags.map((risk, idx) => (
                      <div
                        key={idx}
                        className="flex gap-3 rounded-lg border border-yellow-200 bg-yellow-50 p-3 dark:border-yellow-800 dark:bg-yellow-950"
                      >
                        <AlertTriangle className="h-5 w-5 flex-shrink-0 text-yellow-600 dark:text-yellow-400" />
                        <p className="text-sm text-yellow-900 dark:text-yellow-100">
                          {risk}
                        </p>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rounded-lg border bg-muted p-4 text-center text-sm text-muted-foreground">
                    No risk flags detected
                  </div>
                )}
              </div>
            )}
          </ScrollArea>
        </CardContent>
      )}

      {!brief && !generateBriefMutation.isPending && (
        <CardContent>
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Calendar className="mb-4 h-12 w-12 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Generate a pre-meeting brief to prepare for your conversation
            </p>
          </div>
        </CardContent>
      )}
    </Card>
  );
}
