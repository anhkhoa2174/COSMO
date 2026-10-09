'use client';

import { ContentLayout } from '@/components/nav/content-layout';
import { ContactEnrichmentButton } from '@/components/intelligence/contact-enrichment-button';
import { VectorSearchDialog } from '@/components/intelligence/vector-search-dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHero } from '@/components/ui/page-hero';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Brain,
  Layers,
  Loader2,
  Plus,
  Search,
  Sparkles,
  Target,
} from 'lucide-react';
import Link from 'next/link';
import React from 'react';
import { resolveContactEmail } from '@/lib/contact-info';
import { useQuery, useMutation } from '@tanstack/react-query';
import ContactApi from '@/network/client/contact';
import IntelligenceApi, {
  type CalculateScoresResponse,
} from '@/network/client/intelligence';
import { useSegmentationsQuery } from '@/network/client/segmentation';
import { cn } from '@/lib/utils';
import { toast } from 'sonner';

export default function SmartInsightsPage() {
  const [contactId, setContactId] = React.useState<string>('');
  const [scoresData, setScoresData] =
    React.useState<CalculateScoresResponse | null>(null);

  const { data: contacts, isLoading: loadingContacts } = useQuery({
    queryKey: ['smart-insights-contacts'],
    queryFn: () => ContactApi.list({ filter: {} }, { offset: 0, limit: 200 }),
  });

  const { data: segmentsRes, isLoading: loadingSegments } =
    useSegmentationsQuery();

  // Radix throws on a SelectItem whose value is empty, so a row that arrives
  // without an id is dropped rather than rendered as an unselectable option.
  const contactOptions = (contacts?.data?.list ?? [])
    .map((row: any) => {
      const entity = row.entity ?? row;
      // The contact list returns a single `name` and the address in
      // `contact_information`; reading first_name/last_name/email alone
      // labelled every contact "Unnamed contact".
      const name =
        (entity?.name && entity.name !== 'N/A' ? entity.name : '') ||
        [entity?.first_name, entity?.last_name].filter(Boolean).join(' ');
      const email = resolveContactEmail(entity) || entity?.email;
      const company =
        entity?.company && entity.company !== 'N/A' ? entity.company : '';
      return {
        id: entity?.id as string | undefined,
        label: name || email || 'Unnamed contact',
        sublabel: [company, email].filter(Boolean).join(' · ') || undefined,
      };
    })
    .filter(
      (
        option
      ): option is { id: string; label: any; sublabel: string | undefined } =>
        Boolean(option.id)
    );

  const segments = segmentsRes?.data ?? [];
  const hasSegments = segments.length > 0;

  const calculateScoresMutation = useMutation({
    mutationFn: (id: string) => IntelligenceApi.calculateScores(id),
    onSuccess: (response) => {
      setScoresData(response.data);
      toast.success('Match scores calculated');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to calculate scores');
    },
  });

  return (
    <ContentLayout title="Smart Insights" section="Daily Work" icon={Sparkles}>
      <PageHero
        icon={Brain}
        eyebrow="Daily Work"
        accent="violet"
        title="Smart Insights"
        description="AI-powered prospect analysis, semantic search, and ICP scoring."
        actions={
          <div className="flex items-center gap-3">
            <span className="hidden text-[0.7rem] font-semibold uppercase tracking-[0.14em] text-muted-foreground lg:inline">
              Working contact
            </span>
            <Select value={contactId || undefined} onValueChange={setContactId}>
              <SelectTrigger className="h-11 w-[16rem] bg-background/80">
                <SelectValue
                  placeholder={
                    loadingContacts ? 'Loading contacts…' : 'Select a contact'
                  }
                />
              </SelectTrigger>
              <SelectContent>
                {contactOptions.map((option) => (
                  <SelectItem key={option.id} value={option.id}>
                    <span className="font-medium">{option.label}</span>
                    {option.sublabel && (
                      <span className="ml-2 text-muted-foreground">
                        {option.sublabel}
                      </span>
                    )}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        }
      />

      <section className="min-w-0 space-y-4">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">Capabilities</h2>
          <p className="text-[0.9rem] text-muted-foreground">
            Pick a contact above, then run any of these against it
          </p>
        </div>

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <CapabilityCard
            icon={Sparkles}
            tone="violet"
            title="Prospect Insights"
            description="Surface pain points, buying signals, and what the contact cares about."
            hint={!contactId ? 'Select a contact above to enable.' : undefined}
          >
            <ContactEnrichmentButton
              contactId={contactId}
              variant="default"
              size="default"
              showLabel
            />
          </CapabilityCard>

          <CapabilityCard
            icon={Search}
            tone="blue"
            title="Smart Search"
            description="Find contacts by describing them in plain English — no exact keywords needed."
            hint="Works across every contact, no selection needed."
          >
            <VectorSearchDialog
              trigger={
                <Button className="w-full">
                  <Search className="mr-2 h-4 w-4" />
                  Open search
                </Button>
              }
            />
          </CapabilityCard>

          <CapabilityCard
            icon={Target}
            tone="emerald"
            title="Match Score"
            description="Rate contacts against your Ideal Customer Profile."
            hint={
              !hasSegments
                ? 'Define a segment below to enable.'
                : !contactId
                  ? 'Select a contact above to enable.'
                  : undefined
            }
          >
            <Button
              className="w-full"
              disabled={
                !contactId || !hasSegments || calculateScoresMutation.isPending
              }
              onClick={() => calculateScoresMutation.mutate(contactId)}
            >
              {calculateScoresMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Target className="mr-2 h-4 w-4" />
              )}
              Calculate
            </Button>
          </CapabilityCard>
        </div>

        {scoresData && <ScoreResults data={scoresData} />}
      </section>

      <section className="min-w-0 space-y-4">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="text-xl font-semibold tracking-tight">
              Ideal Customer Profiles
            </h2>
            <p className="text-[0.9rem] text-muted-foreground">
              {hasSegments
                ? `${segments.length} segment${segments.length === 1 ? '' : 's'} the AI scores every contact against`
                : 'Define who you sell to, and the AI scores every contact against it'}
            </p>
          </div>
          {hasSegments && (
            <Button size="sm" asChild>
              <Link href="/audiences">
                <Plus className="mr-1.5 size-4" />
                New segment
              </Link>
            </Button>
          )}
        </div>

        {loadingSegments ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <Skeleton className="h-40 rounded-xl" />
            <Skeleton className="h-40 rounded-xl" />
          </div>
        ) : hasSegments ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {segments.map((segment: any) => (
              <SegmentCard key={segment.id} segment={segment} />
            ))}
          </div>
        ) : (
          <div className="flex flex-col items-center gap-4 rounded-2xl border border-dashed bg-card px-6 py-12 text-center">
            <span className="grid size-14 place-items-center rounded-2xl bg-gradient-to-br from-emerald-500 to-teal-600 text-white">
              <Layers className="size-7" />
            </span>
            <div className="max-w-md">
              <p className="text-lg font-semibold">No segments yet</p>
              <p className="mt-1.5 text-[0.9rem] leading-relaxed text-muted-foreground">
                A segment describes your ideal customer — industry, company size
                and job titles. Match Score stays disabled until there is at
                least one.
              </p>
            </div>
            <Button asChild>
              <Link href="/audiences">
                <Plus className="mr-1.5 size-4" />
                Create first segment
              </Link>
            </Button>
          </div>
        )}
      </section>
    </ContentLayout>
  );
}

