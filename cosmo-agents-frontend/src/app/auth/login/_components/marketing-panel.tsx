import { Bot, BrainCircuit, Mail, Sparkles, Star, Target } from 'lucide-react';
import React from 'react';

const FEATURES = [
  { icon: Bot, label: 'Autonomous AI Agents' },
  { icon: Mail, label: 'Personalized Outreach' },
  { icon: BrainCircuit, label: 'Knowledge-Grounded' },
  { icon: Target, label: 'Intent-Driven Routing' },
];

const STATS = [
  { value: '60+', label: 'AI tools' },
  { value: '3-5x', label: 'More meetings' },
  { value: '24/7', label: 'Autonomous' },
];

/** The right half of the sign-in split screen. */
export function MarketingPanel() {
  return (
    <div className="relative hidden flex-col justify-between gap-10 overflow-hidden bg-gradient-to-br from-violet-500 via-purple-500 to-blue-600 p-10 text-white lg:flex">
      <div className="space-y-6">
        <span className="inline-flex items-center gap-2 rounded-full bg-white/15 px-3.5 py-1.5 text-[0.8rem] font-medium backdrop-blur">
          <Sparkles className="size-3.5" />
          Trusted by 100+ teams
        </span>

        <h2 className="max-w-xl text-4xl font-bold leading-tight tracking-tight xl:text-5xl">
          Convert leads into revenue
          <br />
          with an <span className="text-sky-200">AI workforce</span>
        </h2>

        <p className="max-w-lg text-[1.05rem] leading-relaxed text-white/85">
          Personalized outreach, intent-classified replies, and
          knowledge-grounded drafts — all on autopilot.
        </p>

        <div className="grid max-w-2xl gap-3 sm:grid-cols-2">
          {FEATURES.map(({ icon: Icon, label }) => (
            <div
              key={label}
              className="flex items-center gap-3 rounded-xl border border-white/20 bg-white/10 px-4 py-3.5 backdrop-blur"
            >
              <Icon className="size-[1.15rem] shrink-0" />
              <span className="text-[0.95rem] font-medium">{label}</span>
            </div>
          ))}
        </div>
      </div>

      <figure className="max-w-2xl space-y-5 rounded-2xl border border-white/20 bg-white/10 p-6 backdrop-blur">
        <div className="flex gap-1">
          {Array.from({ length: 5 }).map((_, i) => (
            <Star key={i} className="size-4 fill-amber-300 text-amber-300" />
          ))}
        </div>
        <blockquote className="border-l-2 border-white/40 pl-4 text-[1.05rem] font-medium leading-relaxed">
          “Cosmo simplifies our sales efforts. The lead segmentation features
          have dramatically improved our conversion rates and efficiency.”
        </blockquote>
        <figcaption className="flex items-center gap-3">
          <span className="grid size-10 shrink-0 place-items-center rounded-full bg-sky-200/90 text-[0.85rem] font-bold text-blue-900">
            PN
          </span>
          <span>
            <span className="block font-semibold">Pemi Nguyen</span>
            <span className="block text-[0.85rem] text-white/75">
              CEO at Ike Education
            </span>
          </span>
        </figcaption>
      </figure>

      <dl className="grid max-w-2xl grid-cols-3 gap-6 border-t border-white/20 pt-6">
        {STATS.map(({ value, label }) => (
          <div key={label}>
            <dt className="text-3xl font-bold tracking-tight">{value}</dt>
            <dd className="text-[0.85rem] text-white/75">{label}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
