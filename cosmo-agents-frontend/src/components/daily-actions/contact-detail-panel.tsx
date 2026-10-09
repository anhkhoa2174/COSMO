'use client';

import { useState, useEffect } from 'react';
import { useAtom } from 'jotai';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X, Loader2, Send, Sparkles, Check, Trash2, Pencil, AlertTriangle, Brain, BarChart3, MessageSquare, Search, Plus, CheckCircle, XCircle } from 'lucide-react';
import { toast } from 'sonner';
import { Sheet, SheetContent } from '@/components/ui/sheet';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { activePanelAtom } from '@/stores/daily-actions';
import { cn } from '@/lib/utils';
import ContactApi from '@/network/client/contact';
import OutreachApi, { type InteractionChannel, type Meeting } from '@/network/client/outreach';
import IntelligenceApi, { type ContactEnrichmentResponse, type MeetingBriefResponse, type CalculateScoresResponse } from '@/network/client/intelligence';
import { ContactAvatar } from './parts/contact-avatar';

/** Recursively render any JSON-like value safely as React nodes */
function RenderValue({ value, depth = 0 }: { value: unknown; depth?: number }) {
  if (value === null || value === undefined) return null;
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return <span className="text-foreground">{String(value)}</span>;
  }
  if (Array.isArray(value)) {
    return (
      <div className={cn('space-y-1.5', depth > 0 && 'ml-2')}>
        {value.map((item, i) => (
          <div key={i} className="rounded-md border border-border bg-card px-3 py-2 text-[0.82rem]">
            <RenderValue value={item} depth={depth + 1} />
          </div>
        ))}
      </div>
    );
  }
  if (typeof value === 'object') {
    return (
      <div className={cn('space-y-0.5', depth > 0 && 'ml-2')}>
        {Object.entries(value as Record<string, unknown>).map(([k, v]) => {
          const isComplex = typeof v === 'object' && v !== null;
          return (
            <div key={k} className={isComplex ? 'py-1' : 'flex gap-2 py-0.5'}>
              <span className="text-[0.62rem] font-mono text-muted-foreground/80 uppercase shrink-0">
                {k.replace(/_/g, ' ')}{isComplex ? '' : ':'}
              </span>
              {isComplex ? (
                <div className="mt-1 rounded-md border border-border bg-muted/50 px-3 py-2">
                  <RenderValue value={v} depth={depth + 1} />
                </div>
              ) : (
                <RenderValue value={v} depth={depth + 1} />
              )}
            </div>
          );
        })}
      </div>
    );
  }
  return <span className="text-foreground">{String(value)}</span>;
}

/** Safely convert any value to a renderable React node */
function safeRender(v: unknown): React.ReactNode {
  if (v === null || v === undefined) return '—';
  if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') return String(v);
  // React elements pass through
  if (typeof v === 'object' && '$$typeof' in (v as object)) return v as React.ReactNode;
  // Plain objects/arrays → use RenderValue
  return <RenderValue value={v} />;
}

function FieldRow({ label, value, missing }: { label: string; value: unknown; missing?: boolean }) {
  const rendered = safeRender(value);
  const isComplex = typeof value === 'object' && value !== null && !('$$typeof' in (value as object));
  return (
    <div className={cn(isComplex ? 'border-b border-border py-3' : 'flex items-center justify-between border-b border-border py-3')}>
      <span className="text-[0.68rem] font-bold uppercase tracking-[0.04em] text-muted-foreground/80 font-mono">
        {label}
      </span>
      {isComplex ? (
        <div className="mt-1 rounded-md border border-border bg-card px-3 py-2 text-[0.82rem]">
          {rendered}
        </div>
      ) : (
        <span className={cn('text-[0.85rem] font-semibold text-right max-w-[60%] break-words', missing ? 'text-destructive italic' : 'text-foreground')}>
          {rendered}
        </span>
      )}
    </div>
  );
}

function SectionHeader({ title, icon }: { title: string; icon?: string }) {
  return (
    <div className="mb-1 mt-5 flex items-center gap-2 first:mt-0">
      {icon && <span className="text-sm">{icon}</span>}
      <span className="text-[0.62rem] font-bold uppercase tracking-[0.06em] text-indigo-600 dark:text-indigo-400 font-mono">
        {title}
      </span>
      <div className="flex-1 border-b border-border" />
    </div>
  );
}