const TONE = {
  violet: 'bg-gradient-to-br from-violet-500 to-indigo-600',
  blue: 'bg-gradient-to-br from-sky-500 to-blue-600',
  emerald: 'bg-gradient-to-br from-emerald-500 to-teal-600',
} as const;

function CapabilityCard({
  icon: Icon,
  tone,
  title,
  description,
  hint,
  children,
}: {
  icon: React.ComponentType<{ className?: string }>;
  tone: keyof typeof TONE;
  title: string;
  description: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="group flex h-full flex-col rounded-2xl border bg-card p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-violet-200 hover:shadow-lg">
      <div className="flex items-start gap-3">
        <span
          className={cn(
            'grid size-11 shrink-0 place-items-center rounded-xl text-white shadow-sm',
            TONE[tone]
          )}
        >
          <Icon className="size-5" />
        </span>
        <div className="min-w-0 pt-0.5">
          <h3 className="text-[1.05rem] font-semibold leading-tight tracking-tight">
            {title}
          </h3>
        </div>
      </div>

      <p className="mt-3 text-[0.9rem] leading-relaxed text-muted-foreground">
        {description}
      </p>

      {/* Pushed to the bottom so every card's button sits on the same line,
          however long the description runs. */}
      <div className="mt-auto space-y-2 pt-5">
        {children}
        {hint && (
          <p className="text-[0.8rem] leading-snug text-muted-foreground">
            {hint}
          </p>
        )}
      </div>
    </div>
  );
}

