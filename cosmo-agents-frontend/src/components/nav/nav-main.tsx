'use client';

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  SidebarGroup,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from '@/components/ui/sidebar';
import { ChevronDown, ChevronRight, type LucideIcon } from 'lucide-react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

export type NavItem = {
  id?: string;
  title: string;
  url: string;
  icon?: LucideIcon;
  badge?: number;
  items?: {
    title: string;
    url: string;
  }[];
};

export type NavSection = {
  /** Uppercase label above the group. Omit for the ungrouped items at the top. */
  label?: string;
  items: NavItem[];
  /** Groups start expanded unless this is false — INBOX ships collapsed. */
  defaultOpen?: boolean;
};

/**
 * A nav url matches when the pathname is the url itself or a child of it, so
 * `/tasks` does not light up on `/tasks-archive` the way a substring test would.
 */
function isUrlActive(pathname: string, url: string) {
  if (!url || url === '#') return false;
  return pathname === url || pathname.startsWith(`${url}/`);
}

function NavLeaf({ item }: { item: NavItem }) {
  const pathname = usePathname();
  const { open } = useSidebar();

  if (item.items) {
    const groupActive = item.items.some((sub) =>
      isUrlActive(pathname, sub.url)
    );

    if (!open) {
      return (
        <DropdownMenu>
          <SidebarMenuItem>
            <DropdownMenuTrigger asChild>
              <SidebarMenuButton tooltip={item.title} className="h-10">
                {item.icon && <item.icon />}
                <span>{item.title}</span>
                <ChevronRight className="ml-auto" />
              </SidebarMenuButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent side="right">
              {item.items.map((subItem) => (
                <DropdownMenuItem asChild key={subItem.title}>
                  <Link href={subItem.url} prefetch>
                    {subItem.title}
                  </Link>
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </SidebarMenuItem>
        </DropdownMenu>
      );
    }

    return (
      <Collapsible
        asChild
        defaultOpen={groupActive}
        className="group/collapsible"
      >
        <SidebarMenuItem>
          <CollapsibleTrigger asChild>
            <SidebarMenuButton
              tooltip={item.title}
              className="h-10"
              id={item.id}
            >
              {item.icon && <item.icon />}
              <span>{item.title}</span>
              <ChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
            </SidebarMenuButton>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <SidebarMenuSub>
              {item.items.map((subItem) => (
                <SidebarMenuSubItem key={subItem.title}>
                  <SidebarMenuSubButton
                    asChild
                    isActive={isUrlActive(pathname, subItem.url)}
                    className="h-9"
                  >
                    <Link href={subItem.url} prefetch>
                      <span>{subItem.title}</span>
                    </Link>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              ))}
            </SidebarMenuSub>
          </CollapsibleContent>
        </SidebarMenuItem>
      </Collapsible>
    );
  }

  const active = isUrlActive(pathname, item.url);

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        tooltip={item.title}
        asChild
        isActive={active}
        id={item.id}
        className={
          active
            ? 'h-10 rounded-lg bg-white/[0.07] font-semibold text-white ring-[1.5px] ring-inset ring-violet-500/70 hover:bg-white/[0.09] hover:text-white'
            : 'h-10 rounded-lg text-slate-400 hover:bg-white/[0.05] hover:text-slate-100'
        }
      >
        <Link href={item.url} prefetch>
          {item.icon && <item.icon />}
          <span>{item.title}</span>
        </Link>
      </SidebarMenuButton>
      {item.badge != null && item.badge > 0 && (
        <SidebarMenuBadge className="pointer-events-none top-2.5 h-5 min-w-5 justify-center rounded-full bg-red-500 px-1 text-[0.65rem] font-bold text-white">
          {item.badge}
        </SidebarMenuBadge>
      )}
    </SidebarMenuItem>
  );
}

function NavSectionGroup({ section }: { section: NavSection }) {
  const pathname = usePathname();
  const { open } = useSidebar();

  const body = (
    <SidebarMenu>
      {section.items.map((item) => (
        <NavLeaf key={item.title} item={item} />
      ))}
    </SidebarMenu>
  );

  // Ungrouped items (Dashboard) and the icon-collapsed rail render without a label.
  if (!section.label || !open) {
    return <SidebarGroup className="py-1">{body}</SidebarGroup>;
  }

  const sectionActive = section.items.some(
    (item) =>
      isUrlActive(pathname, item.url) ||
      item.items?.some((sub) => isUrlActive(pathname, sub.url))
  );

  return (
    // Sections start collapsed; only the one holding the current page opens,
    // so the sidebar stays short and still shows where the user is.
    <Collapsible
      defaultOpen={(section.defaultOpen ?? false) || sectionActive}
      className="group/section"
    >
      <SidebarGroup className="py-1">
        <CollapsibleTrigger
          // The tour points at a section even while it is collapsed, when
          // the items inside it are not rendered.
          id={`section-${section.label.toLowerCase()}`}
          className="flex h-8 w-full items-center justify-between rounded-md px-3 text-[0.7rem] font-semibold uppercase tracking-[0.14em] text-slate-500 transition-colors hover:text-slate-300"
        >
          <span>{section.label}</span>
          <ChevronDown className="size-4 transition-transform duration-200 group-data-[state=closed]/section:-rotate-90" />
        </CollapsibleTrigger>
        <CollapsibleContent>{body}</CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  );
}

export function NavMain({ sections }: { sections: NavSection[] }) {
  return (
    <>
      {sections.map((section, index) => (
        <NavSectionGroup
          key={section.label ?? `root-${index}`}
          section={section}
        />
      ))}
    </>
  );
}
