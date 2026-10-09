'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { toast } from 'sonner';
import { useState } from 'react';

import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { Calendar } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DeleteButton } from '@/components/buttons/delete-button';
import OutreachApi from '@/network/client/outreach';
import type { MeetingStatus } from '@/network/client/outreach';

export default function MeetingsPage() {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const { data, isLoading } = useQuery({
    queryKey: ['meetings'],
    queryFn: () => OutreachApi.getAllMeetings(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => OutreachApi.deleteMeeting(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['meetings'] });
      toast.success('Meeting deleted');
    },
    onError: () => toast.error('Failed to delete meeting'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: MeetingStatus }) =>
      OutreachApi.updateMeeting(id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['meetings'] });
      toast.success('Meeting updated');
    },
    onError: () => toast.error('Failed to update meeting'),
  });

  const meetings = (data?.data || []).filter(
    (m) => statusFilter === 'all' || m.status === statusFilter
  );

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      scheduled: 'text-blue-600 bg-blue-500/10',
      completed: 'text-green-600 bg-green-500/10',
      cancelled: 'text-gray-500 bg-gray-500/10',
      no_show: 'text-red-600 bg-red-500/10',
    };
    return (
      <Badge className={`uppercase ${styles[status] || styles.cancelled}`}>
        {status.replace('_', ' ')}
      </Badge>
    );
  };

  const leftSection = (
    <Select value={statusFilter} onValueChange={setStatusFilter}>
      <SelectTrigger className="w-[140px]">
        <SelectValue placeholder="Filter status" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="all">All</SelectItem>
        <SelectItem value="scheduled">Scheduled</SelectItem>
        <SelectItem value="completed">Completed</SelectItem>
        <SelectItem value="cancelled">Cancelled</SelectItem>
        <SelectItem value="no_show">No Show</SelectItem>
      </SelectContent>
    </Select>
  );

  return (
    <ContentLayout
      title="Meetings"
      section="Pipeline"
      icon={Calendar}
      leftSection={leftSection}
    >
      <PageHero
        icon={Calendar}
        eyebrow="Pipeline"
        accent="amber"
        title="Meetings"
        description="Calls booked out of your outreach, with AI-generated prep notes and talking points for each one."
      />
      {isLoading && (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-24 w-full rounded-lg" />
          ))}
        </div>
      )}
      {!isLoading && meetings.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <p>No meetings found</p>
        </div>
      )}
      <div className="space-y-3">
        {meetings.map((meeting) => (
          <Card key={meeting.id}>
            <CardContent className="flex items-center justify-between p-4">
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <p className="font-medium">
                    {meeting.title || 'Untitled Meeting'}
                  </p>
                  {getStatusBadge(meeting.status)}
                </div>
                <p className="mt-1 text-sm text-muted-foreground">
                  {format(new Date(meeting.time), 'MMM d, yyyy HH:mm')}
                  {meeting.duration_minutes
                    ? ` · ${meeting.duration_minutes} min`
                    : ''}
                  {meeting.channel ? ` · ${meeting.channel}` : ''}
                  {meeting.location ? ` · ${meeting.location}` : ''}
                </p>
                {meeting.note && (
                  <p className="mt-1 text-sm text-muted-foreground line-clamp-1">
                    {meeting.note}
                  </p>
                )}
              </div>
              <div className="flex items-center gap-2">
                {meeting.status === 'scheduled' && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      updateMutation.mutate({
                        id: meeting.id,
                        status: 'completed',
                      })
                    }
                  >
                    Complete
                  </Button>
                )}
                <DeleteButton
                  size="icon"
                  variant="destructive-secondary"
                  onConfirm={() => deleteMutation.mutateAsync(meeting.id)}
                  description="This will permanently delete this meeting."
                />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </ContentLayout>
  );
}
