'use client';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { Sparkles, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import IntelligenceApi, {
  type ContactEnrichmentResponse,
  type ResearchSuggestion,
  type ValidateInsightRequest,
} from '@/network/client/intelligence';
import { toast } from 'sonner';
import { AddFindingDialog } from './add-finding-dialog';

interface ContactEnrichmentButtonProps {
  contactId: string;
  variant?: 'default' | 'outline' | 'ghost';
  size?: 'default' | 'sm' | 'lg' | 'icon';
  showLabel?: boolean;
  onEnrichmentComplete?: () => void;
}

export function ContactEnrichmentButton({
  contactId,
  variant = 'outline',
  size = 'sm',
  showLabel = true,
  onEnrichmentComplete,
}: ContactEnrichmentButtonProps) {
  const [showDialog, setShowDialog] = useState(false);
  const [enrichmentData, setEnrichmentData] =
    useState<ContactEnrichmentResponse | null>(null);
  const [addFindingDialogOpen, setAddFindingDialogOpen] = useState(false);
  const [selectedSuggestion, setSelectedSuggestion] = useState<ResearchSuggestion | null>(null);
  const [forceRefresh, setForceRefresh] = useState(false);

  const enrichMutation = useMutation({
    mutationFn: (forceRefresh: boolean) =>
      IntelligenceApi.enrichContact(contactId, forceRefresh),
    onSuccess: (response) => {
      setEnrichmentData(response.data);
      setShowDialog(true);
      toast.success('Contact enriched successfully!');
      onEnrichmentComplete?.();
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to enrich contact');
    },
  });

  const validateInsightMutation = useMutation({
    mutationFn: (payload: ValidateInsightRequest) =>
      IntelligenceApi.validateInsight(contactId, payload),
    onSuccess: (_, vars) => {
      setEnrichmentData((prev) => {
        if (!prev) return prev;
        const updated = { ...prev, ai_insights: { ...prev.ai_insights } };

        if (vars.insight_type === 'pain_point') {
          updated.ai_insights.suspected_pain_points = prev.ai_insights.suspected_pain_points.filter(
            (p) => p.pain_point !== vars.insight_text
          );
        } else if (vars.insight_type === 'goal') {
          updated.ai_insights.suspected_goals = prev.ai_insights.suspected_goals.filter(
            (g) => g.goal !== vars.insight_text
          );
        }
        return updated;
      });
      toast.success(
        vars.validation === 'confirmed'
          ? 'Insight confirmed and moved to confirmed facts'
          : 'Insight rejected and removed'
      );
      onEnrichmentComplete?.();
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to update insight');
    },
  });

  const handleValidateInsight = (
    insightType: 'pain_point' | 'goal',
    insightText: string,
    validation: 'confirmed' | 'rejected'
  ) => {
    validateInsightMutation.mutate({
      insight_type: insightType,
      insight_text: insightText,
      validation,
    });
  };

  return (
    <>
      <div className="space-y-3">
        <div className="flex items-center space-x-2">
          <Checkbox
            id={`force-refresh-${contactId}`}
            checked={forceRefresh}
            onCheckedChange={(checked) => setForceRefresh(checked as boolean)}
          />
          <Label
            htmlFor={`force-refresh-${contactId}`}
            className="text-sm font-normal cursor-pointer"
          >
            Force Refresh (regenerate embedding)
          </Label>
        </div>
        <Button
          variant={variant}
          size={size}
          onClick={() => enrichMutation.mutate(forceRefresh)}
          disabled={enrichMutation.isPending || !contactId}
          className="w-full"
        >
          {enrichMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Sparkles className="h-4 w-4" />
          )}
          {showLabel && (
            <span className="ml-2">
              {enrichMutation.isPending ? 'Enriching...' : 'Enrich with AI'}
            </span>
          )}
        </Button>
      </div>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>AI Enrichment Results</DialogTitle>
            <DialogDescription>
              AI-generated insights for this contact
            </DialogDescription>
          </DialogHeader>

          {enrichmentData && (
            <div className="space-y-6">
              {/* Summary Stats */}
              <div className="grid grid-cols-3 gap-4">
                <div className="rounded-lg border p-4">
                  <div className="text-sm text-muted-foreground">
                    Insights Generated
                  </div>
                  <div className="text-2xl font-bold">
                    {enrichmentData.insights_generated}
                  </div>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="text-sm text-muted-foreground">
                    Avg Confidence
                  </div>
                  <div className="text-2xl font-bold">
                    {Math.round(enrichmentData.confidence_avg * 100)}%
                  </div>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="text-sm text-muted-foreground">
                    Vector Embedding
                  </div>
                  <div className="text-2xl font-bold">
                    {enrichmentData.embedding_created ? (
                      <Badge variant="default">Created</Badge>
                    ) : (
                      <Badge variant="secondary">No</Badge>
                    )}
                  </div>
                </div>
              </div>

              {/* Pain Points */}
              {(enrichmentData.ai_insights.suspected_pain_points ?? []).length > 0 && (
                <div>
                  <h3 className="mb-3 font-semibold">Suspected Pain Points</h3>
                  <div className="space-y-2">
                    {(enrichmentData.ai_insights.suspected_pain_points ?? []).map(
                      (point) => (
                        <div key={point.pain_point} className="rounded-lg border p-3">
                          <div className="mb-1 flex items-start justify-between">
                            <p className="font-medium">{point.pain_point}</p>
                            <Badge variant="outline">
                              {Math.round(point.confidence * 100)}% confident
                            </Badge>
                          </div>
                          <div className="mb-2 flex gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={validateInsightMutation.isPending}
                              onClick={() =>
                                handleValidateInsight('pain_point', point.pain_point, 'confirmed')
                              }
                            >
                              Confirm
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              disabled={validateInsightMutation.isPending}
                              onClick={() =>
                                handleValidateInsight('pain_point', point.pain_point, 'rejected')
                              }
                            >
                              Reject
                            </Button>
                          </div>
                          {point.evidence?.length > 0 && (
                            <ul className="ml-4 list-disc text-sm text-muted-foreground">
                              {point.evidence.map((ev) => (
                                <li key={ev}>{ev}</li>
                              ))}
                            </ul>
                          )}
                        </div>
                      )
                    )}
                  </div>
                </div>
              )}

              {/* Goals */}
              {(enrichmentData.ai_insights.suspected_goals ?? []).length > 0 && (
                <div>
                  <h3 className="mb-3 font-semibold">Suspected Goals</h3>
                  <div className="space-y-2">
                    {(enrichmentData.ai_insights.suspected_goals ?? []).map(
                      (goal) => (
                        <div key={goal.goal} className="rounded-lg border p-3">
                          <div className="mb-1 flex items-start justify-between">
                            <p className="font-medium">{goal.goal}</p>
                            <Badge variant="outline">
                              {Math.round(goal.confidence * 100)}% confident
                            </Badge>
                          </div>
                          <div className="mb-2 flex gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={validateInsightMutation.isPending}
                              onClick={() =>
                                handleValidateInsight('goal', goal.goal, 'confirmed')
                              }
                            >
                              Confirm
                            </Button>
                            <Button
                              size="sm"
                              variant="ghost"
                              disabled={validateInsightMutation.isPending}
                              onClick={() =>
                                handleValidateInsight('goal', goal.goal, 'rejected')
                              }
                            >
                              Reject
                            </Button>
                          </div>
                          {goal.evidence?.length > 0 && (
                            <ul className="ml-4 list-disc text-sm text-muted-foreground">
                              {goal.evidence.map((ev) => (
                                <li key={ev}>{ev}</li>
                              ))}
                            </ul>
                          )}
                        </div>
                      )
                    )}
                  </div>
                </div>
              )}

              {/* Buying Signals */}
              {(enrichmentData.ai_insights.buying_signals ?? []).length > 0 && (
                <div>
                  <h3 className="mb-3 font-semibold">Buying Signals</h3>
                  <div className="space-y-2">
                    {(enrichmentData.ai_insights.buying_signals ?? []).map(
                      (signal) => (
                        <div
                          key={`${signal.signal}-${signal.occurred_at}`}
                          className="flex items-center justify-between rounded-lg border p-3"
                        >
                          <div>
                            <p className="font-medium">{signal.signal}</p>
                            <p className="text-sm text-muted-foreground">
                              {new Date(signal.occurred_at).toLocaleDateString()}
                            </p>
                          </div>
                          <div className="flex items-center gap-2">
                            <Badge
                              variant={
                                signal.strength === 'High'
                                  ? 'default'
                                  : signal.strength === 'Medium'
                                    ? 'secondary'
                                    : 'outline'
                              }
                            >
                              {signal.strength}
                            </Badge>
                            <Badge variant="outline">
                              Score: {signal.recency_score}
                            </Badge>
                          </div>
                        </div>
                      )
                    )}
                  </div>
                </div>
              )}

              {/* Research Suggestions */}
              {(enrichmentData.ai_insights.research_suggestions?.length ?? 0) > 0 && (
                <div>
                  <h3 className="mb-3 font-semibold">
                    What to Research Next
                  </h3>
                  <div className="space-y-3">
                    {enrichmentData.ai_insights.research_suggestions?.map(
                      (suggestion, idx) => (
                        <div
                          key={idx}
                          className="rounded-lg border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-950"
                        >
                          <div className="mb-2 flex items-start justify-between">
                            <div className="flex-1">
                              <div className="mb-1 flex items-center gap-2">
                                <Badge variant="outline">
                                  {suggestion.category}
                                </Badge>
                                <Badge
                                  variant={
                                    suggestion.priority === 'High'
                                      ? 'destructive'
                                      : suggestion.priority === 'Medium'
                                        ? 'default'
                                        : 'secondary'
                                  }
                                >
                                  {suggestion.priority} Priority
                                </Badge>
                              </div>
                              <p className="font-medium">
                                {suggestion.data_point}
                              </p>
                            </div>
                            <Button
                              size="sm"
                              variant="outline"
                              className="ml-2"
                              onClick={() => {
                                setSelectedSuggestion(suggestion);
                                setAddFindingDialogOpen(true);
                              }}
                            >
                              Add Finding
                            </Button>
                          </div>
                          <p className="mb-2 text-sm text-muted-foreground">
                            {suggestion.why_important}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            Where to find: {suggestion.where_to_find}
                          </p>
                        </div>
                      )
                    )}
                  </div>
                </div>
              )}

              {/* Decision Style */}
              {enrichmentData.ai_insights.decision_style && (
                <div>
                  <h3 className="mb-3 font-semibold">Decision Style</h3>
                  <div className="rounded-lg border p-3">
                    <div className="mb-2 flex items-center justify-between">
                      <p className="font-medium">
                        {enrichmentData.ai_insights.decision_style.type}
                      </p>
                      <Badge variant="outline">
                        {Math.round(
                          enrichmentData.ai_insights.decision_style.confidence *
                            100
                        )}
                        % confident
                      </Badge>
                    </div>
                    {enrichmentData.ai_insights.decision_style
                      .key_decision_factors?.length > 0 && (
                      <div className="mt-2">
                        <p className="mb-1 text-sm font-medium">
                          Key Decision Factors:
                        </p>
                        <div className="flex flex-wrap gap-2">
                          {enrichmentData.ai_insights.decision_style.key_decision_factors.map(
                            (factor, i) => (
                              <Badge key={i} variant="secondary">
                                {factor}
                              </Badge>
                            )
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Add Finding Dialog */}
      {selectedSuggestion && (
        <AddFindingDialog
          open={addFindingDialogOpen}
          onOpenChange={setAddFindingDialogOpen}
          contactId={contactId}
          suggestion={selectedSuggestion}
          onSuccess={() => {
            toast.success('Finding added! You can view it in the contact profile.');
            // Optionally re-enrich to update data
          }}
        />
      )}
    </>
  );
}
