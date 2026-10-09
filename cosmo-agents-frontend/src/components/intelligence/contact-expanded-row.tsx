'use client';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import ContactApi from '@/network/client/contact';
import EnrollmentApi from '@/network/client/enrollment';
import PlaybookApi, { type PlaybookRead } from '@/network/client/playbook';
import type { Contact } from '@/models/contact';
import { format } from 'date-fns';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import React, { useState } from 'react';
import { Pencil, Trash2, Check, X, Workflow } from 'lucide-react';
import { toast } from 'sonner';
import { ExtractFromURLSimpleDialog } from '@/components/forms/extract-from-url-simple-dialog';
import { ExtractFromScreenshotDialog } from '@/components/forms/extract-from-screenshot-dialog';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface ContactExpandedRowProps {
  contact: Contact;
}

export function ContactExpandedRow({ contact }: ContactExpandedRowProps) {
  const queryClient = useQueryClient();
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [enrollDialogOpen, setEnrollDialogOpen] = useState(false);
  const [selectedPlaybookId, setSelectedPlaybookId] = useState<string>('');

  const {
    data: detail,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ['contact-detail', contact.id],
    queryFn: () => ContactApi.getById(contact.id),
    staleTime: 30_000,
  });

  // Fetch available playbooks for enrollment
  const { data: playbooks } = useQuery({
    queryKey: ['playbooks'],
    queryFn: () => PlaybookApi.list(),
  });

  const effective = detail?.data || contact;

  const customFields = (effective.profile as any)?.custom_fields || {};
  const researchFindings = (effective.profile as any)?.research_findings || [];
  const aiInsights = effective.ai_insights || {};

  const parseMaybeJSON = (raw: string) => {
    const trimmed = raw.trim();
    if (!trimmed) return raw;
    if (!(trimmed.startsWith('{') || trimmed.startsWith('['))) return raw;
    try {
      return JSON.parse(trimmed);
    } catch {
      return raw;
    }
  };

  const orderObjectKeys = (obj: Record<string, any>) => {
    const preferred = [
      'name',
      'company',
      'title',
      'role',
      'school',
      'degree',
      'field',
      'duration',
      'years',
      'location',
      'description',
    ];
    const keys = Object.keys(obj);
    const ordered = preferred.filter((key) => keys.includes(key));
    const rest = keys.filter((key) => !ordered.includes(key)).sort();
    return [...ordered, ...rest];
  };

  const objectLabel = (obj: Record<string, any>) => {
    if (obj.company && obj.title) return `${obj.company} — ${obj.title}`;
    if (obj.school && obj.degree) return `${obj.school} — ${obj.degree}`;
    if (obj.school) return obj.school;
    if (obj.name) return obj.name;
    if (obj.title) return obj.title;
    return null;
  };

  const renderObject = (obj: Record<string, any>) => (
    <dl className="text-sm text-gray-600">
      {orderObjectKeys(obj).map((key) => (
        <div key={key} className="flex flex-wrap gap-x-2 gap-y-1 py-0.5">
          <dt className="font-medium text-gray-700">{key}:</dt>
          <dd className="break-words text-gray-600">{renderValue(obj[key])}</dd>
        </div>
      ))}
    </dl>
  );

  const renderArray = (arr: any[]) => {
    if (arr.length === 0) {
      return <span className="text-gray-400">-</span>;
    }
    const allPrimitive = arr.every(
      (item) =>
        item === null || ['string', 'number', 'boolean'].includes(typeof item)
    );
    if (allPrimitive) {
      return (
        <ul className="list-disc pl-4 text-sm text-gray-600">
          {arr.map((item, idx) => (
            <li key={idx} className="break-words">
              {String(item)}
            </li>
          ))}
        </ul>
      );
    }
    return (
      <div className="space-y-2">
        {arr.map((item, idx) => {
          const label =
            item && typeof item === 'object' && !Array.isArray(item)
              ? objectLabel(item)
              : null;
          return (
            <div
              key={idx}
              className="rounded-md border border-gray-200 bg-gray-50 p-2"
            >
              <div className="mb-1 text-xs font-medium text-gray-500">
                {label ? label : `Item ${idx + 1}`}
              </div>
              {renderValue(item)}
            </div>
          );
        })}
      </div>
    );
  };

  const renderLongText = (text: string) => {
    const trimmed = text.trim();
    if (trimmed.length <= 200 && !trimmed.includes('\n')) {
      return <span className="break-words">{trimmed}</span>;
    }
    const preview = trimmed.slice(0, 180);
    return (
      <details className="text-sm text-gray-600">
        <summary className="cursor-pointer text-gray-700">
          Show full text
        </summary>
        <p className="mt-2 whitespace-pre-wrap break-words">{trimmed}</p>
        <p className="mt-2 text-xs text-gray-500">Preview: {preview}...</p>
      </details>
    );
  };

  const renderValue = (value: any): React.ReactNode => {
    if (value === null || value === undefined || value === '') {
      return <span className="text-gray-400">-</span>;
    }

    if (typeof value === 'string') {
      if (value.includes('map[')) {
        return <span className="text-gray-400">-</span>;
      }
      const parsed = parseMaybeJSON(value);
      if (parsed !== value) {
        return renderValue(parsed);
      }
      return renderLongText(value);
    }

    if (typeof value === 'number' || typeof value === 'boolean') {
      return <span>{String(value)}</span>;
    }

    if (Array.isArray(value)) {
      return renderArray(value);
    }

    if (typeof value === 'object') {
      return renderObject(value);
    }

    return <span>{String(value)}</span>;
  };

  const normalizeEditableValue = (value: any): string => {
    if (value === null || value === undefined) return '';
    if (typeof value === 'string') return value;
    if (typeof value === 'number' || typeof value === 'boolean')
      return String(value);
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  };

  // Update custom field mutation
  const updateFieldMutation = useMutation({
    mutationFn: async ({
      fieldName,
      value,
    }: {
      fieldName: string;
      value: string;
    }) => {
      const payload = { [fieldName]: value };
      return ContactApi.update(contact.id, payload);
    },
    onSuccess: () => {
      toast.success('Field updated successfully');
      setEditingField(null);
      refetch();
      queryClient.invalidateQueries({ queryKey: ['contacts'] });
    },
    onError: () => {
      toast.error('Failed to update field');
    },
  });

  // Delete custom field mutation
  const deleteFieldMutation = useMutation({
    mutationFn: async (fieldName: string) => {
      // Send empty string to delete the field
      const payload = { [fieldName]: '' };
      return ContactApi.update(contact.id, payload);
    },
    onSuccess: () => {
      toast.success('Field deleted successfully');
      refetch();
      queryClient.invalidateQueries({ queryKey: ['contacts'] });
    },
    onError: () => {
      toast.error('Failed to delete field');
    },
  });

  // Enroll contact in playbook mutation
  const enrollMutation = useMutation({
    mutationFn: async (playbookId: string) => {
      return EnrollmentApi.enrollContact(contact.id, playbookId);
    },
    onSuccess: () => {
      toast.success('Contact enrolled in playbook successfully!');
      setEnrollDialogOpen(false);
      setSelectedPlaybookId('');
    },
    onError: (error: any) => {
      toast.error(
        `Failed to enroll contact: ${error.message || 'Unknown error'}`
      );
    },
  });

  // Update business stage mutation
  const updateBusinessStageMutation = useMutation({
    mutationFn: async (stage: string) => {
      return ContactApi.update(contact.id, { business_stage: stage });
    },
    onSuccess: () => {
      toast.success('Business stage updated');
      refetch();
      queryClient.invalidateQueries({ queryKey: ['contacts'] });
    },
    onError: () => {
      toast.error('Failed to update business stage');
    },
  });

  const handleEdit = (fieldName: string, currentValue: any) => {
    setEditingField(fieldName);
    setEditValue(normalizeEditableValue(currentValue));
  };

  const handleSave = (fieldName: string) => {
    updateFieldMutation.mutate({ fieldName, value: editValue });
  };

  const handleCancel = () => {
    setEditingField(null);
    setEditValue('');
  };

  const handleDelete = (fieldName: string) => {
    if (confirm(`Are you sure you want to delete the field "${fieldName}"?`)) {
      deleteFieldMutation.mutate(fieldName);
    }
  };

  const handleEnroll = () => {
    if (!selectedPlaybookId) {
      toast.error('Please select a playbook');
      return;
    }
    enrollMutation.mutate(selectedPlaybookId);
  };

  const hasCustomFields = Object.keys(customFields).length > 0;
  const hasResearchFindings = researchFindings.length > 0;
  const hasAIInsights =
    (aiInsights.suspected_pain_points?.length ?? 0) > 0 ||
    (aiInsights.suspected_goals?.length ?? 0) > 0 ||
    (aiInsights.buying_signals?.length ?? 0) > 0;

  if (!hasCustomFields && !hasResearchFindings && !hasAIInsights) {
    return (
      <div className="bg-gray-50 p-6">
        <div className="mb-4 flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            {isLoading
              ? 'Loading contact details...'
              : 'No additional research data available for this contact.'}
          </p>
          <div className="flex gap-2">
            <Dialog open={enrollDialogOpen} onOpenChange={setEnrollDialogOpen}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm">
                  <Workflow className="mr-2 h-4 w-4" />
                  Enroll in Playbook
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Enroll Contact in Playbook</DialogTitle>
                  <DialogDescription>
                    Select a playbook to manually enroll this contact. The
                    contact will start from the first stage.
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  <div className="space-y-2">
                    <Label>Select Playbook</Label>
                    <Select
                      value={selectedPlaybookId}
                      onValueChange={setSelectedPlaybookId}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Choose a playbook..." />
                      </SelectTrigger>
                      <SelectContent>
                        {playbooks?.data?.map((playbook: PlaybookRead) => (
                          <SelectItem
                            key={playbook.playbook_id}
                            value={playbook.playbook_id}
                          >
                            {playbook.name} ({playbook.playbook_type})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div className="flex justify-end gap-2">
                  <Button
                    variant="outline"
                    onClick={() => setEnrollDialogOpen(false)}
                  >
                    Cancel
                  </Button>
                  <Button
                    onClick={handleEnroll}
                    disabled={enrollMutation.isPending}
                  >
                    {enrollMutation.isPending ? 'Enrolling...' : 'Enroll'}
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
            <ExtractFromURLSimpleDialog
              contactId={contact.id}
              onSuccess={() => refetch()}
            />
            <ExtractFromScreenshotDialog
              contactId={contact.id}
              onSuccess={() => refetch()}
            />
          </div>
        </div>
        {/* Business Stage Selector */}
        <div className="mt-4 flex items-center gap-3 rounded-lg border bg-white p-3">
          <span className="text-sm font-medium text-gray-700">
            Pipeline Stage:
          </span>
          <Select
            value={effective.business_stage || 'PRE_SALES'}
            onValueChange={(value) => updateBusinessStageMutation.mutate(value)}
            disabled={updateBusinessStageMutation.isPending}
          >
            <SelectTrigger className="w-[160px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="PRE_SALES">
                <span className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full bg-blue-500" />
                  Pre-Sales
                </span>
              </SelectItem>
              <SelectItem value="SALES">
                <span className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full bg-amber-500" />
                  Sales
                </span>
              </SelectItem>
              <SelectItem value="POST_SALES">
                <span className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full bg-green-500" />
                  Post-Sales
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-50 p-6">
      <div className="mb-4 flex items-center justify-end gap-2">
        <Dialog open={enrollDialogOpen} onOpenChange={setEnrollDialogOpen}>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm">
              <Workflow className="mr-2 h-4 w-4" />
              Enroll in Playbook
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Enroll Contact in Playbook</DialogTitle>
              <DialogDescription>
                Select a playbook to manually enroll this contact. The contact
                will start from the first stage.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label>Select Playbook</Label>
                <Select
                  value={selectedPlaybookId}
                  onValueChange={setSelectedPlaybookId}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Choose a playbook..." />
                  </SelectTrigger>
                  <SelectContent>
                    {playbooks?.data?.map((playbook: PlaybookRead) => (
                      <SelectItem
                        key={playbook.playbook_id}
                        value={playbook.playbook_id}
                      >
                        {playbook.name} ({playbook.playbook_type})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setEnrollDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button
                onClick={handleEnroll}
                disabled={enrollMutation.isPending}
              >
                {enrollMutation.isPending ? 'Enrolling...' : 'Enroll'}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
        <ExtractFromURLSimpleDialog
          contactId={contact.id}
          onSuccess={() => refetch()}
        />
        <ExtractFromScreenshotDialog
          contactId={contact.id}
          onSuccess={() => refetch()}
        />
      </div>
      <div className="grid gap-6 md:grid-cols-2">
        {/* Custom Fields Section */}
        {hasCustomFields && (
          <div>
            <h3 className="mb-3 font-semibold text-gray-900">Custom Fields</h3>
            <div className="space-y-3">
              {Object.entries(customFields).map(
                ([fieldName, fieldData]: [string, any]) => {
                  const rawValue = fieldData?.value;
                  const updatedAt = fieldData?.updated_at;

                  // Skip invalid data
                  if (
                    typeof rawValue === 'string' &&
                    rawValue.includes('map[')
                  ) {
                    return null;
                  }

                  const displayValue = renderValue(rawValue);
                  const isEditing = editingField === fieldName;

                  return (
                    <div
                      key={fieldName}
                      className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="text-sm font-medium text-gray-900">
                            {fieldName}
                          </span>
                        </div>
                        <div className="flex items-center gap-1">
                          {!isEditing ? (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={() => handleEdit(fieldName, rawValue)}
                              >
                                <Pencil className="h-3 w-3" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-red-600"
                                onClick={() => handleDelete(fieldName)}
                              >
                                <Trash2 className="h-3 w-3" />
                              </Button>
                            </>
                          ) : (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-green-600"
                                onClick={() => handleSave(fieldName)}
                              >
                                <Check className="h-3 w-3" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={handleCancel}
                              >
                                <X className="h-3 w-3" />
                              </Button>
                            </>
                          )}
                        </div>
                      </div>
                      {updatedAt && (
                        <div className="mt-1 text-xs text-gray-500">
                          Updated: {format(new Date(updatedAt), 'PP p')}
                        </div>
                      )}
                      {isEditing ? (
                        <Textarea
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          className="mt-2 text-sm"
                          rows={4}
                          autoFocus
                        />
                      ) : (
                        <div className="mt-2 text-sm text-gray-600">
                          {displayValue}
                        </div>
                      )}
                    </div>
                  );
                }
              )}
            </div>
          </div>
        )}

        {/* AI Insights - Simplified */}
        {hasAIInsights && (
          <div className="md:col-span-2">
            <h3 className="mb-3 font-semibold text-gray-900">AI Insights</h3>
            <div className="space-y-3">
              {(aiInsights.suspected_pain_points ?? []).length > 0 && (
                <div className="border-l-4 border-amber-400 pl-3">
                  <p className="mb-1 text-sm font-medium text-amber-900">
                    Pain Points
                  </p>
                  <ul className="space-y-1 text-sm text-gray-700">
                    {(aiInsights.suspected_pain_points ?? []).map((p) => (
                      <li key={p.pain_point}>• {p.pain_point}</li>
                    ))}
                  </ul>
                </div>
              )}

              {(aiInsights.suspected_goals ?? []).length > 0 && (
                <div className="border-l-4 border-emerald-400 pl-3">
                  <p className="mb-1 text-sm font-medium text-emerald-900">
                    Goals
                  </p>
                  <ul className="space-y-1 text-sm text-gray-700">
                    {(aiInsights.suspected_goals ?? []).map((g) => (
                      <li key={g.goal}>• {g.goal}</li>
                    ))}
                  </ul>
                </div>
              )}

              {(aiInsights.buying_signals ?? []).length > 0 && (
                <div className="border-l-4 border-blue-400 pl-3">
                  <p className="mb-1 text-sm font-medium text-blue-900">
                    Buying Signals
                  </p>
                  <ul className="space-y-1 text-sm text-gray-700">
                    {(aiInsights.buying_signals ?? []).map((s) => (
                      <li key={s.signal}>• {s.signal}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Research Findings - Simplified */}
        {hasResearchFindings && (
          <div>
            <h3 className="mb-3 font-semibold text-gray-900">
              Research Findings
            </h3>
            <div className="space-y-2">
              {researchFindings
                .sort(
                  (a, b) =>
                    new Date(b.added_at).getTime() -
                    new Date(a.added_at).getTime()
                )
                .map((finding, idx) => (
                  <div key={idx} className="border-b pb-2">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <p className="text-sm font-medium">
                          {finding.field_name}
                        </p>
                        <p className="text-sm text-gray-600">
                          {renderValue(finding.value)}
                        </p>
                      </div>
                      <Badge variant="outline" className="text-xs">
                        {finding.category}
                      </Badge>
                    </div>
                    <p className="mt-1 text-xs text-gray-500">
                      {format(new Date(finding.added_at), 'MMM d, yyyy')}
                      {finding.source && ` • ${finding.source}`}
                    </p>
                  </div>
                ))}
            </div>
          </div>
        )}

        {/* Business Stage Selector */}
        <div className="flex items-center gap-3 rounded-lg border bg-white p-3 md:col-span-2">
          <span className="text-sm font-medium text-gray-700">
            Pipeline Stage:
          </span>
          <Select
            value={effective.business_stage || 'PRE_SALES'}
            onValueChange={(value) => updateBusinessStageMutation.mutate(value)}
            disabled={updateBusinessStageMutation.isPending}
          >
            <SelectTrigger className="w-[160px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="PRE_SALES">
                <span className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full bg-blue-500" />
                  Pre-Sales
                </span>
              </SelectItem>
              <SelectItem value="SALES">
                <span className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full bg-amber-500" />
                  Sales
                </span>
              </SelectItem>
              <SelectItem value="POST_SALES">
                <span className="flex items-center gap-2">
                  <span className="h-2 w-2 rounded-full bg-green-500" />
                  Post-Sales
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>
  );
}
