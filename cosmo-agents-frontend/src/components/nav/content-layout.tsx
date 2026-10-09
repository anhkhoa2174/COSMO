import React from 'react';
import { TopBar } from '@/components/nav/top-bar';
import { cn } from '@/lib/utils';

export function ContentLayout({
  title,
  section,
  icon,
  children,
  className,
  leftSection,
  rightSection,
  variant = 'page',
}: {
  title: string | JSX.Element;
  /** Breadcrumb parent shown before the page title, e.g. "Pipeline". */
  section?: string;
  icon?: React.ComponentType<{ className?: string }>;
  children: React.ReactNode;
  className?: string;
  leftSection?: React.ReactNode;
  rightSection?: React.ReactNode;
  /**
   * `editor` is for full-bleed document screens — the campaign builder, the
   * template editor, the playbook picker. They drop the utility cluster and
   * the page padding so the canvas owns the whole area.
   */
  variant?: 'page' | 'editor';
}) {
  const isEditor = variant === 'editor';

  return (
    <>
      <TopBar
        title={title}
        section={section}
        icon={icon}
        leftSection={leftSection}
        rightSection={rightSection}
        utilities={!isEditor}
      />
      <div
        className={cn(
          // min-w-0 lets this shrink inside the sidebar's flex row; without it
          // a wide child stretches the whole shell and clips content at both
          // edges instead of scrolling inside its own container.
          'flex min-w-0 flex-1 flex-col overflow-y-auto',
          isEditor ? 'bg-background' : 'gap-6 bg-muted/40 p-4 md:p-6',
          className
        )}
      >
        {children}
      </div>
    </>
  );
}
