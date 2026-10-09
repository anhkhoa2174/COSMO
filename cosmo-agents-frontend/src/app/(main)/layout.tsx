import { PropsWithChildren } from 'react';

import { AppSidebar } from '@/components/nav/app-sidebar';
import { GlobalNotifications } from '@/components/global-notifications';
import { CosmoChatProvider } from '@/components/cosmo-widget/cosmo-chat-provider';
import dynamic from 'next/dynamic';

/**
 * The widget sits in this layout so its thread survives navigation, which means
 * every authenticated page paid for it — the markdown renderer and the chat
 * runtime were in the first load of pages that never open it. It starts closed,
 * so the panel is loaded on demand instead.
 *
 * `ssr: false` because there is nothing to render server-side: the panel is
 * hidden until a click, and the click cannot happen during the server render.
 */
const CosmoChatWidget = dynamic(
  () =>
    import('@/components/cosmo-widget/cosmo-chat-widget').then(
      (m) => m.CosmoChatWidget
    ),
  { ssr: false }
);
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';

export default function MainLayout({ children }: PropsWithChildren) {
  return (
    <SidebarProvider defaultOpen>
      <AppSidebar />
      <SidebarInset>
        <GlobalNotifications />
        {/* The landmark is what lets a screen reader jump past the sidebar to
            the page itself. It wraps the children rather than the inset: the
            sidebar is navigation and belongs outside.
            
            It has to carry the layout classes too. A bare <main> is a block
            box, and dropping one between the flex column above and the pages
            below broke the chain every full-height screen depends on: the
            page's own `flex-1` had no flex parent left to grow inside, so it
            resolved to zero and took its children with it. The campaign
            builder showed this most plainly — React Flow measures its
            container and refuses to draw at zero height — but every page that
            fills the viewport was collapsing the same way. */}
        <main className="flex min-h-0 flex-1 flex-col">{children}</main>
      </SidebarInset>
      {/* Ask COSMO — mounted once here so the thread and open/closed state
          survive navigation between authenticated pages. Not present on the
          landing page, the legal pages or the auth screens. */}
      <CosmoChatProvider>
        <CosmoChatWidget />
      </CosmoChatProvider>
    </SidebarProvider>
  );
}
