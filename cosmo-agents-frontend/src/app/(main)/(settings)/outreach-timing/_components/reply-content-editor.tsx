'use client';

import { useRef } from 'react';

import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';
import type { AutoReplyContent } from '@/network/client/outreach-settings';

/** Limits mirror the server's, so the form stops where the save would fail. */
const MAX_SUBJECT = 200;
const MAX_BODY = 5000;
const MAX_GUIDANCE = 1000;

/** A sample contact for the preview. It is never sent anywhere. */
const SAMPLE: Record<string, string> = {
  first_name: 'Rachel',
  name: 'Rachel Kim',
  company: 'Northwind',
  job_title: 'Head of Operations',
  sender_name: 'Alex',
};

/**
 * Fills merge fields the way the server does: a field with no value is removed
 * together with the space before it, so the preview never promises a greeting
 * the prospect will not get.
 */
export function renderMergeFields(
  text: string,
  values: Record<string, string>
): string {
  return text.replace(
    /( ?)\{\{\s*([A-Za-z_]+)\s*\}\}/g,
    (_match, space: string, tag: string) => {
      const value = values[tag.toLowerCase()];
      return value ? space + value : '';
    }
  );
}

/**
 * Drops what would not change behaviour before saving or comparing: AI mode
 * with no guidance is the standard draft, and the fields of the mode not in
 * use are not stored. A fixed reply with an empty message is kept, so the
 * server can say it needs one rather than the reply silently reverting.
 */
export function normaliseContents(
  contents: Record<string, AutoReplyContent>
): Record<string, AutoReplyContent> {
  const out: Record<string, AutoReplyContent> = {};
  for (const intent of Object.keys(contents).sort()) {
    const c = contents[intent];
    if (c.mode === 'template') {
      const subject = c.subject?.trim();
      out[intent] = {
        mode: 'template',
        ...(subject ? { subject } : {}),
        body: c.body ?? '',
      };
    } else if (c.guidance?.trim()) {
      out[intent] = { mode: 'ai', guidance: c.guidance.trim() };
    }
  }
  return out;
}

type Props = {
  intent: string;
  content: AutoReplyContent | undefined;
  mergeTags: string[];
  onChange: (next: AutoReplyContent) => void;
};

/**
 * Edits what one intent's automatic reply says: either a fixed message, or
 * guidance the AI follows when it writes the draft.
 */
export function ReplyContentEditor({
  intent,
  content,
  mergeTags,
  onChange,
}: Props) {
  const value: AutoReplyContent = content ?? { mode: 'ai', guidance: '' };
  const subjectRef = useRef<HTMLInputElement>(null);
  const bodyRef = useRef<HTMLTextAreaElement>(null);
  // Merge fields go wherever the admin was last typing.
  const lastField = useRef<'subject' | 'body'>('body');

  const update = (patch: Partial<AutoReplyContent>) =>
    onChange({ ...value, ...patch });

  const insertTag = (tag: string) => {
    const field = lastField.current;
    const el = field === 'subject' ? subjectRef.current : bodyRef.current;
    const current = (field === 'subject' ? value.subject : value.body) ?? '';
    const token = `{{${tag}}}`;
    const start = el?.selectionStart ?? current.length;
    const end = el?.selectionEnd ?? current.length;
    update({
      [field]: current.slice(0, start) + token + current.slice(end),
    });
    requestAnimationFrame(() => {
      el?.focus();
      el?.setSelectionRange(start + token.length, start + token.length);
    });
  };

  const modeButton = (mode: AutoReplyContent['mode'], label: string) => (
    <button
      type="button"
      onClick={() => update({ mode })}
      className={cn(
        'rounded-md border px-2 py-1 text-xs transition-colors',
        value.mode === mode
          ? 'border-primary bg-primary/10 text-foreground'
          : 'text-muted-foreground hover:bg-accent'
      )}
    >
      {label}
    </button>
  );

  return (
    <div className="space-y-3 rounded-lg border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm font-medium">{intent}</p>
        <div className="flex gap-1.5">
          {modeButton('ai', 'AI writes it')}
          {modeButton('template', 'Fixed message')}
        </div>
      </div>

      {value.mode === 'ai' ? (
        <div className="space-y-1.5">
          <Textarea
            value={value.guidance ?? ''}
            maxLength={MAX_GUIDANCE}
            rows={3}
            placeholder="e.g. Always mention the 14-day free trial and offer the calendar link. Keep it under 120 words."
            onChange={(e) => update({ guidance: e.target.value })}
          />
          <p className="text-xs text-muted-foreground">
            Added to the instructions the AI follows when it drafts this kind of
            reply. Leave it empty to use the standard draft. Only replies COSMO
            drafts with AI — by default Interested, Request for pricing and
            Request for information — can be sent this way; other kinds need a
            fixed message.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          <Input
            ref={subjectRef}
            value={value.subject ?? ''}
            maxLength={MAX_SUBJECT}
            placeholder="Subject — leave empty to reply as “Re: their subject”"
            onFocus={() => (lastField.current = 'subject')}
            onChange={(e) => update({ subject: e.target.value })}
            className="h-9"
          />
          <Textarea
            ref={bodyRef}
            value={value.body ?? ''}
            maxLength={MAX_BODY}
            rows={5}
            placeholder={
              'Hi {{first_name}},\n\nThanks for letting us know. We will pick this up when you are back.'
            }
            onFocus={() => (lastField.current = 'body')}
            onChange={(e) => update({ body: e.target.value })}
          />
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-xs text-muted-foreground">Insert:</span>
            {mergeTags.map((tag) => (
              <button
                key={tag}
                type="button"
                onClick={() => insertTag(tag)}
                className="rounded border px-1.5 py-0.5 font-mono text-[0.7rem] text-muted-foreground hover:bg-accent"
              >
                {`{{${tag}}}`}
              </button>
            ))}
          </div>

          <div className="rounded-md bg-muted/40 px-3 py-2 text-sm">
            <p className="mb-1 text-xs text-muted-foreground">
              Preview with a sample contact. A field the contact does not have
              is left out.
            </p>
            <p className="font-medium">
              {value.subject?.trim()
                ? renderMergeFields(value.subject, SAMPLE)
                : 'Re: <their subject>'}
            </p>
            <p className="whitespace-pre-wrap text-muted-foreground">
              {value.body?.trim()
                ? renderMergeFields(value.body, SAMPLE)
                : 'Write the message to send.'}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
