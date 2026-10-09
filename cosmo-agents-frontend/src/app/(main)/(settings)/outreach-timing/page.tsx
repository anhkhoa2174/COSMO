'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, RotateCcw, ShieldAlert } from 'lucide-react';
import { useEffect, useState } from 'react';
import { toast } from 'sonner';

import { MainButton } from '@/components/buttons/main-button';
import { ContentLayout } from '@/components/nav/content-layout';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { useUser } from '@/hooks/use-user';
import { cn } from '@/lib/utils';
import OutreachSettingsApi, {
  type AutoReplyContent,
  type AutoReplyPolicy,
  type EffectiveOutreachConfig,
  type OutreachSettings,
} from '@/network/client/outreach-settings';

import {
  normaliseContents,
  ReplyContentEditor,
} from './_components/reply-content-editor';

type Field = {
  key: keyof OutreachSettings;
  label: string;
  unit: string;
  min: number;
  max: number;
  help: string;
  /**
   * Whether the follow-up scheduler actually reads this value yet.
   *
   * Five of these fields are stored and validated but not consulted by the
   * transition logic, which still advances every follow-up on the no-reply
   * window alone. Marking them is the honest option: a control that silently
   * does nothing is worse than one labelled as not yet wired, and the stored
   * value will take effect unchanged once the scheduler reads it.
   */
  applied?: boolean;
};

const GROUPS: { title: string; blurb: string; fields: Field[] }[] = [
  {
    title: 'Waiting for a reply',
    blurb:
      'How long a contact is given before COSMO treats silence as silence.',
    fields: [
      {
        key: 'no_reply_hours',
        applied: true,
        label: 'No-reply window',
        unit: 'hours',
        min: 1,
        max: 720,
        help: 'A thread with no answer after this long becomes a follow-up.',
      },
    ],
  },
  {
    title: 'Follow-up cadence',
    blurb:
      'Each follow-up is sent somewhere inside its window, not on a fixed day, so a campaign does not arrive on the same schedule for everyone. The day windows below are stored but not yet consulted: every follow-up currently advances on the no-reply window above. The values are kept, so they take effect unchanged once the scheduler reads them.',
    fields: [
      {
        key: 'follow_up1_min_days',
        applied: false,
        label: 'First follow-up, earliest',
        unit: 'days',
        min: 0,
        max: 90,
        help: '',
      },
      {
        key: 'follow_up1_max_days',
        applied: false,
        label: 'First follow-up, latest',
        unit: 'days',
        min: 0,
        max: 90,
        help: '',
      },
      {
        key: 'follow_up2_min_days',
        applied: false,
        label: 'Second follow-up, earliest',
        unit: 'days',
        min: 0,
        max: 90,
        help: '',
      },
      {
        key: 'follow_up2_max_days',
        applied: false,
        label: 'Second follow-up, latest',
        unit: 'days',
        min: 0,
        max: 90,
        help: '',
      },
      {
        key: 'max_followups',
        applied: true,
        label: 'Maximum follow-ups',
        unit: 'emails',
        min: 0,
        max: 10,
        help: 'After this many unanswered follow-ups the contact is left alone.',
      },
    ],
  },
  {
    title: 'Meetings and re-engagement',
    blurb:
      'What happens around a booked meeting, and when a cold contact is worth another try.',
    fields: [
      {
        key: 'meeting_confirm_min_days',
        applied: false,
        label: 'Confirm a meeting this far ahead',
        unit: 'days',
        min: 0,
        max: 30,
        help: '',
      },
      {
        key: 're_engage_threshold_days',
        applied: true,
        label: 'Re-engage a quiet contact after',
        unit: 'days',
        min: 1,
        max: 730,
        help: '',
      },
    ],
  },
];

const ALL_FIELDS = GROUPS.flatMap((g) => g.fields);

/**
 * Admin-only page for the outreach cadence.
 *
 * The form is driven by the *effective* config rather than by the stored
 * overrides, so every box shows the number that is really in use. A box left
 * equal to the default is not sent, which keeps the organisation tracking the
 * default if it ever changes.
 */
