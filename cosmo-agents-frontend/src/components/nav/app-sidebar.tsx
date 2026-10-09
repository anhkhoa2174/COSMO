'use client';

import { NavMain, type NavSection } from '@/components/nav/nav-main';
import {
  Building2,
  CheckSquare,
  ClipboardList,
  Flag,
  FolderOpen,
  Inbox,
  LayoutDashboard,
  LayoutTemplate,
  Library,
  ListFilter,
  Mail,
  Send,
  Settings,
  SlidersHorizontal,
  Sparkles,
  Users,
  Zap,
} from 'lucide-react';
import * as React from 'react';
import { NavUser } from '@/components/nav/nav-user';
import { TeamSwitcher } from '@/components/nav/team-switcher';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar';

const sections: NavSection[] = [
  {
    items: [
      {
        id: 'dashboard-nav',
        title: 'Dashboard',
        url: '/dashboard',
        icon: LayoutDashboard,
      },
    ],
  },
  {
    label: 'Daily Work',
    items: [
      {
        id: 'daily-actions-nav',
        title: 'Daily Actions',
        url: '/daily-actions',
        icon: Zap,
        badge: 8,
      },
      {
        id: 'smart-insights-nav',
        title: 'Smart Insights',
        url: '/smart-insights',
        icon: Sparkles,
      },
    ],
  },
  {
    label: 'Inbox',
    defaultOpen: false,
    items: [
      {
        id: 'ai-inboxes-nav',
        title: 'AI Inboxes',
        url: '/ai-inboxes',
        icon: Inbox,
      },
      {
        id: 'emails-nav',
        title: 'Emails',
        url: '/emails',
        icon: Mail,
      },
    ],
  },
  {
    label: 'Pipeline',
    items: [
      {
        id: 'outreach-nav',
        title: 'Outreach',
        url: '/outreach',
        icon: Send,
      },
      {
        id: 'tasks-nav',
        title: 'Tasks',
        url: '/tasks',
        icon: CheckSquare,
      },
    ],
  },
  {
    label: 'Prospects',
    items: [
      {
        id: 'all-prospects-nav',
        title: 'All Prospects',
        url: '/all-prospects',
        icon: Users,
      },
      {
        id: 'audiences-nav',
        title: 'Audiences',
        url: '/audiences',
        icon: ListFilter,
      },
      {
        id: 'profile-fields-nav',
        title: 'Profile Fields',
        url: '/profile-fields',
        icon: SlidersHorizontal,
      },
      {
        id: 'lead-forms-nav',
        title: 'Lead Forms',
        url: '/lead-forms',
        icon: ClipboardList,
      },
    ],
  },
  {
    label: 'Campaigns',
    items: [
      {
        id: 'campaigns-nav',
        title: 'Campaigns',
        url: '/campaigns',
        icon: Flag,
      },
      {
        id: 'templates-nav',
        title: 'Templates',
        url: '/templates',
        icon: LayoutTemplate,
      },
    ],
  },
  {
    label: 'Knowledge',
    items: [
      {
        id: 'libraries-nav',
        title: 'Knowledge Base',
        url: '/libraries',
        icon: Library,
      },
      {
        id: 'files-nav',
        title: 'Files',
        url: '/files',
        icon: FolderOpen,
      },
    ],
  },
  {
    label: 'Settings',
    defaultOpen: false,
    items: [
      {
        id: 'agents-nav',
        title: 'Agents',
        url: '/agents',
        icon: Settings,
      },
      {
        id: 'organization-nav',
        title: 'Organization',
        url: '#',
        icon: Building2,
        items: [
          { title: 'Company Information', url: '/company-information' },
          { title: 'Team Members', url: '/team-members' },
          { title: 'Sales Reps', url: '/sales-reps' },
          { title: 'Outreach Timing', url: '/outreach-timing' },
        ],
      },
    ],
  },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="icon" {...props} style={{ zIndex: 10 }}>
      <SidebarHeader className="px-3 pt-4">
        <TeamSwitcher />
      </SidebarHeader>
      <SidebarContent className="px-1.5">
        <NavMain sections={sections} />
      </SidebarContent>
      <SidebarFooter className="border-t border-sidebar-border">
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