function SegmentCard({ segment }: { segment: any }) {
  const icp = segment.icp_definition ?? {};
  const rows = [
    { label: 'Industries', values: icp.ideal_industries },
    { label: 'Company size', values: icp.ideal_company_size },
    { label: 'Job titles', values: icp.ideal_titles },
  ].filter((row) => Array.isArray(row.values) && row.values.length > 0);

  return (
    <div className="flex h-full flex-col rounded-2xl border bg-card p-5 transition-shadow hover:shadow-md">
      <div className="flex items-start gap-3">
        <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 text-white">
          <Layers className="size-[1.15rem]" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate font-semibold" title={segment.name}>
            {segment.name}
          </p>
          {segment.priority != null && (
            <p className="text-[0.75rem] text-muted-foreground">
              Priority {segment.priority}
            </p>
          )}
        </div>
        {segment.is_active === false && (
          <Badge variant="secondary" className="shrink-0">
            Inactive
          </Badge>
        )}
      </div>

      {rows.length > 0 ? (
        <dl className="mt-4 space-y-2.5">
          {rows.map((row) => (
            <div key={row.label}>
              <dt className="text-[0.7rem] font-semibold uppercase tracking-[0.1em] text-muted-foreground">
                {row.label}
              </dt>
              <dd className="mt-1 flex flex-wrap gap-1.5">
                {row.values.map((value: string) => (
                  <span
                    key={value}
                    className="rounded-full border bg-muted/60 px-2.5 py-0.5 text-[0.75rem]"
                  >
                    {value}
                  </span>
                ))}
              </dd>
            </div>
          ))}
        </dl>
      ) : (
        segment.description && (
          <p className="mt-3 text-[0.85rem] leading-relaxed text-muted-foreground">
            {segment.description}
          </p>
        )
      )}
    </div>
  );
}

function ScoreResults({ data }: { data: CalculateScoresResponse }) {
  return (
    <div className="space-y-4 rounded-xl border bg-card p-5">
      <div>
        <h3 className="text-lg font-semibold tracking-tight">
          Match score results
        </h3>
        <p className="text-[0.9rem] text-muted-foreground">
          Contact evaluated against {data.segments_evaluated} segment
          {data.segments_evaluated === 1 ? '' : 's'}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <SummaryTile
          label="Segments evaluated"
          value={data.segments_evaluated}
        />
        <SummaryTile label="Segments matched" value={data.segments_matched} />
        <SummaryTile label="Priority score" value={data.priority_score} />
      </div>

      {data.scores.length > 0 && (
        <div className="space-y-3">
          <h4 className="font-semibold">Segment matches</h4>
          {data.scores.map((score, idx) => (
            <div key={idx} className="rounded-lg border p-4">
              <div className="mb-2 flex items-center justify-between gap-3">
                <p className="font-medium">{score.segmentation_name}</p>
                <Badge
                  variant={
                    score.fit_score >= 80
                      ? 'default'
                      : score.fit_score >= 60
                        ? 'secondary'
                        : 'outline'
                  }
                >
                  Score: {score.fit_score}
                </Badge>
              </div>
              {score.passes_filters && (
                <Badge variant="outline" className="mb-2">
                  Passes filters
                </Badge>
              )}
              {Object.keys(score.score_breakdown).length > 0 && (
                <div className="mt-2 grid grid-cols-2 gap-2 text-sm">
                  {Object.entries(score.score_breakdown).map(([key, value]) => (
                    <div
                      key={key}
                      className="flex justify-between rounded bg-muted p-2"
                    >
                      <span className="text-muted-foreground">{key}:</span>
                      <span className="font-medium">{value}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SummaryTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border p-4">
      <p className="text-[0.85rem] text-muted-foreground">{label}</p>
      <p className="text-2xl font-bold">{value}</p>
    </div>
  );
}
