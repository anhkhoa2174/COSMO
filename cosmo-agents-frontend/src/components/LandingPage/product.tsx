'use client';

import {
  Inbox,
  Plug,
  Sparkles,
  Split,
  Target,
  UserCheck,
  Users,
  Zap,
} from 'lucide-react';
import Image from 'next/image';
import Link from 'next/link';
import React from 'react';
import { cn } from '@/lib/utils';
import headerImage from '../../../public/landing-page/header.webp';

const WAITLIST_URL = 'https://e3tbtbnfxh7.typeform.com/to/TxJsQ5wz';

/** The four cards that float around the headline. */
const FLOATING = [
  {
    icon: Sparkles,
    title: 'AI-Drafted Replies',
    subtitle: 'Grounded in your knowledge base',
    position: 'left-0 top-4',
  },
  {
    icon: Inbox,
    title: 'Smart Inbox',
    subtitle: 'Auto-classified by intent',
    position: 'left-0 top-40',
  },
  {
    icon: Zap,
    title: '60-Second Setup',
    subtitle: 'Connect Gmail in one click',
    position: 'right-0 top-4',
  },
  {
    icon: Target,
    title: 'Daily Action Briefing',
    subtitle: 'Know who to contact and why',
    position: 'right-0 top-40',
  },
] as const;

/** Stat chips floating beside the hero screenshot. Claims reused from
 *  elsewhere on the page (key features + proof row) — nothing invented. */
const CHIPS = [
  {
    icon: Split,
    label: '9 reply intents auto-routed',
    position: 'left-[6%] -top-4',
  },
  {
    icon: UserCheck,
    label: 'Every draft human-approved',
    position: 'left-[3%] bottom-[16%]',
  },
  {
    icon: Plug,
    label: 'Apollo · LinkedIn · Gmail',
    position: 'right-[2%] top-[22%]',
  },
  {
    icon: Users,
    label: '100+ teams on the waitlist',
    position: 'right-[3%] bottom-[20%]',
  },
] as const;

const PROOF = [
  { dot: 'bg-emerald-500', label: '100+ teams on the waitlist' },
  { dot: 'bg-blue-500', label: 'Self-hosted & open architecture' },
  { dot: 'bg-violet-500', label: 'Apollo · LinkedIn · Gmail integrations' },
];

function FloatingCard({
  icon: Icon,
  title,
  subtitle,
  className,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  subtitle: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex items-center gap-3 rounded-2xl border bg-background/95 p-4 shadow-lg backdrop-blur',
        className
      )}
    >
      <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-violet-500 to-indigo-600 text-white">
        <Icon className="size-5" />
      </span>
      <span className="min-w-0">
        <span className="block font-semibold leading-tight">{title}</span>
        <span className="block text-[0.8rem] leading-tight text-muted-foreground">
          {subtitle}
        </span>
      </span>
    </div>
  );
}

function HeroChip({
  icon: Icon,
  label,
  className,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'flex items-center gap-2 rounded-xl border bg-background/90 px-3.5 py-2.5 shadow-lg backdrop-blur',
        className
      )}
    >
      <Icon className="size-4 shrink-0 text-indigo-500" />
      <span className="text-sm font-semibold">{label}</span>
    </div>
  );
}

const Product = () => (
  <div className="py-10">
    <div id="product" className="container relative mx-auto">
      {/* Below xl there is no room beside the headline, so the cards stack
          into a plain grid underneath it instead of floating. */}
      <div className="pointer-events-none absolute inset-x-0 top-0 hidden h-full xl:block">
        {FLOATING.map(({ icon, title, subtitle, position }) => (
          <FloatingCard
            key={title}
            icon={icon}
            title={title}
            subtitle={subtitle}
            className={cn('absolute w-[19rem]', position)}
          />
        ))}
      </div>

      <div className="mx-auto max-w-3xl px-4 text-center">
        <h1 className="text-4xl font-bold leading-tight tracking-tight md:text-5xl">
          Convert leads into revenue with an AI workforce
        </h1>
        <p className="mx-auto mt-5 max-w-2xl text-[1.05rem] leading-relaxed text-muted-foreground">
          COSMO unifies prospecting, AI-personalized outreach, intent-aware
          replies, and pipeline tracking — so your team focuses on closing, not
          chasing.
        </p>

        <div className="mt-8">
          <Link href={WAITLIST_URL} target="_blank" rel="noopener noreferrer">
            <button
              className="rounded-xl bg-gradient-to-r from-violet-600 to-blue-600 px-8 py-3.5 font-semibold text-white transition-shadow hover:shadow-xl"
              style={{ boxShadow: '0px 16px 20px rgba(98, 87, 243, 0.35)' }}
            >
              Join our waitlist
            </button>
          </Link>
        </div>

        <ul className="mt-7 flex flex-wrap items-center justify-center gap-x-8 gap-y-3">
          {PROOF.map(({ dot, label }) => (
            <li
              key={label}
              className="flex items-center gap-2.5 text-[0.9rem] text-muted-foreground"
            >
              <span className={cn('size-2 shrink-0 rounded-full', dot)} />
              {label}
            </li>
          ))}
        </ul>
      </div>

      <div className="mt-10 grid gap-4 px-4 sm:grid-cols-2 xl:hidden">
        {FLOATING.map(({ icon, title, subtitle }) => (
          <FloatingCard
            key={title}
            icon={icon}
            title={title}
            subtitle={subtitle}
          />
        ))}
      </div>

      <div className="relative mt-12 w-full px-4">
        <div className="overflow-hidden rounded-2xl border shadow-lg">
          <Image
            src={headerImage}
            alt="COSMO inbox with AI-classified replies"
            className="w-full scale-[1.55]"
            priority
            sizes="100vw"
          />
        </div>

        {/* Stat chips floating over the glow that flanks the screenshot —
            every claim also appears elsewhere on this page. */}
        <div className="pointer-events-none absolute inset-4 hidden lg:block">
          {CHIPS.map(({ icon, label, position }) => (
            <HeroChip
              key={label}
              icon={icon}
              label={label}
              className={cn('absolute', position)}
            />
          ))}
        </div>
      </div>
    </div>
  </div>
);

export default Product;
