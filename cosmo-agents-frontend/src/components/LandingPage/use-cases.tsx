'use client';

import {
  ArrowUpRight,
  BookOpen,
  Cpu,
  Cross,
  Home,
  Split,
  UserCheck,
} from 'lucide-react';
import Image, { StaticImageData } from 'next/image';
import { IconLeaf } from '@/assets/icons';
import { useState } from 'react';
import classes from './UseCase.module.css';
import techImage from '../../../public/landing-page/tech.webp';
import educationImage from '../../../public/landing-page/education.webp';
import healthcareImage from '../../../public/landing-page/healthcare.webp';
import realEstateImage from '../../../public/landing-page/real_estate.webp';

/** Mini-cards inside the purple band. Claims reused from the key-features
 *  section — nothing invented. */
const BAND_CARDS = [
  {
    icon: Split,
    title: '9 reply intents',
    desc: 'Every inbound reply is classified and auto-routed',
  },
  {
    icon: UserCheck,
    title: 'Human-approved',
    desc: 'A human check before anything leaves your inbox',
  },
  {
    icon: BookOpen,
    title: 'Your knowledge base',
    desc: 'Drafts grounded in your own docs and pricing',
  },
] as const;

export function UseCases() {
  const [activeTab, setActiveTab] = useState('tech');

  const tabs = [
    {
      id: 'tech',
      label: 'Tech',
      icon: <Cpu />,
    },
    {
      id: 'education',
      label: 'Education',
      icon: <BookOpen />,
    },
    {
      id: 'healthcare',
      label: 'Healthcare',
      icon: <Cross />,
    },
    {
      id: 'real-estate',
      label: 'Real Estate',
      icon: <Home />,
    },
  ];

  return (
    <div className="py-8">
      <div id="use-cases" className="container mx-auto">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          {/* Badge */}
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-700">
            <span className="text-xs font-bold uppercase">Use cases</span>
          </div>

          {/* Title */}
          <h2 className="text-2xl font-bold">
            <span className="bg-gradient-to-r from-indigo-500 to-purple-500 bg-clip-text text-transparent">
              Automate
            </span>{' '}
            your outreach with an AI workforce across your{' '}
            <span className="text-indigo-500">entire pipeline</span>
          </h2>

          {/* Description */}
        </div>

        {/* Bottom section */}
        <div className="mt-12">
          <div className="mb-4 flex items-center justify-center gap-4">
            {tabs.map(({ id, label, icon }) => (
              <button
                type="button"
                key={id}
                className={`${classes.tab} text-muted-foreground`}
                onClick={() => setActiveTab(id)}
                data-active={activeTab === id}
              >
                <span className="sm:hidden">{icon}</span>
                <span className="hidden font-semibold sm:inline-block">
                  {label}
                </span>
              </button>
            ))}
          </div>
          {activeTab === 'tech' && (
            <UseCasePanel
              title="Technology"
              items={[
                {
                  lead: 'Inbound on autopilot',
                  desc: 'Scale your inbound with a team of AI agents',
                },
                {
                  lead: 'Research and enrich',
                  desc: 'Automatically research, enrich, and prioritize inbound leads',
                },
                {
                  lead: 'Personalized outreach',
                  desc: 'Hyper-personalized outreach and replies',
                },
                {
                  lead: 'Meetings booked',
                  desc: 'Generate meetings and sales opportunities from marketing and product leads',
                },
              ]}
              image={techImage}
              alt="tech.png"
            />
          )}
          {activeTab === 'education' && (
            <UseCasePanel
              title="Grow your student enrollment with AI agents"
              items={[
                {
                  lead: 'On-brand content',
                  desc: "Content created from your school's guidelines and knowledge",
                },
                {
                  lead: 'Reach students and parents',
                  desc: 'An AI agent helps reach out to students and parents',
                },
                {
                  lead: 'Campus tours',
                  desc: 'Book tours and automatically follow up with more information',
                },
                {
                  lead: 'Enrollment support',
                  desc: 'Qualify, follow up, and schedule appointments for your team',
                },
              ]}
              image={educationImage}
              alt="education.png"
            />
          )}
          {activeTab === 'healthcare' && (
            <UseCasePanel
              title="Streamline patient care with AI assistance"
              subtitle="Enhance patient engagement and optimize healthcare operations"
              items={[
                {
                  lead: 'Scheduling and reminders',
                  desc: 'Automate appointment scheduling and reminders',
                },
                {
                  lead: 'Follow-up care',
                  desc: 'Provide personalized follow-up care instructions',
                },
                {
                  lead: 'Insurance and billing',
                  desc: 'Assist with insurance inquiries and billing processes',
                },
              ]}
              image={healthcareImage}
              alt="healthcare.png"
            />
          )}
          {activeTab === 'real-estate' && (
            <UseCasePanel
              title="Streamline real estate with AI assistance"
              items={[
                {
                  lead: 'Client interactions',
                  desc: 'Let agents handle client interactions and streamline processes',
                },
                {
                  lead: 'Property viewings',
                  desc: 'Schedule and manage property viewings efficiently',
                },
                {
                  lead: 'Automatic follow-ups',
                  desc: 'Follow up with prospects to keep every conversation moving',
                },
              ]}
              image={realEstateImage}
              alt="real_estate.png"
            />
          )}
        </div>
      </div>
      <div className="bg-[url(/landing-page/use_case.webp)] bg-cover bg-center">
        <div className="container mx-auto flex flex-col items-center justify-center gap-8 px-4 py-14">
          <div className="flex max-w-xl flex-col items-center gap-4">
            <h2 className="text-2xl font-bold">
              Agents working on your leads 24/7
            </h2>
            <p className="text-center text-muted-foreground">
              Watch as your <span className="text-indigo-500">Cosmo</span>{' '}
              generate personalized content, segment leads, nurture
              conversations, and{' '}
              <span className="underline decoration-green-500 underline-offset-2">
                drive revenue automatically
              </span>
              .
            </p>
          </div>
          <div className="grid w-full max-w-4xl gap-4 sm:grid-cols-3">
            {BAND_CARDS.map(({ icon: Icon, title, desc }) => (
              <div
                key={title}
                className="flex items-start gap-3 rounded-xl border border-white/50 bg-white/85 p-4 shadow-md backdrop-blur"
              >
                <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 text-white">
                  <Icon className="size-4" />
                </span>
                <span className="min-w-0">
                  <span className="block font-semibold leading-tight text-zinc-900">
                    {title}
                  </span>
                  <span className="mt-1 block text-sm leading-snug text-zinc-600">
                    {desc}
                  </span>
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function UseCasePanel({
  title,
  subtitle,
  items,
  image,
  alt,
}: {
  title: string;
  subtitle?: string;
  items: { lead: string; desc: string }[];
  image: StaticImageData;
  alt: string;
}) {
  return (
    <div className="grid grid-cols-1 items-center gap-8 md:grid-cols-2">
      <div className="flex flex-col justify-center gap-5">
        <div className="space-y-2">
          <h3 className="text-2xl font-bold">{title}</h3>
          {subtitle && <p className="font-semibold">{subtitle}</p>}
        </div>
        <ul className="space-y-4">
          {items.map(({ lead, desc }) => (
            <li key={lead} className="flex items-start gap-3">
              <IconLeaf className="mt-0.5 flex-shrink-0" />
              <div>
                <p className="font-semibold leading-tight">{lead}</p>
                <p className="mt-0.5 text-sm text-gray-500">{desc}</p>
              </div>
            </li>
          ))}
        </ul>
        <button className="flex w-fit items-center rounded-lg border border-indigo-500 px-4 py-2 text-black">
          Learn more
          <ArrowUpRight className="ml-2 text-indigo-500" />
        </button>
      </div>
      <Image
        src={image}
        alt={alt}
        className="h-auto w-full rounded-xl"
        sizes="(min-width: 768px) 50vw, 100vw"
      />
    </div>
  );
}