/** Inline edit form for contact fields */
function EditContactForm({ contactId, onClose }: { contactId: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const { data: contactResponse } = useQuery({
    queryKey: ['contact', contactId],
    queryFn: () => ContactApi.getById(contactId),
    enabled: !!contactId,
  });
  const contact = contactResponse?.data;

  const [form, setForm] = useState({
    name: '',
    email: '',
    company: '',
    job_title: '',
    phone: '',
    linkedin_url: '',
    city: '',
    country: '',
    industry: '',
  });

  // Populate form when contact loads
  useEffect(() => {
    if (contact) {
      setForm({
        name: contact.name || '',
        email: contact.profile?.email || contact.email || '',
        company: contact.company || '',
        job_title: contact.job_title || '',
        phone: contact.profile?.phone || contact.phone || '',
        linkedin_url: contact.profile?.linkedin_url || '',
        city: contact.city || '',
        country: contact.country || '',
        industry: contact.industry || '',
      });
    }
  }, [contact]);

  const updateMutation = useMutation({
    mutationFn: (payload: Record<string, string>) =>
      ContactApi.update(contactId, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contact', contactId] });
      toast.success('Contact updated');
      onClose();
    },
    onError: () => toast.error('Failed to update contact'),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // Only send changed fields
    const payload: Record<string, string> = {};
    for (const [key, val] of Object.entries(form)) {
      if (val.trim()) payload[key] = val.trim();
    }
    updateMutation.mutate(payload);
  };

  const fields = [
    { key: 'name', label: 'Name' },
    { key: 'email', label: 'Email' },
    { key: 'company', label: 'Company' },
    { key: 'job_title', label: 'Job Title' },
    { key: 'phone', label: 'Phone' },
    { key: 'linkedin_url', label: 'LinkedIn URL' },
    { key: 'city', label: 'City' },
    { key: 'country', label: 'Country' },
    { key: 'industry', label: 'Industry' },
  ] as const;

  return (
    <ScrollArea className="h-[60vh]">
      <form onSubmit={handleSubmit} className="pr-4 space-y-3">
        {fields.map((f) => (
          <div key={f.key}>
            <label className="block text-[0.62rem] font-mono font-bold uppercase tracking-[0.04em] text-muted-foreground/80 mb-1">
              {f.label}
            </label>
            <input
              type="text"
              value={form[f.key]}
              onChange={(e) => setForm((prev) => ({ ...prev, [f.key]: e.target.value }))}
              className="w-full rounded-md border border-border bg-card px-3 py-2 text-[0.85rem] text-foreground placeholder:text-muted-foreground/80 focus:border-indigo-400 focus:outline-none"
              placeholder={f.label}
            />
          </div>
        ))}
        <div className="flex gap-2 pt-2">
          <button
            type="submit"
            disabled={updateMutation.isPending}
            className="flex items-center gap-1.5 rounded-md bg-indigo-600 px-4 py-2 text-[0.78rem] font-bold text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {updateMutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
            Save
          </button>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-border bg-card px-4 py-2 text-[0.78rem] font-bold text-muted-foreground hover:bg-muted"
          >
            Cancel
          </button>
        </div>
      </form>
    </ScrollArea>
  );
}

/** Delete confirmation dialog */
function DeleteContactConfirm({ contactId, contactName, onClose }: { contactId: string; contactName?: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const [, setActivePanel] = useAtom(activePanelAtom);

  const deleteMutation = useMutation({
    mutationFn: () => ContactApi.delete({ ids: [contactId] }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contact'] });
      queryClient.invalidateQueries({ queryKey: ['contacts'] });
      toast.success('Contact deleted');
      setActivePanel(null);
    },
    onError: () => toast.error('Failed to delete contact'),
  });

  return (
    <div className="rounded-lg border border-destructive/[0.3] bg-destructive/[0.06] p-5">
      <div className="flex items-center gap-2 mb-3">
        <AlertTriangle className="h-5 w-5 text-destructive" />
        <span className="text-[0.92rem] font-bold text-destructive">Delete Contact</span>
      </div>
      <p className="text-[0.85rem] text-muted-foreground mb-4">
        Are you sure you want to delete <strong className="text-foreground">{contactName || contactId}</strong>? This action cannot be undone.
      </p>
      <div className="flex gap-2">
        <button
          onClick={() => deleteMutation.mutate()}
          disabled={deleteMutation.isPending}
          className="flex items-center gap-1.5 rounded-md bg-destructive px-4 py-2 text-[0.78rem] font-bold text-destructive-foreground hover:bg-destructive/90 disabled:opacity-50"
        >
          {deleteMutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
          Delete
        </button>
        <button
          onClick={onClose}
          className="rounded-md border border-border bg-card px-4 py-2 text-[0.78rem] font-bold text-muted-foreground hover:bg-muted"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

function ContactOverviewTab({ contactId }: { contactId: string }) {
  const [mode, setMode] = useState<'view' | 'edit' | 'delete'>('view');
  const { data: contactResponse, isLoading } = useQuery({
    queryKey: ['contact', contactId],
    queryFn: () => ContactApi.getById(contactId),
    enabled: !!contactId,
  });

  const contact = contactResponse?.data;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-indigo-600 dark:text-indigo-400" />
      </div>
    );
  }

  if (!contact) {
    return (
      <p className="py-4 text-sm text-muted-foreground/80">Contact not found.</p>
    );
  }

  if (mode === 'edit') {
    return <EditContactForm contactId={contactId} onClose={() => setMode('view')} />;
  }

  if (mode === 'delete') {
    return <DeleteContactConfirm contactId={contactId} contactName={contact.name} onClose={() => setMode('view')} />;
  }

  const email = contact.profile?.email || contact.email;
  const phone = contact.profile?.phone || contact.phone;
  const linkedinUrl = contact.profile?.linkedin_url;
  const customFields = contact.profile?.custom_fields;
  const researchFindings = contact.profile?.research_findings;

  // Build location string
  const locationParts = [contact.city, contact.state, contact.country].filter(Boolean);
  const location = locationParts.length > 0 ? locationParts.join(', ') : null;

  return (
    <ScrollArea className="h-[60vh]">
      <div className="pr-4">
        {/* ── Action Buttons ── */}
        <div className="flex gap-2 mb-4">
          <button
            onClick={() => setMode('edit')}
            className="flex items-center gap-1.5 rounded-md border border-border bg-card px-3 py-1.5 text-[0.75rem] font-bold text-muted-foreground hover:bg-muted hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors"
          >
            <Pencil className="h-3.5 w-3.5" />
            Edit
          </button>
          <button
            onClick={() => setMode('delete')}
            className="flex items-center gap-1.5 rounded-md border border-destructive/[0.2] bg-destructive/[0.06] px-3 py-1.5 text-[0.75rem] font-bold text-destructive hover:bg-destructive/[0.12] transition-colors"
          >
            <Trash2 className="h-3.5 w-3.5" />
            Delete
          </button>
        </div>

        {/* ── Contact Info ── */}
        <SectionHeader title="Contact Info" />
        {contact.job_title && <FieldRow label="Title" value={contact.job_title} />}
        {contact.company && <FieldRow label="Company" value={contact.company} />}
        {contact.industry && <FieldRow label="Industry" value={contact.industry} />}
        {email && (
          <FieldRow
            label="Email"
            value={<a href={`mailto:${email}`} className="text-indigo-600 dark:text-indigo-400 underline">{email}</a>}
          />
        )}
        {phone && <FieldRow label="Phone" value={phone} />}
        {linkedinUrl && (
          <FieldRow
            label="LinkedIn"
            value={<a href={linkedinUrl} target="_blank" rel="noopener noreferrer" className="text-indigo-600 dark:text-indigo-400 underline">Profile</a>}
          />
        )}
        {contact.contact_information && !linkedinUrl && !email && (
          <FieldRow label="Contact" value={contact.contact_information} />
        )}
        {location && <FieldRow label="Location" value={location} />}
        {contact.source && <FieldRow label="Source" value={contact.source} />}

        {/* ── Outreach Status ── */}
        <SectionHeader title="Outreach Status" />
        {contact.outreach_stage && <FieldRow label="Stage" value={contact.outreach_stage} />}
        {contact.lifecycle_stage && <FieldRow label="Lifecycle" value={contact.lifecycle_stage} />}
        {contact.business_stage && <FieldRow label="Pipeline" value={contact.business_stage} />}
        {contact.outreach_decision && <FieldRow label="Decision" value={contact.outreach_decision} />}
        {contact.context_level && <FieldRow label="Context" value={contact.context_level} />}
        {contact.next_step && <FieldRow label="Next Step" value={contact.next_step} />}
        {contact.scenario && <FieldRow label="Scenario" value={contact.scenario} />}
        {contact.last_outcome && <FieldRow label="Last Outcome" value={contact.last_outcome} />}
        {contact.followup_count != null && contact.followup_count > 0 && (
          <FieldRow label="Follow-ups" value={contact.followup_count} />
        )}
        {contact.meeting && <FieldRow label="Meeting" value={contact.meeting} />}
        {contact.contact_channel && <FieldRow label="Channel" value={contact.contact_channel} />}

        {/* ── Missing Fields ── */}
        {contact.missing_fields && contact.missing_fields.length > 0 && (
          <>
            <SectionHeader title="Missing Fields" />
            <div className="flex flex-wrap gap-1.5 py-3">
              {contact.missing_fields.map((f) => (
                <span
                  key={f}
                  className="rounded px-2 py-0.5 font-mono text-[0.62rem] font-bold border border-destructive/[0.25] bg-destructive/[0.1] text-destructive"
                >
                  {f}
                </span>
              ))}
            </div>
          </>
        )}

        {/* ── Custom Fields (special display) ── */}
        {customFields && Object.keys(customFields).length > 0 && (
          <>
            <SectionHeader title="Custom Fields" />
            <div className="mt-1 space-y-2">
              {Object.entries(customFields).map(([key, field]) => (
                <div
                  key={key}
                  className="rounded-lg border border-dashed border-violet-500/[0.3] bg-violet-500/[0.06] px-4 py-2.5"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-[0.68rem] font-bold uppercase tracking-[0.04em] text-violet-600 dark:text-violet-400 font-mono">
                      {key.replace(/_/g, ' ')}
                    </span>
                    {field.source && (
                      <span className="text-[0.55rem] font-mono text-muted-foreground/80 bg-muted rounded px-1.5 py-0.5">
                        {field.source}
                      </span>
                    )}
                  </div>
                  <div className="mt-1 text-[0.85rem] text-foreground leading-relaxed whitespace-pre-wrap">
                    {typeof field.value === 'object' && field.value !== null
                      ? <RenderValue value={field.value} />
                      : field.value}
                  </div>
                  {field.updated_at && (
                    <span className="mt-1 block text-[0.55rem] font-mono text-muted-foreground/80">
                      Updated {new Date(field.updated_at).toLocaleDateString()}
                    </span>
                  )}
                </div>
              ))}
            </div>
          </>
        )}

        {/* ── Research Findings ── */}
        {researchFindings && researchFindings.length > 0 && (
          <>
            <SectionHeader title="Research Findings" />
            <div className="mt-1 space-y-2">
              {researchFindings.map((finding, i) => (
                <div
                  key={i}
                  className="rounded-lg border border-dashed border-amber-500/[0.3] bg-amber-500/[0.06] px-4 py-2.5"
                >
                  <div className="flex items-center justify-between">
                    <span className="text-[0.68rem] font-bold uppercase tracking-[0.04em] text-amber-600 dark:text-amber-400 font-mono">
                      {finding.field_name.replace(/_/g, ' ')}
                    </span>
                    <span className="text-[0.55rem] font-mono text-muted-foreground/80 bg-muted rounded px-1.5 py-0.5">
                      {finding.category}
                    </span>
                  </div>
                  <p className="mt-1 text-[0.85rem] text-foreground leading-relaxed whitespace-pre-wrap">
                    {typeof finding.value === 'object' ? JSON.stringify(finding.value, null, 2) : finding.value}
                  </p>
                  <div className="mt-1 flex items-center gap-2">
                    {finding.source && (
                      <span className="text-[0.55rem] font-mono text-muted-foreground/80">
                        via {finding.source}
                      </span>
                    )}
                    {finding.why_important && (
                      <span className="text-[0.55rem] italic text-muted-foreground">
                        — {finding.why_important}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

        {/* ── Confirmed Facts ── */}
        {contact.confirmed_facts && Object.keys(contact.confirmed_facts).length > 0 && (
          <>
            <SectionHeader title="Confirmed Facts" />
            {Object.entries(contact.confirmed_facts).map(([key, value]) => {
              const isComplex = typeof value === 'object' && value !== null;
              if (isComplex) {
                return (
                  <div key={key} className="mt-2">
                    <span className="text-[0.68rem] font-bold uppercase tracking-[0.04em] text-muted-foreground/80 font-mono">
                      {key.replace(/_/g, ' ')}
                    </span>
                    <div className="mt-1 rounded-md border border-border bg-card px-3 py-2 text-[0.82rem]">
                      <RenderValue value={value} />
                    </div>
                  </div>
                );
              }
              return (
                <FieldRow
                  key={key}
                  label={key.replace(/_/g, ' ')}
                  value={String(value ?? '')}
                />
              );
            })}
          </>
        )}

        {/* ── AI Insights (with validation) ── */}
        {contact.ai_insights && (
          <>
            <SectionHeader title="AI Insights" />
            <div className="mt-1 space-y-2">
              {contact.ai_insights.suspected_pain_points?.map((pp: any) => (
                <InsightItem
                  key={pp.pain_point}
                  label="Pain Point"
                  text={pp.pain_point}
                  type="pain_point"
                  contactId={contact.id}
                />
              ))}
              {contact.ai_insights.suspected_goals?.map((g: any) => (
                <InsightItem
                  key={g.goal}
                  label="Goal"
                  text={g.goal}
                  type="goal"
                  contactId={contact.id}
                />
              ))}
              {contact.ai_insights.anticipated_objections?.map((o: any) => (
                <InsightItem
                  key={o.objection}
                  label="Objection"
                  text={o.objection}
                  type="objection"
                  contactId={contact.id}
                />
              ))}
              {contact.ai_insights.buying_signals?.map((bs: any) => (
                <InsightItem
                  key={bs.signal}
                  label="Buying Signal"
                  text={bs.signal}
                  type="signal"
                  contactId={contact.id}
                />
              ))}

              {contact.ai_insights.decision_style?.type && (
                <div className="rounded-lg border border-border bg-muted/50 px-4 py-2.5">
                  <p className="text-[0.85rem] text-muted-foreground">
                    <strong className="text-foreground">Decision Style: </strong>
                    {contact.ai_insights.decision_style.type}
                  </p>
                </div>
              )}

              {contact.ai_insights.communication_preferences?.preferred_channel && (
                <div className="rounded-lg border border-border bg-muted/50 px-4 py-2.5">
                  <p className="text-[0.85rem] text-muted-foreground">
                    <strong className="text-foreground">Preferred Channel: </strong>
                    {contact.ai_insights.communication_preferences.preferred_channel}
                    {contact.ai_insights.communication_preferences.best_time_to_contact &&
                      ` · Best time: ${contact.ai_insights.communication_preferences.best_time_to_contact}`}
                  </p>
                </div>
              )}

              {/* ── Suggested Research ── */}
              {(contact.ai_insights.research_suggestions?.length ?? 0) > 0 && (
                <div className="mt-3">
                  <span className="text-[0.6rem] font-mono font-bold uppercase tracking-[0.06em] text-violet-600 dark:text-violet-400">
                    <Search className="mr-1 inline h-3 w-3" />
                    Suggested Research
                  </span>
                  <div className="mt-1.5 space-y-1.5">
                    {contact.ai_insights.research_suggestions!.map((s: any, i: number) => (
                      <div
                        key={i}
                        className="rounded-lg border border-violet-500/[0.2] bg-violet-500/[0.05] px-3 py-2"
                      >
                        <div className="flex items-center gap-2">
                          <span className="text-[0.82rem] font-medium text-foreground">
                            {s.data_point}
                          </span>
                          {s.priority && (
                            <span className={cn(
                              'rounded px-1.5 py-0.5 font-mono text-[0.55rem] font-bold uppercase',
                              s.priority === 'High' && 'border border-destructive/[0.3] bg-destructive/[0.1] text-destructive',
                              s.priority === 'Medium' && 'border border-amber-500/[0.3] bg-amber-500/[0.1] text-amber-600 dark:text-amber-400',
                              s.priority === 'Low' && 'border border-indigo-500/[0.3] bg-indigo-500/[0.1] text-indigo-600 dark:text-indigo-400',
                            )}>
                              {s.priority}
                            </span>
                          )}
                        </div>
                        {s.why_important && (
                          <p className="mt-0.5 text-[0.75rem] text-muted-foreground">
                            {s.why_important}
                          </p>
                        )}
                        {s.where_to_find && (
                          <p className="mt-0.5 text-[0.62rem] font-mono text-muted-foreground/80">
                            Where: {s.where_to_find}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </>
        )}

        {/* ── Tags ── */}
        {contact.tags && Object.keys(contact.tags).length > 0 && (
          <>
            <SectionHeader title="Tags" />
            <div className="flex flex-wrap gap-1.5 py-3">
              {Object.entries(contact.tags).map(([key, val]) => (
                <span
                  key={key}
                  className="rounded px-2 py-0.5 font-mono text-[0.62rem] font-bold border border-indigo-500/[0.25] bg-indigo-500/[0.1] text-indigo-600 dark:text-indigo-400"
                >
                  {key}{val ? `: ${val}` : ''}
                </span>
              ))}
            </div>
          </>
        )}

        {/* ── Message Draft ── */}
        {contact.message_draft && (
          <>
            <SectionHeader title="Draft Message" />
            <div className="mt-1 rounded-lg border border-dashed border-border bg-card px-4 py-3">
              <p className="text-[0.85rem] text-muted-foreground leading-relaxed whitespace-pre-wrap">
                {contact.message_draft}
              </p>
            </div>
          </>
        )}
      </div>
    </ScrollArea>
  );
}

type ParsedMessage = { role: 'me' | 'client'; content: string; channel?: string };

function AddConversationForm({ contactId, contactName }: { contactId: string; contactName?: string }) {
  const queryClient = useQueryClient();
  const [content, setContent] = useState('');
  const [role, setRole] = useState<'me' | 'client'>('me');
  const [channel, setChannel] = useState<InteractionChannel>('LinkedIn');
  const [mode, setMode] = useState<'manual' | 'auto'>('manual');
  const [parsedMessages, setParsedMessages] = useState<ParsedMessage[]>([]);
  const [isParsing, setIsParsing] = useState(false);
  const [isSavingAll, setIsSavingAll] = useState(false);

  const addMutation = useMutation({
    mutationFn: (input: { content: string; role: 'me' | 'client'; channel?: InteractionChannel }) =>
      OutreachApi.addInteraction(contactId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contact-interactions', contactId] });
      queryClient.invalidateQueries({ queryKey: ['contact-outreach-state', contactId] });
    },
    onError: () => toast.error('Failed to save conversation'),
  });

  const handleManualSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!content.trim()) return;
    addMutation.mutate(
      { content, role, channel },
      { onSuccess: () => { setContent(''); toast.success('Conversation saved'); } },
    );
  };

  const handleAutoParse = async () => {
    if (!content.trim()) return;
    setIsParsing(true);
    try {
      const res = await fetch('/api/ai/parse-conversation', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ rawText: content, contactName }),
      });
      if (!res.ok) throw new Error('Parse failed');
      const data = await res.json();
      setParsedMessages(data.messages || []);
    } catch {
      toast.error('Failed to parse conversation');
    } finally {
      setIsParsing(false);
    }
  };

  const handleSaveAll = async () => {
    if (parsedMessages.length === 0) return;
    setIsSavingAll(true);
    try {
      for (const msg of parsedMessages) {
        await OutreachApi.addInteraction(contactId, {
          content: msg.content,
          role: msg.role,
          channel: (msg.channel as InteractionChannel) || channel,
        });
      }
      queryClient.invalidateQueries({ queryKey: ['contact-interactions', contactId] });
      queryClient.invalidateQueries({ queryKey: ['contact-outreach-state', contactId] });
      setParsedMessages([]);
      setContent('');
      toast.success(`Saved ${parsedMessages.length} messages`);
    } catch {
      toast.error('Failed to save some messages');
    } finally {
      setIsSavingAll(false);
    }
  };

  const removeParsedMessage = (index: number) => {
    setParsedMessages((prev) => prev.filter((_, i) => i !== index));
  };

  const toggleParsedRole = (index: number) => {
    setParsedMessages((prev) =>
      prev.map((m, i) => i === index ? { ...m, role: m.role === 'me' ? 'client' : 'me' } : m)
    );
  };

  return (
    <div className="mb-4 rounded-[10px] border border-border bg-card p-3">
      {/* Mode toggle */}
      <div className="mb-2.5 flex items-center gap-2">
        <div className="flex rounded-md border border-border overflow-hidden">
          <button
            type="button"
            onClick={() => { setMode('manual'); setParsedMessages([]); }}
            className={cn(
              'px-3 py-1 text-[0.68rem] font-bold font-mono uppercase tracking-[0.04em] transition-colors',
              mode === 'manual'
                ? 'bg-indigo-500/[0.15] text-indigo-600 dark:text-indigo-400'
                : 'bg-transparent text-muted-foreground/80 hover:text-muted-foreground'
            )}
          >
            Manual
          </button>
          <button
            type="button"
            onClick={() => { setMode('auto'); setParsedMessages([]); }}
            className={cn(
              'px-3 py-1 text-[0.68rem] font-bold font-mono uppercase tracking-[0.04em] transition-colors border-l border-border',
              mode === 'auto'
                ? 'bg-violet-500/[0.15] text-violet-600 dark:text-violet-400'
                : 'bg-transparent text-muted-foreground/80 hover:text-muted-foreground'
            )}
          >
            <Sparkles className="inline h-3 w-3 mr-1" />
            Auto-parse
          </button>
        </div>

        {mode === 'manual' && (
          <>
            <div className="flex rounded-md border border-border overflow-hidden">
              <button
                type="button"
                onClick={() => setRole('me')}
                className={cn(
                  'px-2.5 py-1 text-[0.62rem] font-bold font-mono uppercase tracking-[0.04em] transition-colors',
                  role === 'me'
                    ? 'bg-indigo-500/[0.15] text-indigo-600 dark:text-indigo-400'
                    : 'bg-transparent text-muted-foreground/80 hover:text-muted-foreground'
                )}
              >
                Me
              </button>
              <button
                type="button"
                onClick={() => setRole('client')}
                className={cn(
                  'px-2.5 py-1 text-[0.62rem] font-bold font-mono uppercase tracking-[0.04em] transition-colors border-l border-border',
                  role === 'client'
                    ? 'bg-emerald-500/[0.15] text-emerald-600 dark:text-emerald-400'
                    : 'bg-transparent text-muted-foreground/80 hover:text-muted-foreground'
                )}
              >
                Contact
              </button>
            </div>

            <select
              value={channel}
              onChange={(e) => setChannel(e.target.value as InteractionChannel)}
              className="rounded-md border border-border bg-background px-2.5 py-1 text-[0.68rem] font-mono text-muted-foreground outline-none focus:border-indigo-400"
            >
              {['LinkedIn', 'Email', 'Call', 'Meeting', 'Note'].map((ch) => (
                <option key={ch} value={ch}>{ch}</option>
              ))}
            </select>
          </>
        )}

        {mode === 'auto' && (
          <select
            value={channel}
            onChange={(e) => setChannel(e.target.value as InteractionChannel)}
            className="rounded-md border border-border bg-background px-2.5 py-1 text-[0.68rem] font-mono text-muted-foreground outline-none focus:border-indigo-400"
          >
            {['LinkedIn', 'Email', 'Call', 'Meeting', 'Note'].map((ch) => (
              <option key={ch} value={ch}>{ch}</option>
            ))}
          </select>
        )}
      </div>

      {/* Textarea */}
      <textarea
        value={content}
        onChange={(e) => { setContent(e.target.value); if (parsedMessages.length > 0) setParsedMessages([]); }}
        placeholder={
          mode === 'auto'
            ? 'Paste entire conversation here... AI will split into individual messages'
            : role === 'me' ? 'Paste your message...' : 'Paste contact\'s reply...'
        }
        rows={mode === 'auto' ? 5 : 3}
        className="mb-2 w-full resize-none rounded-md border border-border bg-background px-3 py-2 text-[0.85rem] text-foreground placeholder:text-muted-foreground/80 outline-none focus:border-indigo-400"
        onKeyDown={(e) => {
          if (mode === 'manual' && e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            if (content.trim()) addMutation.mutate(
              { content, role, channel },
              { onSuccess: () => { setContent(''); toast.success('Saved'); } },
            );
          }
        }}
      />

      {/* Parsed preview */}
      {parsedMessages.length > 0 && (
        <div className="mb-2 space-y-1.5">
          <div className="flex items-center justify-between">
            <span className="text-[0.62rem] font-bold font-mono uppercase text-violet-600 dark:text-violet-400">
              Preview ({parsedMessages.length} messages)
            </span>
            <span className="text-[0.55rem] font-mono text-muted-foreground/80">
              Click role to swap, trash to remove
            </span>
          </div>
          {parsedMessages.map((msg, i) => (
            <div
              key={i}
              className={cn(
                'rounded-md border px-3 py-2 text-[0.82rem]',
                msg.role === 'me'
                  ? 'border-indigo-500/[0.25] bg-indigo-500/[0.06]'
                  : 'border-emerald-500/[0.2] bg-emerald-500/[0.06]'
              )}
            >
              <div className="mb-1 flex items-center justify-between">
                <button
                  type="button"
                  onClick={() => toggleParsedRole(i)}
                  className={cn(
                    'text-[0.62rem] font-bold font-mono uppercase tracking-[0.06em] hover:underline',
                    msg.role === 'me' ? 'text-indigo-600 dark:text-indigo-400' : 'text-emerald-600 dark:text-emerald-400'
                  )}
                >
                  {msg.role === 'me' ? 'You' : 'Contact'}
                </button>
                <button
                  type="button"
                  onClick={() => removeParsedMessage(i)}
                  className="text-muted-foreground/80 hover:text-destructive transition-colors"
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              </div>
              <p className="text-muted-foreground whitespace-pre-wrap leading-relaxed">
                {msg.content}
              </p>
            </div>
          ))}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between">
        <span className="text-[0.62rem] text-muted-foreground/80 font-mono">
          {mode === 'manual' ? 'Ctrl+Enter to save' : 'AI-powered parsing (OpenAI)'}
        </span>
        <div className="flex items-center gap-2">
          {mode === 'auto' && parsedMessages.length === 0 && (
            <button
              type="button"
              onClick={handleAutoParse}
              disabled={!content.trim() || isParsing}
              className="flex items-center gap-1.5 rounded-md border border-violet-500/[0.3] bg-violet-500/[0.1] px-3.5 py-1.5 text-[0.75rem] font-bold text-violet-600 dark:text-violet-400 transition-opacity disabled:opacity-40"
            >
              {isParsing ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Sparkles className="h-3.5 w-3.5" />
              )}
              Parse
            </button>
          )}
          {mode === 'auto' && parsedMessages.length > 0 && (
            <button
              type="button"
              onClick={handleSaveAll}
              disabled={isSavingAll}
              className="flex items-center gap-1.5 rounded-md bg-emerald-500 px-3.5 py-1.5 text-[0.75rem] font-bold text-background transition-opacity disabled:opacity-40"
            >
              {isSavingAll ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Check className="h-3.5 w-3.5" />
              )}
              Save All ({parsedMessages.length})
            </button>
          )}
          {mode === 'manual' && (
            <button
              type="button"
              onClick={(e) => handleManualSubmit(e)}
              disabled={!content.trim() || addMutation.isPending}
              className="flex items-center gap-1.5 rounded-md bg-indigo-600 px-3.5 py-1.5 text-[0.75rem] font-bold text-white transition-opacity disabled:opacity-40"
            >
              {addMutation.isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Send className="h-3.5 w-3.5" />
              )}
              Save
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

function ContactConversationTab({ contactId }: { contactId: string }) {
  const { data: contactResponse } = useQuery({
    queryKey: ['contact', contactId],
    queryFn: () => ContactApi.getById(contactId),
    enabled: !!contactId,
  });
  const { data: historyResponse, isLoading } = useQuery({
    queryKey: ['contact-interactions', contactId],
    queryFn: () => OutreachApi.getInteractionHistory(contactId),
    enabled: !!contactId,
  });

  const contactName = contactResponse?.data?.name;
  const interactions = historyResponse?.data || [];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-indigo-600 dark:text-indigo-400" />
      </div>
    );
  }

  return (
    <div className="flex h-[60vh] flex-col">
      <AddConversationForm contactId={contactId} contactName={contactName} />

      {interactions.length === 0 ? (
        <p className="py-4 text-sm text-muted-foreground/80">
          No conversation history yet. Add one above.
        </p>
      ) : (
        <ScrollArea className="flex-1">
          <div className="space-y-3 pr-4">
            {interactions.map((interaction) => {
              const isOutgoing = interaction.direction === 'outgoing';
              return (
                <div
                  key={interaction.id}
                  className={cn(
                    'rounded-[10px] border px-4 py-3.5',
                    isOutgoing
                      ? 'border-indigo-500/[0.25] bg-indigo-500/[0.1]'
                      : 'border-emerald-500/[0.2] bg-emerald-500/[0.1]'
                  )}
                >
                  <div className="mb-2 flex items-center justify-between">
                    <span
                      className={cn(
                        'text-[0.62rem] font-bold uppercase tracking-[0.06em] font-mono',
                        isOutgoing ? 'text-indigo-600 dark:text-indigo-400' : 'text-emerald-600 dark:text-emerald-400'
                      )}
                    >
                      {isOutgoing ? 'You' : interaction.channel}
                    </span>
                    <span className="font-mono text-[0.62rem] text-muted-foreground/80">
                      {new Date(interaction.timestamp).toLocaleString([], {
                        month: 'short',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </span>
                  </div>
                  <p className="text-[0.85rem] leading-relaxed text-muted-foreground whitespace-pre-wrap">
                    {interaction.content}
                  </p>
                </div>
              );
            })}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}

function ContactTimelineTab({ contactId }: { contactId: string }) {
  const queryClient = useQueryClient();
  const [selectedMeeting, setSelectedMeeting] = useState<Meeting | null>(null);
  const [meetingContentOpen, setMeetingContentOpen] = useState(false);
  const [meetingContentInput, setMeetingContentInput] = useState('');

  const { data: stateResponse, isLoading: stateLoading } = useQuery({
    queryKey: ['contact-outreach-state', contactId],
    queryFn: () => OutreachApi.getOutreachState(contactId),
    enabled: !!contactId,
  });

  const { data: historyResponse, isLoading: historyLoading } = useQuery({
    queryKey: ['contact-interactions', contactId],
    queryFn: () => OutreachApi.getInteractionHistory(contactId),
    enabled: !!contactId,
  });

  const { data: meetingsResponse } = useQuery({
    queryKey: ['contact-meetings', contactId],
    queryFn: () => OutreachApi.getMeetings(contactId),
    enabled: !!contactId,
  });

  const updateMeetingMutation = useMutation({
    mutationFn: ({ meetingId, input }: { meetingId: string; input: Parameters<typeof OutreachApi.updateMeeting>[1] }) =>
      OutreachApi.updateMeeting(meetingId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contact-meetings', contactId] });
      setMeetingContentOpen(false);
      setMeetingContentInput('');
      setSelectedMeeting(null);
      toast.success('Meeting updated');
    },
    onError: () => toast.error('Failed to update meeting'),
  });

  const state = stateResponse?.data;
  const interactions = historyResponse?.data || [];
  const meetings = meetingsResponse?.data || [];

  function openContentDialog(m: Meeting) {
    setSelectedMeeting(m);
    setMeetingContentInput(m.meeting_content || '');
    setMeetingContentOpen(true);
  }

  if (stateLoading || historyLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-indigo-600 dark:text-indigo-400" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {state && (
        <div className="space-y-2 rounded-lg border border-border bg-muted/50 p-3">
          <p className="text-xs font-medium text-muted-foreground">Current State</p>
          <div className="flex flex-wrap items-center gap-2">
            <Badge className="bg-indigo-500/[0.1] text-indigo-600 dark:text-indigo-400 border border-indigo-500/[0.25]">{state.conversation_state}</Badge>
            <span className="text-xs text-muted-foreground/80">→</span>
            <Badge variant="outline" className="border-border text-muted-foreground">{state.next_step}</Badge>
          </div>
          <div className="flex gap-4 text-xs text-muted-foreground/80">
            <span>Follow-ups: {state.followup_count}/{state.max_followups}</span>
            <span>{state.days_since_last_interaction} days since last interaction</span>
          </div>
        </div>
      )}

      {meetings.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-bold uppercase tracking-[0.06em] text-muted-foreground/80 font-mono">Meetings</p>
          {meetings.map((m) => (
            <div key={m.id} className="rounded-lg border border-border bg-muted/50 p-3 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-[0.82rem] font-medium text-foreground">{m.title || 'Meeting'}</span>
                <span className={`text-[0.68rem] font-bold px-2 py-0.5 rounded font-mono ${
                  m.status === 'scheduled' ? 'bg-indigo-500/[0.15] text-indigo-600 dark:text-indigo-400' :
                  m.status === 'completed' ? 'bg-emerald-500/[0.15] text-emerald-600 dark:text-emerald-400' :
                  'bg-border/[0.15] text-muted-foreground/80'
                }`}>{m.status}</span>
              </div>
              <p className="text-[0.75rem] text-muted-foreground/80">
                {new Date(m.time).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })} · {m.channel} · {m.duration_minutes}min
              </p>
              {m.outcome && <p className="text-[0.78rem] text-muted-foreground">Outcome: {m.outcome}</p>}
              {m.next_steps && <p className="text-[0.78rem] text-muted-foreground">Next: {m.next_steps}</p>}
              {m.meeting_content && (
                <p className="text-[0.75rem] text-muted-foreground/80 line-clamp-2 italic">{m.meeting_content}</p>
              )}
              <div className="flex gap-2 pt-1">
                {m.status === 'scheduled' && (
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-6 px-2 text-[0.7rem] border-border text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/[0.1]"
                    onClick={() => updateMeetingMutation.mutate({ meetingId: m.id, input: { status: 'completed' } })}
                    disabled={updateMeetingMutation.isPending}
                  >
                    <CheckCircle className="mr-1 h-3 w-3" />
                    Mark Completed
                  </Button>
                )}
                {m.status === 'completed' && (
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-6 px-2 text-[0.7rem] border-border text-indigo-600 dark:text-indigo-400 hover:bg-indigo-500/[0.1]"
                    onClick={() => openContentDialog(m)}
                  >
                    <Pencil className="mr-1 h-3 w-3" />
                    {m.meeting_content ? 'Edit Notes' : 'Add Notes'}
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {interactions.length === 0 && meetings.length === 0 ? (
        <p className="py-4 text-sm text-muted-foreground/80">No timeline events yet.</p>
      ) : interactions.length > 0 ? (
        <ScrollArea className="h-[50vh]">
          <div className="relative space-y-0 pl-[22px] pr-4">
            <div className="absolute bottom-2 left-[7px] top-2 w-[2px] bg-border rounded" />
            {interactions.map((interaction) => (
              <div key={interaction.id} className="relative flex gap-3.5 border-b border-border py-3 pl-4">
                <div className={cn(
                  'absolute -left-[2px] top-[17px] h-[10px] w-[10px] shrink-0 rounded-full border-2 border-background',
                  interaction.direction === 'incoming' ? 'bg-emerald-500' :
                  interaction.direction === 'outgoing' ? 'bg-indigo-600' : 'bg-amber-500'
                )} />
                <div className="flex-1">
                  <span className="font-mono text-[0.62rem] text-muted-foreground/80">
                    {new Date(interaction.timestamp).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                  </span>
                  <p className="mt-0.5 text-[0.82rem] text-muted-foreground">{interaction.content}</p>
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>
      ) : null}

      <Dialog open={meetingContentOpen} onOpenChange={setMeetingContentOpen}>
        <DialogContent className="bg-background border-border text-foreground">
          <DialogHeader>
            <DialogTitle className="text-foreground">
              {selectedMeeting?.meeting_content ? 'Edit Meeting Notes' : 'Add Meeting Notes'}
            </DialogTitle>
            <DialogDescription className="text-muted-foreground/80">
              {selectedMeeting?.title || 'Meeting'} · {selectedMeeting && new Date(selectedMeeting.time).toLocaleDateString()}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            value={meetingContentInput}
            onChange={(e) => setMeetingContentInput(e.target.value)}
            placeholder="Enter meeting notes, transcript, or outcome details..."
            className="min-h-[120px] bg-card border-border text-foreground placeholder:text-muted-foreground/80 resize-none"
          />
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              className="border-border text-muted-foreground"
              onClick={() => setMeetingContentOpen(false)}
            >
              Cancel
            </Button>
            <Button
              className="bg-indigo-600 text-white hover:bg-indigo-700"
              onClick={() => selectedMeeting && updateMeetingMutation.mutate({
                meetingId: selectedMeeting.id,
                input: { meeting_content: meetingContentInput },
              })}
              disabled={updateMeetingMutation.isPending}
            >
              {updateMeetingMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Save'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ContactNotesTab({ contactId: _contactId }: { contactId: string }) {
  return (
    <p className="py-8 text-center text-sm text-muted-foreground/80">
      No notes yet.
    </p>
  );
}

/** AI Intelligence tab — enrich, score, meeting brief, find similar */
function ContactIntelligenceTab({ contactId }: { contactId: string }) {
  const queryClient = useQueryClient();
  const [enrichData, setEnrichData] = useState<ContactEnrichmentResponse | null>(null);
  const [scoresData, setScoresData] = useState<CalculateScoresResponse | null>(null);
  const [briefData, setBriefData] = useState<MeetingBriefResponse | null>(null);
  const [briefTab, setBriefTab] = useState<'talking' | 'questions' | 'risks'>('talking');
  const [addFindingFor, setAddFindingFor] = useState<{ category: string; data_point: string; why_important: string; where_to_find: string; priority: string } | null>(null);
  const [findingForm, setFindingForm] = useState({ field_name: '', value: '', source: '' });

  const enrichMutation = useMutation({
    mutationFn: () => IntelligenceApi.enrichContact(contactId, true),
    onSuccess: (res) => { setEnrichData(res.data); queryClient.invalidateQueries({ queryKey: ['contact', contactId] }); toast.success('Enrichment complete'); },
    onError: () => toast.error('Enrichment failed'),
  });

  const scoreMutation = useMutation({
    mutationFn: () => IntelligenceApi.calculateScores(contactId),
    onSuccess: (res) => { setScoresData(res.data); toast.success('Scoring complete'); },
    onError: () => toast.error('Scoring failed'),
  });

  const briefMutation = useMutation({
    mutationFn: () => IntelligenceApi.generateMeetingBrief(contactId),
    onSuccess: (res) => { setBriefData(res.data); toast.success('Brief generated'); },
    onError: () => toast.error('Brief generation failed'),
  });

  const validateMutation = useMutation({
    mutationFn: (p: { type: 'pain_point' | 'goal'; text: string; validation: 'confirmed' | 'rejected' }) =>
      IntelligenceApi.validateInsight(contactId, { insight_type: p.type, insight_text: p.text, validation: p.validation }),
    onSuccess: (_, vars) => {
      setEnrichData((prev) => {
        if (!prev) return prev;
        const updated = { ...prev, ai_insights: { ...prev.ai_insights } };
        if (vars.type === 'pain_point') {
          updated.ai_insights.suspected_pain_points = prev.ai_insights.suspected_pain_points.filter((p) => p.pain_point !== vars.text);
        } else {
          updated.ai_insights.suspected_goals = prev.ai_insights.suspected_goals.filter((g) => g.goal !== vars.text);
        }
        return updated;
      });
      queryClient.invalidateQueries({ queryKey: ['contact', contactId] });
      toast.success(vars.validation === 'confirmed' ? 'Insight confirmed' : 'Insight rejected');
    },
    onError: () => toast.error('Validation failed'),
  });

  const addFindingMutation = useMutation({
    mutationFn: () =>
      IntelligenceApi.addResearchFinding(contactId, {
        category: addFindingFor!.category,
        field_name: findingForm.field_name,
        value: findingForm.value,
        source: findingForm.source || undefined,
        priority: addFindingFor!.priority,
        why_important: addFindingFor!.why_important,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['contact', contactId] });
      toast.success('Finding added');
      setAddFindingFor(null);
      setFindingForm({ field_name: '', value: '', source: '' });
    },
    onError: () => toast.error('Failed to add finding'),
  });

  return (
    <ScrollArea className="h-[60vh]">
      <div className="pr-4 space-y-5">
        {/* ── Quick Actions ── */}
        <div className="grid grid-cols-3 gap-2">
          <button
            onClick={() => enrichMutation.mutate()}
            disabled={enrichMutation.isPending}
            className="flex flex-col items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-3 text-[0.72rem] font-bold text-muted-foreground hover:bg-muted hover:text-violet-600 dark:hover:text-violet-400 transition-colors disabled:opacity-50"
          >
            {enrichMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            Enrich
          </button>
          <button
            onClick={() => scoreMutation.mutate()}
            disabled={scoreMutation.isPending}
            className="flex flex-col items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-3 text-[0.72rem] font-bold text-muted-foreground hover:bg-muted hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors disabled:opacity-50"
          >
            {scoreMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <BarChart3 className="h-4 w-4" />}
            Score
          </button>
          <button
            onClick={() => briefMutation.mutate()}
            disabled={briefMutation.isPending}
            className="flex flex-col items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-3 text-[0.72rem] font-bold text-muted-foreground hover:bg-muted hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors disabled:opacity-50"
          >
            {briefMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <MessageSquare className="h-4 w-4" />}
            Brief
          </button>
        </div>

        {/* ── Enrichment Results ── */}
        {enrichData && (
          <div className="space-y-4">
            {/* Stats */}
            <div className="grid grid-cols-3 gap-2">
              <div className="rounded-md border border-border bg-card px-3 py-2 text-center">
                <p className="text-[0.6rem] font-mono text-muted-foreground/80 uppercase">Insights</p>
                <p className="text-[1.1rem] font-bold text-foreground">{enrichData.insights_generated}</p>
              </div>
              <div className="rounded-md border border-border bg-card px-3 py-2 text-center">
                <p className="text-[0.6rem] font-mono text-muted-foreground/80 uppercase">Confidence</p>
                <p className="text-[1.1rem] font-bold text-foreground">{Math.round(enrichData.confidence_avg * 100)}%</p>
              </div>
              <div className="rounded-md border border-border bg-card px-3 py-2 text-center">
                <p className="text-[0.6rem] font-mono text-muted-foreground/80 uppercase">Embedding</p>
                <p className="text-[1.1rem] font-bold">{enrichData.embedding_created ? <span className="text-emerald-600 dark:text-emerald-400">Yes</span> : <span className="text-muted-foreground/80">No</span>}</p>
              </div>
            </div>

            {/* Pain Points */}
            {enrichData.ai_insights.suspected_pain_points?.length > 0 && (
              <>
                <SectionHeader title="Pain Points" />
                <div className="space-y-2">
                  {enrichData.ai_insights.suspected_pain_points.map((pp) => (
                    <div key={pp.pain_point} className="rounded-md border border-border bg-card px-3 py-2.5">
                      <div className="flex items-start justify-between gap-2">
                        <p className="text-[0.82rem] text-foreground leading-relaxed">{pp.pain_point}</p>
                        <span className="shrink-0 rounded px-1.5 py-0.5 text-[0.6rem] font-mono font-bold border border-indigo-500/[0.25] bg-indigo-500/[0.1] text-indigo-600 dark:text-indigo-400">
                          {Math.round(pp.confidence * 100)}%
                        </span>
                      </div>
                      {pp.evidence?.length > 0 && (
                        <ul className="mt-1.5 space-y-0.5">
                          {pp.evidence.map((ev) => (
                            <li key={ev} className="text-[0.75rem] text-muted-foreground/80 pl-3 relative before:content-['·'] before:absolute before:left-0 before:text-muted-foreground/80">{ev}</li>
                          ))}
                        </ul>
                      )}
                      <div className="flex gap-1.5 mt-2">
                        <button
                          onClick={() => validateMutation.mutate({ type: 'pain_point', text: pp.pain_point, validation: 'confirmed' })}
                          disabled={validateMutation.isPending}
                          className="flex items-center gap-1 rounded px-2 py-1 text-[0.68rem] font-bold border border-emerald-500/[0.25] bg-emerald-500/[0.08] text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/[0.15] disabled:opacity-50"
                        >
                          <CheckCircle className="h-3 w-3" /> Confirm
                        </button>
                        <button
                          onClick={() => validateMutation.mutate({ type: 'pain_point', text: pp.pain_point, validation: 'rejected' })}
                          disabled={validateMutation.isPending}
                          className="flex items-center gap-1 rounded px-2 py-1 text-[0.68rem] font-bold border border-destructive/[0.25] bg-destructive/[0.08] text-destructive hover:bg-destructive/[0.15] disabled:opacity-50"
                        >
                          <XCircle className="h-3 w-3" /> Reject
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}

            {/* Goals */}
            {enrichData.ai_insights.suspected_goals?.length > 0 && (
              <>
                <SectionHeader title="Goals" />
                <div className="space-y-2">
                  {enrichData.ai_insights.suspected_goals.map((g) => (
                    <div key={g.goal} className="rounded-md border border-border bg-card px-3 py-2.5">
                      <div className="flex items-start justify-between gap-2">
                        <p className="text-[0.82rem] text-foreground leading-relaxed">{g.goal}</p>
                        <span className="shrink-0 rounded px-1.5 py-0.5 text-[0.6rem] font-mono font-bold border border-indigo-500/[0.25] bg-indigo-500/[0.1] text-indigo-600 dark:text-indigo-400">
                          {Math.round(g.confidence * 100)}%
                        </span>
                      </div>
                      {g.evidence?.length > 0 && (
                        <ul className="mt-1.5 space-y-0.5">
                          {g.evidence.map((ev) => (
                            <li key={ev} className="text-[0.75rem] text-muted-foreground/80 pl-3 relative before:content-['·'] before:absolute before:left-0 before:text-muted-foreground/80">{ev}</li>
                          ))}
                        </ul>
                      )}
                      <div className="flex gap-1.5 mt-2">
                        <button
                          onClick={() => validateMutation.mutate({ type: 'goal', text: g.goal, validation: 'confirmed' })}
                          disabled={validateMutation.isPending}
                          className="flex items-center gap-1 rounded px-2 py-1 text-[0.68rem] font-bold border border-emerald-500/[0.25] bg-emerald-500/[0.08] text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/[0.15] disabled:opacity-50"
                        >
                          <CheckCircle className="h-3 w-3" /> Confirm
                        </button>
                        <button
                          onClick={() => validateMutation.mutate({ type: 'goal', text: g.goal, validation: 'rejected' })}
                          disabled={validateMutation.isPending}
                          className="flex items-center gap-1 rounded px-2 py-1 text-[0.68rem] font-bold border border-destructive/[0.25] bg-destructive/[0.08] text-destructive hover:bg-destructive/[0.15] disabled:opacity-50"
                        >
                          <XCircle className="h-3 w-3" /> Reject
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}

            {/* Buying Signals */}
            {enrichData.ai_insights.buying_signals?.length > 0 && (
              <>
                <SectionHeader title="Buying Signals" />
                <div className="space-y-2">
                  {enrichData.ai_insights.buying_signals.map((bs) => (
                    <div key={`${bs.signal}-${bs.occurred_at}`} className="flex items-center justify-between rounded-md border border-border bg-card px-3 py-2.5">
                      <div>
                        <p className="text-[0.82rem] text-foreground">{bs.signal}</p>
                        <p className="text-[0.7rem] text-muted-foreground/80">{new Date(bs.occurred_at).toLocaleDateString()}</p>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <span className={cn(
                          'rounded px-1.5 py-0.5 text-[0.6rem] font-mono font-bold border',
                          bs.strength === 'High' ? 'border-emerald-500/[0.25] bg-emerald-500/[0.1] text-emerald-600 dark:text-emerald-400'
                            : bs.strength === 'Medium' ? 'border-amber-500/[0.25] bg-amber-500/[0.1] text-amber-600 dark:text-amber-400'
                            : 'border-border bg-muted text-muted-foreground/80'
                        )}>{bs.strength}</span>
                        <span className="rounded px-1.5 py-0.5 text-[0.6rem] font-mono border border-border bg-muted text-muted-foreground/80">
                          {bs.recency_score}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}

            {/* Decision Style */}
            {enrichData.ai_insights.decision_style && (
              <>
                <SectionHeader title="Decision Style" />
                <div className="rounded-md border border-border bg-card px-3 py-2.5">
                  <div className="flex items-center justify-between">
                    <p className="text-[0.85rem] font-bold text-foreground">{enrichData.ai_insights.decision_style.type}</p>
                    <span className="rounded px-1.5 py-0.5 text-[0.6rem] font-mono font-bold border border-indigo-500/[0.25] bg-indigo-500/[0.1] text-indigo-600 dark:text-indigo-400">
                      {Math.round(enrichData.ai_insights.decision_style.confidence * 100)}%
                    </span>
                  </div>
                  {enrichData.ai_insights.decision_style.key_decision_factors?.length > 0 && (
                    <div className="flex flex-wrap gap-1.5 mt-2">
                      {enrichData.ai_insights.decision_style.key_decision_factors.map((f) => (
                        <span key={f} className="rounded px-2 py-0.5 text-[0.65rem] font-mono border border-border bg-muted text-muted-foreground">{f}</span>
                      ))}
                    </div>
                  )}
                </div>
              </>
            )}

            {/* Research Suggestions */}
            {enrichData.ai_insights.research_suggestions && enrichData.ai_insights.research_suggestions.length > 0 && (
              <>
                <SectionHeader title="Research Suggestions" />
                <div className="space-y-2">
                  {enrichData.ai_insights.research_suggestions.map((s, i) => (
                    <div key={i} className="rounded-lg border border-dashed border-indigo-500/[0.3] bg-indigo-500/[0.04] px-3 py-2.5">
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex-1">
                          <div className="flex items-center gap-1.5 mb-1">
                            <span className="rounded px-1.5 py-0.5 text-[0.58rem] font-mono font-bold border border-violet-500/[0.25] bg-violet-500/[0.1] text-violet-600 dark:text-violet-400">{s.category}</span>
                            <span className={cn(
                              'rounded px-1.5 py-0.5 text-[0.58rem] font-mono font-bold border',
                              s.priority === 'High' ? 'border-destructive/[0.25] bg-destructive/[0.1] text-destructive'
                                : s.priority === 'Medium' ? 'border-amber-500/[0.25] bg-amber-500/[0.1] text-amber-600 dark:text-amber-400'
                                : 'border-indigo-500/[0.25] bg-indigo-500/[0.1] text-indigo-600 dark:text-indigo-400'
                            )}>{s.priority}</span>
                          </div>
                          <p className="text-[0.82rem] font-medium text-foreground">{s.data_point}</p>
                          <p className="text-[0.72rem] text-muted-foreground/80 mt-0.5">{s.why_important}</p>
                        </div>
                        <button
                          onClick={() => { setAddFindingFor(s); setFindingForm({ field_name: '', value: '', source: '' }); }}
                          className="shrink-0 flex items-center gap-1 rounded px-2 py-1 text-[0.68rem] font-bold border border-indigo-500/[0.25] bg-indigo-500/[0.08] text-indigo-600 dark:text-indigo-400 hover:bg-indigo-500/[0.15]"
                        >
                          <Plus className="h-3 w-3" /> Add
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>
        )}

        {/* ── Add Finding Inline Form ── */}
        {addFindingFor && (
          <div className="rounded-lg border border-dashed border-violet-500/[0.3] bg-violet-500/[0.06] px-4 py-3 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-[0.78rem] font-bold text-violet-600 dark:text-violet-400">Add Finding: {addFindingFor.category}</span>
              <button onClick={() => setAddFindingFor(null)} className="text-muted-foreground/80 hover:text-muted-foreground"><X className="h-3.5 w-3.5" /></button>
            </div>
            <input
              type="text"
              placeholder="Field name (e.g. Company Size)"
              value={findingForm.field_name}
              onChange={(e) => setFindingForm((p) => ({ ...p, field_name: e.target.value }))}
              className="w-full rounded-md border border-border bg-card px-3 py-2 text-[0.82rem] text-foreground placeholder:text-muted-foreground/80 focus:border-violet-400 focus:outline-none"
            />
            <textarea
              placeholder="Value"
              value={findingForm.value}
              onChange={(e) => setFindingForm((p) => ({ ...p, value: e.target.value }))}
              rows={2}
              className="w-full rounded-md border border-border bg-card px-3 py-2 text-[0.82rem] text-foreground placeholder:text-muted-foreground/80 focus:border-violet-400 focus:outline-none resize-none"
            />
            <input
              type="text"
              placeholder={`Source (hint: ${addFindingFor.where_to_find})`}
              value={findingForm.source}
              onChange={(e) => setFindingForm((p) => ({ ...p, source: e.target.value }))}
              className="w-full rounded-md border border-border bg-card px-3 py-2 text-[0.82rem] text-foreground placeholder:text-muted-foreground/80 focus:border-violet-400 focus:outline-none"
            />
            <button
              onClick={() => addFindingMutation.mutate()}
              disabled={addFindingMutation.isPending || !findingForm.field_name || !findingForm.value}
              className="flex items-center gap-1.5 rounded-md bg-violet-500 px-4 py-2 text-[0.78rem] font-bold text-white hover:bg-violet-700 disabled:opacity-50"
            >
              {addFindingMutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
              Save Finding
            </button>
          </div>
        )}

        {/* ── Segment Scores ── */}
        {scoresData && (
          <div className="space-y-3">
            <SectionHeader title="Segment Scores" />
            <div className="grid grid-cols-2 gap-2">
              <div className="rounded-md border border-border bg-card px-3 py-2 text-center">
                <p className="text-[0.6rem] font-mono text-muted-foreground/80 uppercase">Evaluated</p>
                <p className="text-[1.1rem] font-bold text-foreground">{scoresData.segments_evaluated}</p>
              </div>
              <div className="rounded-md border border-border bg-card px-3 py-2 text-center">
                <p className="text-[0.6rem] font-mono text-muted-foreground/80 uppercase">Priority</p>
                <p className="text-[1.1rem] font-bold text-indigo-600 dark:text-indigo-400">{scoresData.priority_score}</p>
              </div>
            </div>
            {scoresData.scores.map((s) => (
              <div key={s.segmentation_id} className="rounded-md border border-border bg-card px-3 py-2.5">
                <div className="flex items-center justify-between mb-1.5">
                  <p className="text-[0.82rem] font-bold text-foreground">{s.segmentation_name}</p>
                  <div className="flex items-center gap-1.5">
                    <span className={cn(
                      'rounded px-1.5 py-0.5 text-[0.6rem] font-mono font-bold border',
                      s.fit_score >= 80 ? 'border-emerald-500/[0.25] bg-emerald-500/[0.1] text-emerald-600 dark:text-emerald-400'
                        : s.fit_score >= 60 ? 'border-amber-500/[0.25] bg-amber-500/[0.1] text-amber-600 dark:text-amber-400'
                        : 'border-destructive/[0.25] bg-destructive/[0.1] text-destructive'
                    )}>{s.fit_score}%</span>
                    {s.passes_filters && (
                      <span className="rounded px-1.5 py-0.5 text-[0.58rem] font-mono border border-emerald-500/[0.25] bg-emerald-500/[0.08] text-emerald-600 dark:text-emerald-400">Pass</span>
                    )}
                  </div>
                </div>
                {Object.keys(s.score_breakdown).length > 0 && (
                  <div className="grid grid-cols-2 gap-x-4 gap-y-0.5 mt-1">
                    {Object.entries(s.score_breakdown).map(([k, v]) => (
                      <div key={k} className="flex justify-between text-[0.7rem]">
                        <span className="text-muted-foreground/80 font-mono">{k}</span>
                        <span className="text-muted-foreground font-bold">{v}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {/* ── Meeting Brief ── */}
        {briefData && (
          <div className="space-y-3">
            <SectionHeader title="Meeting Brief" />
            {/* Sub-tabs */}
            <div className="flex gap-1.5">
              {([['talking', 'Talking Points'], ['questions', 'Questions'], ['risks', 'Risks']] as const).map(([key, label]) => (
                <button
                  key={key}
                  onClick={() => setBriefTab(key)}
                  className={cn(
                    'rounded px-3 py-1 text-[0.7rem] font-bold font-mono transition-colors',
                    briefTab === key
                      ? 'bg-indigo-600 text-white'
                      : 'bg-card border border-border text-muted-foreground/80 hover:text-muted-foreground'
                  )}
                >
                  {label}
                </button>
              ))}
            </div>
            <div className="space-y-1.5">
              {briefTab === 'talking' && briefData.talking_points.map((tp, i) => (
                <div key={i} className="flex gap-2 rounded-md border border-border bg-card px-3 py-2">
                  <span className="shrink-0 text-[0.7rem] font-bold text-indigo-600 dark:text-indigo-400 font-mono">{i + 1}.</span>
                  <p className="text-[0.82rem] text-foreground leading-relaxed">{tp}</p>
                </div>
              ))}
              {briefTab === 'questions' && briefData.discovery_questions.map((q, i) => (
                <div key={i} className="flex gap-2 rounded-md border border-border bg-card px-3 py-2">
                  <Search className="shrink-0 h-3.5 w-3.5 mt-0.5 text-violet-600 dark:text-violet-400" />
                  <p className="text-[0.82rem] text-foreground leading-relaxed">{q}</p>
                </div>
              ))}
              {briefTab === 'risks' && briefData.risk_flags.map((r, i) => (
                <div key={i} className="flex gap-2 rounded-md border border-amber-500/[0.25] bg-amber-500/[0.06] px-3 py-2">
                  <AlertTriangle className="shrink-0 h-3.5 w-3.5 mt-0.5 text-amber-600 dark:text-amber-400" />
                  <p className="text-[0.82rem] text-foreground leading-relaxed">{r}</p>
                </div>
              ))}
            </div>

            {/* Recent Interactions & Facts */}
            {briefData.last_interactions?.length > 0 && (
              <>
                <SectionHeader title="Recent Interactions" />
                <div className="space-y-1.5">
                  {briefData.last_interactions.map((inter, i) => (
                    <div key={i} className="rounded-md border border-border bg-card px-3 py-2">
                      <div className="flex items-center justify-between">
                        <span className="text-[0.7rem] font-mono font-bold text-muted-foreground uppercase">{inter.type}</span>
                        <span className="text-[0.65rem] font-mono text-muted-foreground/80">{new Date(inter.date).toLocaleDateString()}</span>
                      </div>
                      <p className="text-[0.82rem] text-foreground mt-0.5">{inter.summary}</p>
                    </div>
                  ))}
                </div>
              </>
            )}

            {briefData.segments?.length > 0 && (
              <>
                <SectionHeader title="Segment Fit" />
                <div className="flex flex-wrap gap-2">
                  {briefData.segments.map((seg) => (
                    <span key={seg.name} className="flex items-center gap-1.5 rounded px-2 py-1 text-[0.72rem] font-mono border border-border bg-card text-muted-foreground">
                      {seg.name}
                      <span className={cn(
                        'font-bold',
                        seg.fit_score >= 80 ? 'text-emerald-600 dark:text-emerald-400' : seg.fit_score >= 60 ? 'text-amber-600 dark:text-amber-400' : 'text-destructive'
                      )}>{seg.fit_score}%</span>
                    </span>
                  ))}
                </div>
              </>
            )}
          </div>
        )}

        {/* Empty state */}
        {!enrichData && !scoresData && !briefData && (
          <div className="text-center py-8">
            <Brain className="h-8 w-8 text-border mx-auto mb-3" />
            <p className="text-[0.85rem] text-muted-foreground/80">Click a button above to run AI intelligence</p>
          </div>
        )}
      </div>
    </ScrollArea>
  );
}

function PanelContactHeader({ contactId }: { contactId: string }) {
  const { data: contactResponse } = useQuery({
    queryKey: ['contact', contactId],
    queryFn: () => ContactApi.getById(contactId),
    enabled: !!contactId,
  });

  const contact = contactResponse?.data;
  if (!contact) return null;

  return (
    <div className="px-7 pt-6 pb-3.5">
      <div className="mb-3.5 flex items-center gap-3.5">
        <ContactAvatar name={contact.name} size={44} />
        <div>
          <p className="text-[1.08rem] font-extrabold text-foreground">
            {contact.name}
          </p>
          <p className="text-[0.85rem] text-muted-foreground">
            {contact.job_title && `${contact.job_title} · `}
            {contact.company}
          </p>
        </div>
      </div>
      <div className="flex flex-wrap gap-1.5">
        {contact.status && (
          <span
            className={cn(
              'font-mono text-[0.62rem] font-bold uppercase tracking-[0.06em] rounded px-2 py-0.5',
              contact.status === 'ready'
                ? 'border border-emerald-500/[0.2] bg-emerald-500/[0.1] text-emerald-600 dark:text-emerald-400'
                : 'border border-dashed border-amber-500/[0.3] bg-amber-500/[0.1] text-amber-600 dark:text-amber-400'
            )}
          >
            {contact.status === 'ready' ? 'Approved' : 'Pending'}
          </span>
        )}
        {contact.outreach_stage && (
          <span className="font-mono text-[0.62rem] font-bold uppercase rounded bg-indigo-500/[0.1] px-2 py-0.5 text-indigo-600 dark:text-indigo-400 border border-indigo-500/[0.25]">
            {contact.outreach_stage}
          </span>
        )}
        {contact.source && (
          <span className="font-mono text-[0.62rem] font-medium rounded bg-muted px-2 py-0.5 text-muted-foreground/80 border border-border">
            {contact.source}
          </span>
        )}
        {contact.next_step && (
          <span className="font-mono text-[0.62rem] font-bold rounded bg-indigo-600 px-2 py-0.5 text-white">
            {contact.next_step}
          </span>
        )}
      </div>
    </div>
  );
}

export function ContactDetailPanel() {
  const [activePanel, setActivePanel] = useAtom(activePanelAtom);
  const isOpen = activePanel !== null;

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && setActivePanel(null)}>
      <SheetContent side="right" className="w-full sm:w-[480px] sm:max-w-none p-0 overflow-hidden flex flex-col bg-background border-l border-border text-foreground">

        {activePanel && (
          <>
            <PanelContactHeader contactId={activePanel.contactId} />
          <Tabs defaultValue="overview" className="mt-0">
            <TabsList className="flex h-auto w-full justify-start rounded-none border-b border-border bg-transparent px-7">
              {(['overview', 'intel', 'conversation', 'timeline', 'notes'] as const).map(
                (tab) => (
                  <TabsTrigger
                    key={tab}
                    value={tab}
                    className="mr-5 -mb-px rounded-none border-b-2 border-transparent px-0 py-2.5 text-[0.68rem] font-bold uppercase tracking-[0.05em] text-muted-foreground/80 font-mono shadow-none transition-colors hover:text-muted-foreground data-[state=active]:border-indigo-400 data-[state=active]:bg-transparent data-[state=active]:text-indigo-600 dark:text-indigo-400 data-[state=active]:shadow-none"
                  >
                    {tab}
                  </TabsTrigger>
                )
              )}
            </TabsList>
            <TabsContent value="overview" className="px-7 pt-6">
              <ContactOverviewTab contactId={activePanel.contactId} />
            </TabsContent>
            <TabsContent value="intel" className="px-7 pt-6">
              <ContactIntelligenceTab contactId={activePanel.contactId} />
            </TabsContent>
            <TabsContent value="conversation" className="px-7 pt-6">
              <ContactConversationTab contactId={activePanel.contactId} />
            </TabsContent>
            <TabsContent value="timeline" className="px-7 pt-6">
              <ContactTimelineTab contactId={activePanel.contactId} />
            </TabsContent>
            <TabsContent value="notes" className="px-7 pt-6">
              <ContactNotesTab contactId={activePanel.contactId} />
            </TabsContent>
          </Tabs>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

function InsightItem({
  label,
  text,
  type,
  contactId,
}: {
  label: string;
  text: string;
  type: string;
  contactId: string;
}) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<'idle' | 'confirmed' | 'rejected'>('idle');

  const validateMutation = useMutation({
    mutationFn: (validation: 'confirmed' | 'rejected') =>
      IntelligenceApi.validateInsight(contactId, {
        insight_type: type,
        insight_text: text,
        validation,
      }),
    onSuccess: (_, validation) => {
      setStatus(validation);
      toast.success(validation === 'confirmed' ? 'Insight confirmed' : 'Insight rejected');
      queryClient.invalidateQueries({ queryKey: ['contact', contactId] });
    },
    onError: () => {
      toast.error('Failed to validate insight');
    },
  });

  if (status === 'confirmed') {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-emerald-500/[0.25] bg-emerald-500/[0.06] px-4 py-2.5">
        <CheckCircle className="h-3.5 w-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
        <span className="text-[0.68rem] font-bold uppercase tracking-[0.04em] text-emerald-600 dark:text-emerald-400 font-mono">{label}</span>
        <span className="text-[0.82rem] text-muted-foreground">{text}</span>
      </div>
    );
  }

  if (status === 'rejected') return null;

  return (
    <div className="flex items-start gap-2 rounded-lg border border-border bg-muted/50 px-4 py-2.5">
      <div className="flex-1 min-w-0">
        <span className="text-[0.68rem] font-bold uppercase tracking-[0.04em] text-muted-foreground/80 font-mono">{label}</span>
        <p className="mt-0.5 text-[0.82rem] text-foreground leading-relaxed">{text}</p>
      </div>
      <div className="flex shrink-0 gap-1">
        <button
          onClick={() => validateMutation.mutate('confirmed')}
          disabled={validateMutation.isPending}
          className="rounded p-1 text-muted-foreground/80 transition-colors hover:bg-emerald-500/[0.1] hover:text-emerald-600 dark:hover:text-emerald-400"
          title="Confirm insight"
        >
          <CheckCircle className="h-4 w-4" />
        </button>
        <button
          onClick={() => validateMutation.mutate('rejected')}
          disabled={validateMutation.isPending}
          className="rounded p-1 text-muted-foreground/80 transition-colors hover:bg-destructive/[0.1] hover:text-destructive"
          title="Reject insight"
        >
          <XCircle className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
