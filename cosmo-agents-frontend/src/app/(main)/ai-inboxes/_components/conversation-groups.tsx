'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, FolderPlus, Plus, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { cn } from '@/lib/utils';
import ConversationGroupApi, {
  GROUP_COLORS,
  type ConversationGroup,
  type GroupColor,
} from '@/network/client/conversation-group';
import { HttpError } from '@/network/errors/httpErrors';

/** Tailwind classes per colour: a dot, and a tinted chip. */
export const GROUP_COLOR_STYLES: Record<
  GroupColor,
  { dot: string; chip: string }
> = {
  slate: { dot: 'bg-slate-500', chip: 'bg-slate-100 text-slate-700' },
  red: { dot: 'bg-red-500', chip: 'bg-red-50 text-red-700' },
  orange: { dot: 'bg-orange-500', chip: 'bg-orange-50 text-orange-700' },
  amber: { dot: 'bg-amber-500', chip: 'bg-amber-50 text-amber-800' },
  green: { dot: 'bg-green-500', chip: 'bg-green-50 text-green-700' },
  teal: { dot: 'bg-teal-500', chip: 'bg-teal-50 text-teal-700' },
  blue: { dot: 'bg-blue-500', chip: 'bg-blue-50 text-blue-700' },
  violet: { dot: 'bg-violet-500', chip: 'bg-violet-50 text-violet-700' },
  pink: { dot: 'bg-pink-500', chip: 'bg-pink-50 text-pink-700' },
};

export const GROUPS_QUERY_KEY = ['conversation-groups'];

/** The caller's own groups. Groups are personal, so this is never shared. */
export function useConversationGroups() {
  return useQuery({
    queryKey: GROUPS_QUERY_KEY,
    queryFn: ConversationGroupApi.list,
    staleTime: 30_000,
  });
}

export function GroupChip({ group }: { group: ConversationGroup }) {
  const style = GROUP_COLOR_STYLES[group.color] ?? GROUP_COLOR_STYLES.slate;
  return (
    <span
      className={cn(
        'inline-flex max-w-[10rem] items-center gap-1 rounded-full px-2 py-0.5 text-xs',
        style.chip
      )}
    >
      <span className={cn('size-1.5 shrink-0 rounded-full', style.dot)} />
      <span className="truncate">{group.name}</span>
    </span>
  );
}

// The ky client's beforeError hook rethrows API errors as HttpError
// subclasses carrying the server's message (e.g. the 409 for a duplicate
// name). Other errors fall back to a generic line.
async function errorMessage(err: unknown, fallback: string) {
  if (err instanceof HttpError && err.message) return err.message;
  return fallback;
}

interface GroupPickerProps {
  conversationId: string;
  groupIds: string[];
  /**
   * Called after a change is saved, with the conversation it was made on: the
   * user may have opened another conversation while the request was in flight.
   */
  onChange: (conversationId: string, groupIds: string[]) => void;
}

/**
 * Files the open conversation in the user's groups, and creates or deletes
 * groups on the way. Each toggle is saved as it is clicked, so there is no
 * "Save" to forget.
 */
