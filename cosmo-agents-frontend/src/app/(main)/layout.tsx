import { PropsWithChildren } from 'react';

import { AppSidebar } from '@/components/nav/app-sidebar';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';

export default function MainLayout({ children }: PropsWithChildren) {
  return (
    <SidebarProvider defaultOpen>
      <AppSidebar />
      <SidebarInset>{children}</SidebarInset>
    </SidebarProvider>
  );
}
