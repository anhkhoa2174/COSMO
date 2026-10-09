'use client';

import dynamic from 'next/dynamic';

import { Skeleton } from '@/components/ui/skeleton';

/**
 * The rich-text editor, loaded on demand.
 *
 * Plate and Slate together are the largest dependency in the application, and
 * the AI Inbox pays for them before showing one: the reading pane needs a
 * conversation selected, and the reply composer needs that conversation opened,
 * so on first paint the editor is not on screen. Measured on the production
 * build, the page scored 73 on Lighthouse against 96 for the landing page, with
 * 935 kB of first-load JavaScript.
 *
 * Only this page is switched. The campaign builder and the agent editor keep
 * the eager import, where the editor is closer to the point of entry and a
 * loading skeleton would be more intrusive than the bytes it saves.
 *
 * `ssr: false` is not only about weight. Slate builds its document model
 * against the DOM, so rendering it server-side and rehydrating produces
 * mismatch errors rather than a saving.
 */
export const PlateEditor = dynamic(
  () => import('./plate-editor').then((m) => m.PlateEditor),
  {
    ssr: false,
    loading: () => (
      // Roughly the height the editor settles at, so its arrival does not
      // shove the surrounding layout.
      <div className="space-y-2 p-2">
        <Skeleton className="h-6 w-1/3" />
        <Skeleton className="h-40 w-full" />
      </div>
    ),
  }
);
