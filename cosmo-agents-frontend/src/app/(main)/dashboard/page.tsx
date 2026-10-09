'use client';

import { LayoutDashboard, User, UsersRound } from 'lucide-react';
import React from 'react';

import { ContentLayout } from '@/components/nav/content-layout';
import { Skeleton } from '@/components/ui/skeleton';
import { useUser } from '@/hooks/use-user';
import { cn } from '@/lib/utils';
import { MyDashboard } from '@/app/(main)/dashboard/_components/my-dashboard';
import { TeamDashboard } from '@/app/(main)/dashboard/_components/team-dashboard';

type View = 'team' | 'mine';

const VIEWS: { id: View; label: string; icon: typeof User }[] = [
  { id: 'team', label: 'Team', icon: UsersRound },
  { id: 'mine', label: 'My work', icon: User },
];

/**
 * An admin and a member are asking different questions of the same page. An
 * admin runs the team and opens on the team dashboard, with their own work one
 * click away; a member only ever sees their own. The role only chooses the
 * layout: what each view can show is scoped again on the server.
 */
export default function DashboardPage() {
  const { user, isLoading } = useUser();
  const isAdmin = user?.roles?.some((r) => r.name === 'admin') ?? false;
  const [view, setView] = React.useState<View>('team');

  return (
    <ContentLayout title="Dashboard" section="Home" icon={LayoutDashboard}>
      {isLoading ? (
        <Skeleton className="h-40" />
      ) : isAdmin ? (
        <>
          <div className="flex items-center gap-1 self-start rounded-full border bg-background p-1">
            {VIEWS.map(({ id, label, icon: Icon }) => (
              <button
                key={id}
                type="button"
                onClick={() => setView(id)}
                aria-pressed={view === id}
                className={cn(
                  'inline-flex items-center gap-1.5 rounded-full px-4 py-1.5 text-[0.85rem] font-medium transition-colors',
                  view === id
                    ? 'bg-violet-600 text-white'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                <Icon className="size-4" />
                {label}
              </button>
            ))}
          </div>
          {view === 'team' ? <TeamDashboard /> : <MyDashboard />}
        </>
      ) : (
        <MyDashboard />
      )}
    </ContentLayout>
  );
}