export function GroupPicker({
  conversationId,
  groupIds,
  onChange,
}: GroupPickerProps) {
  const queryClient = useQueryClient();
  const { data: groups = [] } = useConversationGroups();
  const [name, setName] = useState('');
  const [color, setColor] = useState<GroupColor>('blue');

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: GROUPS_QUERY_KEY });
    queryClient.invalidateQueries({ queryKey: ['conversations'] });
  };

  const toggle = useMutation({
    mutationFn: async (group: ConversationGroup) => {
      const inGroup = groupIds.includes(group.id);
      if (inGroup) {
        await ConversationGroupApi.removeConversation(group.id, conversationId);
        return groupIds.filter((id) => id !== group.id);
      }
      await ConversationGroupApi.addConversation(group.id, conversationId);
      return [...groupIds, group.id];
    },
    onSuccess: (next) => {
      onChange(conversationId, next);
      refresh();
    },
    onError: async (err) =>
      toast.error(await errorMessage(err, 'Could not update the group')),
  });

  const create = useMutation({
    mutationFn: async () => {
      const group = await ConversationGroupApi.create({
        name: name.trim(),
        color,
      });
      // A new group is created from this conversation, so file it there.
      try {
        await ConversationGroupApi.addConversation(group.id, conversationId);
      } catch {
        // The group exists now; show it, and say what did not happen,
        // rather than "could not create" for a group that was created.
        refresh();
        throw new HttpError(
          `Created "${group.name}", but could not add this conversation to it`
        );
      }
      return group;
    },
    onSuccess: (group) => {
      setName('');
      onChange(conversationId, [...groupIds, group.id]);
      refresh();
      toast.success(`Added to "${group.name}"`);
    },
    onError: async (err) =>
      toast.error(await errorMessage(err, 'Could not create the group')),
  });

  const remove = useMutation({
    mutationFn: async (group: ConversationGroup) => {
      await ConversationGroupApi.remove(group.id);
      return group;
    },
    onSuccess: (group) => {
      onChange(
        conversationId,
        groupIds.filter((id) => id !== group.id)
      );
      refresh();
      toast.success(`Deleted "${group.name}"`);
    },
    onError: async (err) =>
      toast.error(await errorMessage(err, 'Could not delete the group')),
  });

  const current = groups.filter((g) => groupIds.includes(g.id));

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 gap-1.5 px-2.5">
          <FolderPlus className="size-3.5" />
          {current.length === 0 ? (
            'Group'
          ) : (
            <span className="max-w-[9rem] truncate">
              {current.map((g) => g.name).join(', ')}
            </span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72 p-3">
        <p className="mb-2 text-xs font-medium">Your groups</p>
        {groups.length === 0 ? (
          <p className="mb-3 text-xs text-muted-foreground">
            No groups yet. Groups are yours alone; nobody else sees them.
          </p>
        ) : (
          <ul className="mb-3 max-h-56 space-y-0.5 overflow-y-auto">
            {groups.map((group) => {
              const checked = groupIds.includes(group.id);
              const style =
                GROUP_COLOR_STYLES[group.color] ?? GROUP_COLOR_STYLES.slate;
              return (
                <li key={group.id} className="group flex items-center gap-1">
                  <button
                    type="button"
                    disabled={toggle.isPending}
                    onClick={() => toggle.mutate(group)}
                    className="flex min-w-0 flex-1 items-center gap-2 rounded px-1.5 py-1 text-left text-sm hover:bg-accent"
                  >
                    <span
                      className={cn(
                        'grid size-4 shrink-0 place-items-center rounded border',
                        checked && 'border-primary bg-primary'
                      )}
                    >
                      {checked && (
                        <Check className="size-3 text-primary-foreground" />
                      )}
                    </span>
                    <span
                      className={cn('size-2 shrink-0 rounded-full', style.dot)}
                    />
                    <span className="truncate">{group.name}</span>
                    <span className="ml-auto text-xs text-muted-foreground">
                      {group.conversation_count ?? 0}
                    </span>
                  </button>
                  <button
                    type="button"
                    aria-label={`Delete group ${group.name}`}
                    onClick={() => {
                      if (
                        window.confirm(
                          `Delete "${group.name}"? Its conversations stay in your inbox.`
                        )
                      ) {
                        remove.mutate(group);
                      }
                    }}
                    className="rounded p-1 text-muted-foreground opacity-0 hover:text-destructive group-hover:opacity-100"
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                </li>
              );
            })}
          </ul>
        )}

        <form
          className="space-y-2 border-t pt-3"
          onSubmit={(e) => {
            e.preventDefault();
            if (name.trim()) create.mutate();
          }}
        >
          <p className="text-xs font-medium">New group</p>
          <div className="flex gap-1.5">
            <Input
              value={name}
              maxLength={50}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Acme Corp, VIP, Product A"
              className="h-8 text-sm"
            />
            <Button
              type="submit"
              size="sm"
              className="h-8 px-2"
              disabled={!name.trim() || create.isPending}
              aria-label="Create group"
            >
              <Plus className="size-4" />
            </Button>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {GROUP_COLORS.map((c) => (
              <button
                key={c}
                type="button"
                aria-label={`Colour ${c}`}
                onClick={() => setColor(c)}
                className={cn(
                  'size-5 rounded-full ring-offset-2',
                  GROUP_COLOR_STYLES[c].dot,
                  color === c && 'ring-2 ring-primary'
                )}
              />
            ))}
          </div>
        </form>
      </PopoverContent>
    </Popover>
  );
}
