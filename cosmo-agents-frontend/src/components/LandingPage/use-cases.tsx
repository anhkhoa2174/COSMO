'use client';

import { ArrowUpRight, BookOpen, Cpu, Cross, Home } from 'lucide-react';
import { IconLeaf } from '@/assets/icons';
import { useState } from 'react';
import classes from './UseCase.module.css';

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
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-500">
            <span className="text-xs font-bold uppercase">Use cases</span>
          </div>

          {/* Title */}
          <h2 className="text-2xl font-bold">
            <span className="bg-gradient-to-r from-indigo-500 to-purple-500 bg-clip-text text-transparent">
              Automate
            </span>{' '}
            customer conversations to your AI workforce across your{' '}
            <span className="text-indigo-500">entire business</span>
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
          {activeTab === 'tech' && <Tech />}
          {activeTab === 'education' && <Education />}
          {activeTab === 'healthcare' && <Healthcare />}
          {activeTab === 'real-estate' && <RealEstate />}
        </div>
      </div>
      <div className="bg-[url(/landing-page/use_case.png)] bg-cover bg-center">
        <div className="mx-auto flex max-w-xl flex-col items-center justify-center gap-4 p-12">
          <h2 className="text-2xl font-bold">
            Agents working on your leads 24/7
          </h2>
          <p className="text-center text-muted-foreground">
            Watch as your <span className="text-indigo-500">Cosmo Agents</span>{' '}
            generate personalized content, segment leads, nurture conversations,
            and{' '}
            <span className="underline decoration-green-500 underline-offset-2">
              drive revenue automatically
            </span>
            .
          </p>
        </div>
      </div>
    </div>
  );
}

const Tech = () => {
  const items = [
    'Scale your inbound on auto-pilot with a team of AI Agents',
    'Automatically research, enrich, and prioritize inbound leads',
    'Hyper-personalized outreach and replies',
    'Generate meetings and sales opportunities from marketing and product leads',
  ];

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">Technology</h2>
        <p className="font-semibold">
          Scale your inbound on auto-pilot with a team of AI Agents
        </p>
        {items.map((item, index) => (
          <div
            key={`tech-item-${index}`}
            className="flex items-start space-x-2"
          >
            <IconLeaf className="flex-shrink-0" />
            <p className="text-gray-500">{item}</p>
          </div>
        ))}
        <button className="flex items-center rounded-lg border border-indigo-500 px-4 py-2 text-black">
          Learn more
          <ArrowUpRight className="ml-2 text-indigo-500" />
        </button>
      </div>
      <img src="/landing-page/tech.png" alt="tech.png" />
    </div>
  );
};

const Education = () => {
  const items = [
    "Automatically study and create content based on your school's guidelines and knowledge",
    'AI agent helps reaching out to students and parents',
    'Book campus tours for students/parents and automatically follow up with more information',
    'Helps enrollment teams qualify customers, follow up, and schedule appointments',
  ];

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">
          Grow your student enrollment with AI agents
        </h2>
        {items.map((item, index) => (
          <div
            key={`education-item-${index}`}
            className="flex items-start space-x-2"
          >
            <IconLeaf className="flex-shrink-0" />
            <p className="text-gray-500">{item}</p>
          </div>
        ))}
        <button className="flex items-center rounded-lg border border-indigo-500 px-4 py-2 text-black">
          Learn more
          <ArrowUpRight className="ml-2 text-indigo-500" />
        </button>
      </div>
      <img src="/landing-page/education.png" alt="education.png" />
    </div>
  );
};

const Healthcare = () => {
  const items = [
    'Automate appointment scheduling and reminders',
    'Provide personalized follow-up care instructions',
    'Assist with insurance inquiries and billing processes',
  ];

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">
          Streamline patient care with AI assistance
        </h2>
        <p className="font-semibold">
          Enhance patient engagement and optimize healthcare operations
        </p>
        {items.map((item, index) => (
          <div
            key={`healthcare-item-${index}`}
            className="flex items-start space-x-2"
          >
            <IconLeaf className="flex-shrink-0" />
            <p className="text-gray-500">{item}</p>
          </div>
        ))}
        <button className="flex items-center rounded-lg border border-indigo-500 px-4 py-2 text-black">
          Learn more
          <ArrowUpRight className="ml-2 text-indigo-500" />
        </button>
      </div>
      <img src="/landing-page/healthcare.png" alt="healthcare.png" />
    </div>
  );
};

const RealEstate = () => {
  const items = [
    'Let agents handle client interactions and streamline real estate processes',
    'Schedule and manage property viewings efficiently',
    'Schedule and manage property viewings efficiently',
  ];

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">
          Streamline real estate with AI assistance
        </h2>
        <p className="font-semibold">
          Let agents handle client interactions and streamline real estate
          processes
        </p>
        {items.map((item, index) => (
          <div
            key={`realestate-item-${index}`}
            className="flex items-start space-x-2"
          >
            <IconLeaf className="flex-shrink-0" />
            <p className="text-gray-500">{item}</p>
          </div>
        ))}
        <button className="flex items-center rounded-lg border border-indigo-500 px-4 py-2 text-black">
          Learn more
          <ArrowUpRight className="ml-2 text-indigo-500" />
        </button>
      </div>
      <img src="/landing-page/real_estate.png" alt="real_estate.png" />
    </div>
  );
};
