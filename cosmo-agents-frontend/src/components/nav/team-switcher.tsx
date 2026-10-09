'use client';

import Link from 'next/link';
import { CosmoMark } from '@/components/nav/cosmo-mark';
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar';

export function TeamSwitcher() {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          size="lg"
          className="border-b border-sidebar-border pb-4 hover:bg-transparent active:bg-transparent"
          asChild
        >
          <Link href="/dashboard">
            <CosmoMark size={32} />
            <span className="truncate text-[1.05rem] font-bold tracking-tight text-white">
              COSMO
            </span>
          </Link>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
