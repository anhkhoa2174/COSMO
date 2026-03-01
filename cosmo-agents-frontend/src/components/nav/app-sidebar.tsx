'use client';

import { NavMain } from '@/components/nav/nav-main';
import { AudioWaveform, Command, Flag, GalleryVerticalEnd, Settings, Users } from 'lucide-react';
import * as React from 'react';
// import { NavProjects } from '@/components/nav/nav-projects';
import { NavUser } from '@/components/nav/nav-user';
import { TeamSwitcher } from '@/components/nav/team-switcher';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar';

// This is sample data.
const data = {
  user: {
    name: 'shadcn',
    email: 'm@example.com',
    avatar: '/favicon.svg',
  },
  teams: [
    {
      name: 'Acme Inc',
      logo: GalleryVerticalEnd,
      plan: 'Enterprise',
    },
    {
      name: 'Acme Corp.',
      logo: AudioWaveform,
      plan: 'Startup',
    },
    {
      name: 'Evil Corp.',
      logo: Command,
      plan: 'Free',
    },
  ],
  navMain: [
    {
      id: 'campaigns-nav',
      title: 'Campaigns',
      url: '/campaigns',
      icon: Flag,
    },
    {
      id: 'contacts-nav',
      title: 'Contacts',
      url: '#',
      icon: Users,
      isActive: true,
      items: [
        {
          title: 'Contacts',
          url: '/contacts',
        },
        {
          title: 'Contact Lists',
          url: '/contact-lists',
        },
        {
          title: 'Custom Fields',
          url: '/custom-fields',
        },
      ],
    },
    {
      id: 'settings-nav',
      title: 'Settings',
      url: '#',
      icon: Settings,
      isActive: true,
      items: [
        {
          title: 'Agents',
          url: '/agents',
        },
        {
          title: 'Company Information',
          url: '/company-information',
        },
        {
          title: 'Team Members',
          url: '/team-members',
        },
      ],
    },
  ],
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="icon" {...props} style={{ zIndex: 10 }}>
      <SidebarHeader>
        <TeamSwitcher teams={data.teams} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        {/* <NavProjects projects={data.projects} /> */}
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