export default function OutreachTimingPage() {
  const { user, isLoading: userLoading } = useUser();
  const isAdmin = user?.roles?.some((r) => r.name === 'admin') ?? false;

  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ['outreach-settings'],
    queryFn: () => OutreachSettingsApi.get(),
    enabled: isAdmin,
  });

  const [values, setValues] = useState<Record<string, string>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [autoReply, setAutoReply] = useState<{
    enabled: boolean;
    intents: string[];
    minConfidence: number;
    dailyCap: string;
    contents: Record<string, AutoReplyContent>;
  }>({
    enabled: false,
    intents: [],
    minConfidence: 0.9,
    dailyCap: '20',
    contents: {},
  });

  useEffect(() => {
    if (!data) return;
    const next: Record<string, string> = {};
    for (const f of ALL_FIELDS) {
      next[f.key] = String(
        data.effective[f.key as keyof EffectiveOutreachConfig]
      );
    }
    setValues(next);
    setErrors({});
    setAutoReply({
      enabled: data.auto_reply.enabled,
      intents: data.auto_reply.intents,
      minConfidence: data.auto_reply.min_confidence,
      dailyCap: String(data.auto_reply.daily_cap),
      contents: data.auto_reply.contents ?? {},
    });
  }, [data]);

  const { mutate, isPending } = useMutation({
    mutationFn: (payload: OutreachSettings) =>
      OutreachSettingsApi.update(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['outreach-settings'] });
      toast.success('Timing saved', {
        description: 'Campaigns already running pick this up within a minute.',
      });
    },
    onError: (e: any) =>
      toast.error('Could not save timing', {
        description: e?.message ?? 'Please check the values and try again.',
      }),
  });

  if (userLoading) {
    return (
      <ContentLayout title="Outreach Timing">
        <Skeleton className="h-40" />
      </ContentLayout>
    );
  }

  if (!isAdmin) {
    return (
      <ContentLayout title="Outreach Timing">
        <div className="flex max-w-lg items-start gap-3 rounded-lg border bg-muted/40 p-4">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          <div className="text-sm">
            <p className="font-medium">
              Only an admin can change outreach timing
            </p>
            <p className="text-muted-foreground">
              These settings apply to every campaign in the organisation. Ask an
              admin on your team if a follow-up window needs to move.
            </p>
          </div>
        </div>
      </ContentLayout>
    );
  }

  /** Local validation mirrors the server's bounds so a mistake is caught in the form. */
  const validate = (v: Record<string, string>, only?: string[]) => {
    const found: Record<string, string> = {};
    const inScope = (key: string) => !only || only.includes(key);
    for (const f of ALL_FIELDS) {
      if (!inScope(f.key)) continue;
      const n = Number(v[f.key]);
      if (v[f.key] === '' || Number.isNaN(n) || !Number.isInteger(n)) {
        found[f.key] = 'Enter a whole number';
      } else if (n < f.min || n > f.max) {
        found[f.key] = `Must be between ${f.min} and ${f.max}`;
      }
    }
    const pair = (
      min: keyof OutreachSettings,
      max: keyof OutreachSettings,
      name: string
    ) => {
      // A range is only checked when this save actually writes one of its
      // ends; otherwise the stored pair is left exactly as it was.
      if (!inScope(min as string) && !inScope(max as string)) return;
      if (found[min] || found[max]) return;
      if (Number(v[min]) > Number(v[max])) {
        found[max] = `${name} cannot end before it starts`;
      }
    };
    pair('follow_up1_min_days', 'follow_up1_max_days', 'First follow-up');
    pair('follow_up2_min_days', 'follow_up2_max_days', 'Second follow-up');
    return found;
  };

  const onSave = () => {
    // Only what differs from the default is sent: an untouched field should
    // keep following the default rather than being frozen at today's value.
    const payload: OutreachSettings = {};
    for (const f of ALL_FIELDS) {
      const n = Number(values[f.key]);
      if (data && n !== data.defaults[f.key as keyof EffectiveOutreachConfig]) {
        payload[f.key] = n;
      }
    }

    // Validate what is being written, and nothing else. This used to check
    // every follow-up field on the page, so an administrator who changed only
    // the auto-reply policy could be blocked by a follow-up window they had not
    // touched, reported against a field far below the button they pressed. A
    // value that is not part of this save has no bearing on whether it can
    // proceed.
    const found = validate(values, Object.keys(payload));
    setErrors(found);
    if (Object.keys(found).length > 0) {
      const firstBad = ALL_FIELDS.find((f) => found[f.key])?.key;
      toast.error('Nothing was saved', {
        description: 'Fix the highlighted follow-up timing value first.',
      });
      if (firstBad) {
        document
          .getElementById(firstBad)
          ?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }
      return;
    }

    // The policy is always sent, including when it is off. Omitting it would
    // mean "leave whatever is stored", so an admin could not turn auto-reply
    // back off once it had been enabled.
    payload.auto_reply = {
      enabled: autoReply.enabled,
      intents: autoReply.intents,
      min_confidence: autoReply.minConfidence,
      daily_cap: Number(autoReply.dailyCap) || 1,
      // Content for intents that are not ticked is kept, so unticking an
      // intent does not throw away the message written for it.
      contents: normaliseContents(autoReply.contents),
    };

    mutate(payload);
  };

  const resetToDefaults = () => {
    if (!data) return;
    const next: Record<string, string> = {};
    for (const f of ALL_FIELDS) {
      next[f.key] = String(
        data.defaults[f.key as keyof EffectiveOutreachConfig]
      );
    }
    setValues(next);
    setErrors({});
  };

  const timingDirty =
    !!data &&
    ALL_FIELDS.some(
      (f) =>
        values[f.key] !==
        String(data.effective[f.key as keyof EffectiveOutreachConfig])
    );

  const autoReplyDirty =
    !!data &&
    (autoReply.enabled !== data.auto_reply.enabled ||
      autoReply.minConfidence !== data.auto_reply.min_confidence ||
      autoReply.dailyCap !== String(data.auto_reply.daily_cap) ||
      autoReply.intents.slice().sort().join(',') !==
        data.auto_reply.intents.slice().sort().join(',') ||
      JSON.stringify(normaliseContents(autoReply.contents)) !==
        JSON.stringify(normaliseContents(data.auto_reply.contents ?? {})));

  const dirty = timingDirty || autoReplyDirty;

  const rightSection = dirty ? (
    <MainButton text="Save changes" loading={isPending} onClick={onSave} />
  ) : undefined;

  return (
    <ContentLayout title="Outreach Timing" rightSection={rightSection}>
      {isLoading || !data ? (
        <div className="space-y-4">
          <Skeleton className="h-10" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
      ) : (
        <div className="space-y-8">
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-muted/40 px-4 py-3">
            <p className="text-sm">
              <span className="font-medium">Currently in force: </span>
              <span className="text-muted-foreground">
                COSMO {data.summary}.
              </span>
            </p>
            <Button
              variant="ghost"
              size="sm"
              className="gap-1.5"
              onClick={resetToDefaults}
            >
              <RotateCcw className="size-3.5" />
              Reset to defaults
            </Button>
          </div>

          <div className="grid grid-cols-1 gap-4 border-t pt-8 md:grid-cols-4">
            <div className="md:col-span-1">
              <p className="text-lg font-semibold">Automatic replies</p>
              <p className="text-sm text-muted-foreground">
                By default COSMO writes a reply and holds it for someone to
                read. You can hand specific kinds of reply over to it entirely.
              </p>
            </div>

            <div className="space-y-5 md:col-span-3">
              <label className="flex cursor-pointer items-start gap-3">
                <input
                  type="checkbox"
                  checked={autoReply.enabled}
                  onChange={(e) =>
                    setAutoReply((a) => ({ ...a, enabled: e.target.checked }))
                  }
                  className="mt-0.5 size-4 accent-primary"
                />
                <span className="text-sm">
                  <span className="font-medium">
                    Send some replies without review
                  </span>
                  <span className="block text-muted-foreground">
                    Nothing is sent automatically until you also choose which
                    kinds of reply below.
                  </span>
                </span>
              </label>

              <div
                className={cn(
                  'space-y-5',
                  !autoReply.enabled && 'pointer-events-none opacity-50'
                )}
              >
                <div>
                  <p className="mb-1.5 text-sm font-medium">
                    Reply kinds COSMO may send on its own
                  </p>
                  <div className="flex flex-wrap gap-1.5">
                    {data.auto_reply_selectable_intents.map((intent) => {
                      const on = autoReply.intents.includes(intent);
                      return (
                        <button
                          key={intent}
                          type="button"
                          onClick={() =>
                            setAutoReply((a) => ({
                              ...a,
                              intents: on
                                ? a.intents.filter((i) => i !== intent)
                                : [...a.intents, intent],
                            }))
                          }
                          className={cn(
                            'rounded-md border px-2 py-1 text-xs transition-colors',
                            on
                              ? 'border-primary bg-primary/10 text-foreground'
                              : 'text-muted-foreground hover:bg-accent'
                          )}
                        >
                          {intent}
                        </button>
                      );
                    })}
                  </div>
                  <p className="mt-2 flex items-start gap-2 text-xs text-muted-foreground">
                    <ShieldAlert className="mt-0.5 size-3.5 shrink-0" />
                    Declines, opt-out requests and replies COSMO could not
                    classify are never sent automatically, so they are not
                    listed here.
                  </p>
                </div>

                {autoReply.intents.length > 0 && (
                  <div className="space-y-2">
                    <div>
                      <p className="text-sm font-medium">
                        What each reply says
                      </p>
                      <p className="text-xs text-muted-foreground">
                        Let the AI write a personal reply, optionally following
                        your instructions, or send a fixed message you write
                        once. This changes what is sent, never whether: the
                        confidence and daily limits below still apply.
                      </p>
                    </div>
                    {data.auto_reply_selectable_intents
                      .filter((intent) => autoReply.intents.includes(intent))
                      .map((intent) => (
                        <ReplyContentEditor
                          key={intent}
                          intent={intent}
                          content={autoReply.contents[intent]}
                          mergeTags={data.auto_reply_merge_tags ?? []}
                          onChange={(next) =>
                            setAutoReply((a) => ({
                              ...a,
                              contents: { ...a.contents, [intent]: next },
                            }))
                          }
                        />
                      ))}
                  </div>
                )}

                <div className="flex flex-wrap items-start gap-6">
                  <div>
                    <label
                      htmlFor="min_confidence"
                      className="text-sm font-medium"
                    >
                      Only when COSMO is at least this sure
                    </label>
                    <p className="text-xs text-muted-foreground">
                      Anything less confident is held for review.
                    </p>
                    <div className="mt-1.5 flex items-center gap-2">
                      <input
                        id="min_confidence"
                        type="range"
                        min={50}
                        max={100}
                        step={5}
                        value={Math.round(autoReply.minConfidence * 100)}
                        onChange={(e) =>
                          setAutoReply((a) => ({
                            ...a,
                            minConfidence: Number(e.target.value) / 100,
                          }))
                        }
                        className="w-40 accent-primary"
                      />
                      <span className="w-12 text-sm tabular-nums">
                        {Math.round(autoReply.minConfidence * 100)}%
                      </span>
                    </div>
                  </div>

                  <div>
                    <label htmlFor="daily_cap" className="text-sm font-medium">
                      At most per day
                    </label>
                    <p className="text-xs text-muted-foreground">
                      A ceiling on how much a mistake can cost.
                    </p>
                    <Input
                      id="daily_cap"
                      type="number"
                      min={1}
                      max={500}
                      value={autoReply.dailyCap}
                      onChange={(e) =>
                        setAutoReply((a) => ({
                          ...a,
                          dailyCap: e.target.value,
                        }))
                      }
                      className="mt-1.5 h-9 w-24"
                    />
                  </div>
                </div>
              </div>

              <p className="rounded-lg border bg-muted/40 px-3 py-2 text-sm">
                <span className="font-medium">Right now: </span>
                <span className="text-muted-foreground">
                  COSMO {data.auto_reply.summary}.
                </span>
              </p>
            </div>
          </div>

          {GROUPS.map((group) => (
            <div
              key={group.title}
              className="grid grid-cols-1 gap-4 md:grid-cols-4"
            >
              <div className="md:col-span-1">
                <p className="text-lg font-semibold">{group.title}</p>
                <p className="text-sm text-muted-foreground">{group.blurb}</p>
              </div>

              <div className="space-y-4 md:col-span-3">
                {group.fields.map((f) => {
                  const def =
                    data.defaults[f.key as keyof EffectiveOutreachConfig];
                  const changed = Number(values[f.key]) !== def;
                  return (
                    <div
                      key={f.key}
                      className="flex flex-wrap items-start gap-3"
                    >
                      <div className="min-w-[16rem] flex-1">
                        <label htmlFor={f.key} className="text-sm font-medium">
                          {f.label}
                        </label>
                        {f.help && (
                          <p className="text-xs text-muted-foreground">
                            {f.help}
                          </p>
                        )}
                        {f.applied === false && (
                          <p className="mt-0.5 inline-flex items-center gap-1 rounded bg-amber-50 px-1.5 py-0.5 text-[0.7rem] text-amber-800">
                            <AlertTriangle className="size-3 shrink-0" />
                            Saved, but the scheduler does not read this yet
                          </p>
                        )}
                        {errors[f.key] && (
                          <p className="text-xs text-destructive">
                            {errors[f.key]}
                          </p>
                        )}
                      </div>

                      <div className="flex items-center gap-2">
                        <Input
                          id={f.key}
                          type="number"
                          inputMode="numeric"
                          min={f.min}
                          max={f.max}
                          value={values[f.key] ?? ''}
                          onChange={(e) =>
                            setValues((v) => ({
                              ...v,
                              [f.key]: e.target.value,
                            }))
                          }
                          className={cn(
                            'h-9 w-24',
                            errors[f.key] && 'border-destructive'
                          )}
                        />
                        <span className="w-14 text-sm text-muted-foreground">
                          {f.unit}
                        </span>
                        <span
                          className={cn(
                            'w-24 text-xs',
                            changed ? 'text-foreground' : 'text-transparent'
                          )}
                        >
                          default {def}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}
    </ContentLayout>
  );
}
