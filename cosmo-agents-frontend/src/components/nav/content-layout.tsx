import React from 'react';
// import { Separator } from '@/components/ui/separator';
// import { SidebarTrigger } from '@/components/ui/sidebar';
import { cn } from '@/lib/utils';

export function ContentLayout({
  title,
  children,
  className,
  leftSection,
  rightSection,
}: {
  title: string | JSX.Element;
  children: React.ReactNode;
  className?: string;
  leftSection?: React.ReactNode;
  rightSection?: React.ReactNode;
}) {
  return (
    <>
      <header className="sticky top-0 z-50 flex h-16 w-full shrink-0 items-center justify-between gap-2 border-b bg-background px-4 shadow-sm transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-16">
        <div className="flex items-center gap-4">
          {/* <SidebarTrigger className="-ml-1" /> */}
          {/* <Separator orientation="vertical" className="mr-2 h-4" /> */}
          {leftSection}
          {typeof title === 'string' ? (
            <p className="text-base font-semibold">{title}</p>
          ) : (
            title
          )}
        </div>
        {rightSection}
      </header>
      <div
        className={cn(
          'flex flex-1 flex-col gap-4 overflow-y-auto p-4',
          className
        )}
      >
        {/* <div className="grid auto-rows-min gap-4 md:grid-cols-3">
            <div className="aspect-video rounded-xl bg-muted/50" />
            <div className="aspect-video rounded-xl bg-muted/50" />
            <div className="aspect-video rounded-xl bg-muted/50" />
          </div>
          <div className="min-h-[100vh] flex-1 rounded-xl bg-muted/50 md:min-h-min" /> */}
        {children}
      </div>
    </>
  );
}
