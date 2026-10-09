'use client';

import { ContentLayout } from '@/components/nav/content-layout';
import { PageHero } from '@/components/ui/page-hero';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  MessageSquare,
  Users,
  Send,
  Calendar,
  History,
  Loader2,
  RefreshCw,
  Clock,
  ArrowRight,
  Mail,
  Linkedin,
  Phone,
  FileText,
  BarChart3,
  TrendingUp,
  Lightbulb,
  Target,
  Sparkles,
  FileSearch,
  CheckCircle,
  Copy,
  Trash2,
  Filter,
} from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useState, useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import ContactApi from '@/network/client/contact';
import OutreachApi, {
  type OutreachSuggestion,
  type InteractionLog,
  type Meeting,
  type ConversationState,
  type NextStep,
  type FeedbackStats,
  type ScenarioStats,
} from '@/network/client/outreach';
import { toast } from 'sonner';
import { NextStepPanel } from '@/components/outreach/next-step-panel';
import {
  NEXT_ACTION_GROUPS,
  compareByNextActionDue,
  describeNextAction,
  nextActionColors,
  nextActionGroup,
} from '@/lib/next-step';
import type { NextAction } from '@/network/client/outreach';
import { formatDistanceToNow } from 'date-fns';

// State color mapping
const stateColors: Record<ConversationState, string> = {
  COLD: 'bg-blue-100 text-blue-800',
  NO_REPLY: 'bg-yellow-100 text-yellow-800',
  REPLIED: 'bg-green-100 text-green-800',
  POST_MEETING: 'bg-purple-100 text-purple-800',
  DROPPED: 'bg-gray-100 text-gray-800',
};

const nextStepColors: Record<NextStep, string> = {
  SEND: 'bg-blue-500',
  FOLLOW_UP_1: 'bg-yellow-500',
  FOLLOW_UP_2: 'bg-orange-500',
  SET_MEETING: 'bg-purple-500',
  FOLLOW_UP_MEETING_1: 'bg-indigo-500',
  FOLLOW_UP_MEETING_2: 'bg-violet-500',
  PREPARE_MEETING: 'bg-green-500',
  WAIT: 'bg-gray-500',
  FOLLOW_UP: 'bg-teal-500', // Post-meeting follow-up
  DROP: 'bg-red-500',
};

// NextStep labels for display
const nextStepLabels: Record<NextStep, string> = {
  SEND: 'Send Initial',
  FOLLOW_UP_1: 'Follow-up #1',
  FOLLOW_UP_2: 'Follow-up #2',
  SET_MEETING: 'Set Meeting',
  FOLLOW_UP_MEETING_1: 'Confirm #1',
  FOLLOW_UP_MEETING_2: 'Confirm #2',
  PREPARE_MEETING: 'Prepare Meeting',
  WAIT: 'Wait',
  FOLLOW_UP: 'Follow-up Deal',
  DROP: 'Drop',
};

const channelIcons: Record<string, React.ReactNode> = {
  LinkedIn: <Linkedin className="h-4 w-4" />,
  Email: <Mail className="h-4 w-4" />,
  Call: <Phone className="h-4 w-4" />,
  Meeting: <Calendar className="h-4 w-4" />,
  Note: <FileText className="h-4 w-4" />,
};

export default function OutreachTestPage() {
  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const [selectedContactId, setSelectedContactId] = useState<string | null>(
    null
  );

  // Filter states
  const [statusFilter, setStatusFilter] = useState<string>('ready');
  // The chips filter by the engine's decision (next_action), not the cadence's
  // next_step, so a chip and the badge on each card always agree.
  const [actionGroupFilter, setActionGroupFilter] = useState<string>('__all__');

  // Auto-select contact from URL query parameter
  useEffect(() => {
    const contactParam = searchParams.get('contact');
    if (contactParam && !selectedContactId) {
      setSelectedContactId(contactParam);
    }
  }, [searchParams, selectedContactId]);
  const [updateEvent, setUpdateEvent] = useState<string>('sent');
  const [updateContent, setUpdateContent] = useState('');
  const [updateChannel, setUpdateChannel] = useState<string>('LinkedIn');
  const [updateSentiment, setUpdateSentiment] = useState<string>('neutral');
  const [meetingTitle, setMeetingTitle] = useState('');
  const [meetingTime, setMeetingTime] = useState('');
  const [meetingDuration, setMeetingDuration] = useState(30);

  // Language state for draft generation
  const [draftLanguage, setDraftLanguage] = useState<'vi' | 'en'>('vi');

  // Add interaction form state
  const [interactionContent, setInteractionContent] = useState('');
  const [interactionRole, setInteractionRole] = useState<'me' | 'client'>('me');
  const [interactionChannel, setInteractionChannel] =
    useState<string>('LinkedIn');

  // Meeting prep states
  const [meetingContentDialogOpen, setMeetingContentDialogOpen] =
    useState(false);
  const [meetingPrepDialogOpen, setMeetingPrepDialogOpen] = useState(false);
  const [selectedMeeting, setSelectedMeeting] = useState<Meeting | null>(null);
  const [meetingContentInput, setMeetingContentInput] = useState('');

  // Build filter for contacts query
  const contactsFilter: Record<string, any> = {};
  if (statusFilter && statusFilter !== '__all__') {
    contactsFilter.status = statusFilter;
  }

  // Fetch contacts (increased limit for admin to see all)
  const {
    data: contacts,
    isLoading: loadingContacts,
    isError: contactsError,
  } = useQuery({
    queryKey: ['contacts-for-outreach', statusFilter],
    queryFn: () =>
      ContactApi.list({ filter: contactsFilter }, { offset: 0, limit: 1000 }),
  });

  const contactTotal =
    contacts?.data?.total ?? contacts?.data?.list?.length ?? 0;

  // Grouped and sorted on the client: the chips need a count per group, and
  // the whole list (up to 1000) is already loaded for the status filter.
  const allContacts: any[] = contacts?.data?.list ?? [];
  const groupCounts: Record<string, number> = {};
  for (const item of allContacts) {
    const g = nextActionGroup(item.entity?.next_action);
    if (g) groupCounts[g] = (groupCounts[g] ?? 0) + 1;
  }
  const visibleContacts = allContacts
    .filter(
      (item) =>
        actionGroupFilter === '__all__' ||
        nextActionGroup(item.entity?.next_action) === actionGroupFilter
    )
    .sort((a, b) => compareByNextActionDue(a.entity ?? {}, b.entity ?? {}));
  const activeGroupLabel = NEXT_ACTION_GROUPS.find(
    (g) => g.id === actionGroupFilter
  )?.label;

  // Fetch outreach state for selected contact
  const { data: outreachState, isLoading: loadingState } = useQuery({
    queryKey: ['outreach-state', selectedContactId],
    queryFn: () => OutreachApi.getOutreachState(selectedContactId!),
    enabled: !!selectedContactId,
  });

  // Fetch interaction history for selected contact
  const { data: interactions, isLoading: loadingInteractions } = useQuery({
    queryKey: ['interaction-history', selectedContactId],
    queryFn: () => OutreachApi.getInteractionHistory(selectedContactId!, 20),
    enabled: !!selectedContactId,
  });

  // Fetch meetings for selected contact
  const { data: meetings, isLoading: loadingMeetings } = useQuery({
    queryKey: ['meetings', selectedContactId],
    queryFn: () => OutreachApi.getMeetings(selectedContactId!),
    enabled: !!selectedContactId,
  });

  // Fetch feedback stats
  const {
    data: feedbackStats,
    isLoading: loadingFeedbackStats,
    refetch: refetchFeedbackStats,
  } = useQuery({
    queryKey: ['feedback-stats'],
    queryFn: () => OutreachApi.getFeedbackStats(),
  });

  // Fetch scenario stats
  const { data: scenarioStats, isLoading: loadingScenarioStats } = useQuery({
    queryKey: ['scenario-stats'],
    queryFn: () => OutreachApi.getScenarioStats(),
  });

  // Generate draft mutation
  const generateDraftMutation = useMutation({
    mutationFn: (contactId: string) =>
      OutreachApi.generateDraft(contactId, draftLanguage),
    onSuccess: (response) => {
      toast.success('Draft generated successfully!');
      queryClient.invalidateQueries({
        queryKey: ['outreach-state', selectedContactId],
      });
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to generate draft');
    },
  });

  // Update outreach mutation
  const updateOutreachMutation = useMutation({
    mutationFn: () =>
      OutreachApi.updateOutreach(selectedContactId!, updateEvent as any, {
        content: updateContent,
        channel: updateChannel as any,
        sentiment: updateSentiment as any,
      }),
    onSuccess: (response) => {
      toast.success(
        `Outreach updated: ${response.data.previous_state} → ${response.data.new_state}`
      );
      // Immediately update caches with response data (no refetch needed)
      if (response.data.state) {
        queryClient.setQueryData(
          ['outreach-state', selectedContactId],
          (old: any) => ({
            ...old,
            data: response.data.state,
          })
        );
        // Update contact list sidebar with new next_step + outreach_stage
        queryClient.setQueriesData(
          { queryKey: ['contacts-for-outreach'] },
          (old: any) => {
            if (!old?.data?.list) return old;
            return {
              ...old,
              data: {
                ...old.data,
                list: old.data.list.map((item: any) =>
                  item.entity?.id === selectedContactId
                    ? {
                        ...item,
                        entity: {
                          ...item.entity,
                          next_step: response.data.state.next_step,
                          outreach_stage:
                            response.data.state.conversation_state,
                        },
                      }
                    : item
                ),
              },
            };
          }
        );
      }
      // Only refetch interaction history (new interaction was added by backend)
      queryClient.invalidateQueries({
        queryKey: ['interaction-history', selectedContactId],
      });
      queryClient.invalidateQueries({ queryKey: ['outreach-suggestions'] });
      setUpdateContent('');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to update outreach');
    },
  });

  // Create meeting mutation
  const createMeetingMutation = useMutation({
    mutationFn: () => {
      // Convert datetime-local format to ISO8601/RFC3339
      const meetingTimeISO = meetingTime
        ? new Date(meetingTime).toISOString()
        : '';
      return OutreachApi.createMeeting({
        contact_id: selectedContactId!,
        title: meetingTitle,
        time: meetingTimeISO,
        duration_minutes: meetingDuration,
        channel: 'Zoom',
      });
    },
    onSuccess: () => {
      toast.success('Meeting created!');
      queryClient.invalidateQueries({
        queryKey: ['meetings', selectedContactId],
      });
      queryClient.invalidateQueries({
        queryKey: ['outreach-state', selectedContactId],
      });
      setMeetingTitle('');
      setMeetingTime('');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to create meeting');
    },
  });

  // Add interaction mutation
  const addInteractionMutation = useMutation({
    mutationFn: () =>
      OutreachApi.addInteraction(selectedContactId!, {
        content: interactionContent,
        role: interactionRole,
        channel: interactionChannel as any,
      }),
    onSuccess: () => {
      toast.success(
        `Conversation added (${interactionRole === 'me' ? 'Outgoing' : 'Incoming'})`
      );
      queryClient.invalidateQueries({
        queryKey: ['interaction-history', selectedContactId],
      });
      setInteractionContent('');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to add interaction');
    },
  });

  // Update meeting mutation (for status and content)
  const updateMeetingMutation = useMutation({
    mutationFn: (data: {
      meetingId: string;
      status?: string;
      meeting_content?: string;
    }) =>
      OutreachApi.updateMeeting(data.meetingId, {
        status: data.status as any,
        meeting_content: data.meeting_content,
      }),
    onSuccess: () => {
      toast.success('Meeting updated!');
      queryClient.invalidateQueries({
        queryKey: ['meetings', selectedContactId],
      });
      setMeetingContentDialogOpen(false);
      setMeetingContentInput('');
      setSelectedMeeting(null);
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to update meeting');
    },
  });

  // Generate meeting prep mutation
  const generateMeetingPrepMutation = useMutation({
    mutationFn: (meetingId: string) =>
      OutreachApi.generateMeetingPrep(meetingId),
    onSuccess: () => {
      toast.success('Meeting Prep generated successfully!');
      queryClient.invalidateQueries({
        queryKey: ['meetings', selectedContactId],
      });
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to generate meeting prep');
    },
  });

  // Delete meeting mutation
  const deleteMeetingMutation = useMutation({
    mutationFn: (meetingId: string) => OutreachApi.deleteMeeting(meetingId),
    onSuccess: () => {
      toast.success('Meeting deleted!');
      queryClient.invalidateQueries({
        queryKey: ['meetings', selectedContactId],
      });
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to delete meeting');
    },
  });

  const selectedContact = contacts?.data?.list?.find(
    (item: any) => item.entity?.id === selectedContactId
  )?.entity;

  return (
    <ContentLayout title="Outreach" section="Pipeline" icon={Send}>
      <div className="space-y-6">
        <PageHero
          icon={Send}
          accent="violet"
          title="Outreach"
          titleAdornment={
            <span className="rounded-full border bg-background/70 px-3 py-1 text-[0.8rem] font-medium">
              {contactTotal} contact{contactTotal === 1 ? '' : 's'}
            </span>
          }
          description="Generate AI drafts, log conversations, schedule meetings, and track every touchpoint of your pipeline in one place."
        >
          <div className="flex flex-wrap gap-2 pt-3">
            {[
              { id: '__all__', label: 'All', count: allContacts.length },
              ...NEXT_ACTION_GROUPS.map((g) => ({
                id: g.id,
                label: g.label,
                count: groupCounts[g.id] ?? 0,
              })),
            ]
              .filter(
                (chip) =>
                  chip.id === '__all__' ||
                  chip.count > 0 ||
                  chip.id === actionGroupFilter
              )
              .map((chip) => {
                const active = chip.id === actionGroupFilter;
                return (
                  <button
                    key={chip.id}
                    type="button"
                    aria-pressed={active}
                    onClick={() => setActionGroupFilter(chip.id)}
                    className={cn(
                      'inline-flex items-center gap-1.5 rounded-full px-4 py-1.5 text-[0.85rem] font-medium transition-colors',
                      active
                        ? 'bg-violet-600 text-white shadow-sm'
                        : 'border bg-background/70 text-foreground hover:bg-background'
                    )}
                  >
                    {chip.label}
                    <span
                      className={cn(
                        'rounded-full px-1.5 text-[0.75rem] tabular-nums',
                        active
                          ? 'bg-white/20'
                          : 'bg-muted text-muted-foreground'
                      )}
                    >
                      {chip.count}
                    </span>
                  </button>
                );
              })}
          </div>
        </PageHero>

        <div className="grid gap-6 lg:grid-cols-3">
          {/* Left Column: Contacts */}
          <div className="space-y-4 lg:col-span-1">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  <span className="flex items-center gap-2">
                    <Users className="h-5 w-5" />
                    Contacts
                  </span>
                  <Badge variant="secondary" className="text-xs">
                    {visibleContacts.length}
                  </Badge>
                </CardTitle>
                <CardDescription>
                  Select a contact to manage outreach
                </CardDescription>
                {/* Filters */}
                <div className="flex flex-col gap-2 pt-2">
                  <div className="flex items-center gap-2">
                    <Filter className="h-4 w-4 text-muted-foreground" />
                    <Select
                      value={statusFilter}
                      onValueChange={setStatusFilter}
                    >
                      <SelectTrigger className="h-8 text-xs">
                        <SelectValue placeholder="Status" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__all__">All Status</SelectItem>
                        <SelectItem value="ready">Ready</SelectItem>
                        <SelectItem value="pending">Pending</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="max-h-[500px] space-y-2 overflow-y-auto">
                  {loadingContacts ? (
                    <div className="flex items-center justify-center p-8">
                      <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                    </div>
                  ) : contactsError ? (
                    <p className="p-4 text-center text-sm text-red-500">
                      Failed to load contacts. Please try again.
                    </p>
                  ) : visibleContacts.length === 0 ? (
                    <p className="p-4 text-center text-sm text-muted-foreground">
                      {activeGroupLabel
                        ? `No contacts whose next step is "${activeGroupLabel}"`
                        : 'No contacts available'}
                    </p>
                  ) : (
                    visibleContacts.map((item: any, index: number) => {
                      const nextStep = item.entity?.next_step as NextStep;
                      return (
                        <div
                          key={item.entity?.id}
                          className={`cursor-pointer rounded-lg border p-3 transition-colors ${
                            selectedContactId === item.entity?.id
                              ? 'border-primary bg-primary/5'
                              : 'hover:bg-accent'
                          }`}
                          onClick={() => setSelectedContactId(item.entity?.id)}
                        >
                          <div className="mb-1 flex items-center justify-between">
                            <span className="flex-1 truncate text-sm font-medium">
                              #{index + 1} {item.entity?.name}
                            </span>
                            <Badge variant="outline" className="ml-1 text-xs">
                              {item.entity?.status || 'new'}
                            </Badge>
                          </div>
                          <p className="truncate text-xs text-muted-foreground">
                            {[item.entity?.company, item.entity?.job_title]
                              .filter(Boolean)
                              .join(' • ') || '-'}
                          </p>
                          {item.entity?.next_action ? (
                            <div className="mt-1">
                              <Badge
                                className={`max-w-full truncate text-xs text-white ${nextActionColors[item.entity.next_action as NextAction] || 'bg-gray-500'}`}
                                title={item.entity?.next_action_reason}
                              >
                                {describeNextAction(
                                  item.entity.next_action as NextAction,
                                  item.entity.next_action_args,
                                  item.entity.next_action_due_at
                                )}
                              </Badge>
                            </div>
                          ) : (
                            nextStep && (
                              <div className="mt-1">
                                <Badge
                                  className={`text-xs text-white ${nextStepColors[nextStep] || 'bg-gray-500'}`}
                                >
                                  {nextStepLabels[nextStep] || nextStep}
                                </Badge>
                              </div>
                            )
                          )}
                        </div>
                      );
                    })
                  )}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Right Column: Details & Actions */}
          <div className="space-y-4 lg:col-span-2">
            {selectedContactId ? (
              <Tabs defaultValue="state" className="w-full">
                <TabsList className="grid w-full grid-cols-5">
                  <TabsTrigger value="state">State</TabsTrigger>
                  <TabsTrigger value="draft">Draft</TabsTrigger>
                  <TabsTrigger value="update">Update</TabsTrigger>
                  <TabsTrigger value="meeting">Meeting</TabsTrigger>
                  <TabsTrigger value="feedback">Feedback</TabsTrigger>
                </TabsList>

                {/* State Tab */}
                <TabsContent value="state" className="space-y-4">
                  <NextStepPanel contactId={selectedContactId} />
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center justify-between">
                        <span>Outreach State</span>
                      </CardTitle>
                      {selectedContact && (
                        <CardDescription className="mt-2 space-y-1">
                          <p className="font-medium text-foreground">
                            {selectedContact.name}
                          </p>
                          <p className="text-sm">
                            {selectedContact.job_title &&
                              `${selectedContact.job_title}`}
                            {selectedContact.job_title &&
                              selectedContact.company &&
                              ' at '}
                            {selectedContact.company &&
                              `${selectedContact.company}`}
                          </p>
                          {selectedContact.email && (
                            <p className="text-xs">{selectedContact.email}</p>
                          )}
                          {selectedContact.contact_information && (
                            <a
                              href={selectedContact.contact_information}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="flex items-center gap-1 text-xs text-blue-600 hover:underline"
                            >
                              <Linkedin className="h-3 w-3" />
                              LinkedIn Profile
                            </a>
                          )}
                        </CardDescription>
                      )}
                    </CardHeader>
                    <CardContent>
                      {loadingState ? (
                        <div className="flex items-center justify-center p-8">
                          <Loader2 className="h-8 w-8 animate-spin" />
                        </div>
                      ) : outreachState?.data ? (
                        <div className="space-y-4">
                          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                State
                              </p>
                              <Badge
                                className={
                                  stateColors[
                                    outreachState.data.conversation_state
                                  ]
                                }
                              >
                                {outreachState.data.conversation_state}
                              </Badge>
                            </div>
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Context
                              </p>
                              <p className="font-medium">
                                {outreachState.data.context_level}
                              </p>
                            </div>
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Intent
                              </p>
                              <p className="font-medium">
                                {outreachState.data.outreach_intent}
                              </p>
                            </div>
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Cadence step
                              </p>
                              <Badge
                                className={`${nextStepColors[outreachState.data.next_step] || 'bg-gray-500'} text-white`}
                              >
                                {nextStepLabels[outreachState.data.next_step] ||
                                  outreachState.data.next_step}
                              </Badge>
                            </div>
                          </div>

                          <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Scenario
                              </p>
                              <p className="font-medium">
                                {outreachState.data.scenario}
                              </p>
                            </div>
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Last Outcome
                              </p>
                              <p className="font-medium">
                                {outreachState.data.last_outcome}
                              </p>
                            </div>
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Days Since
                              </p>
                              <p className="font-medium">
                                {outreachState.data.days_since_last_interaction}
                              </p>
                            </div>
                          </div>

                          {outreachState.data.message_draft && (
                            <div className="rounded-lg border bg-muted/50 p-4">
                              <p className="mb-2 text-xs text-muted-foreground">
                                Current Draft
                              </p>
                              <p className="whitespace-pre-wrap text-sm">
                                {outreachState.data.message_draft}
                              </p>
                            </div>
                          )}
                        </div>
                      ) : (
                        <p className="p-4 text-center text-muted-foreground">
                          No outreach state found. Generate a draft to create
                          one.
                        </p>
                      )}
                    </CardContent>
                  </Card>

                  {/* Add Conversation History */}
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <MessageSquare className="h-5 w-5" />
                        Add Conversation
                      </CardTitle>
                      <CardDescription>
                        Log a message you sent or received from the client
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label>Role</Label>
                          <Select
                            value={interactionRole}
                            onValueChange={(v) =>
                              setInteractionRole(v as 'me' | 'client')
                            }
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="me">Me (Outgoing)</SelectItem>
                              <SelectItem value="client">
                                Client (Incoming)
                              </SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="space-y-2">
                          <Label>Channel</Label>
                          <Select
                            value={interactionChannel}
                            onValueChange={setInteractionChannel}
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="LinkedIn">LinkedIn</SelectItem>
                              <SelectItem value="Email">Email</SelectItem>
                              <SelectItem value="Call">Call</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>
                      <div className="space-y-2">
                        <Label>Message Content</Label>
                        <Textarea
                          placeholder={
                            interactionRole === 'me'
                              ? 'What did you send?'
                              : 'What did they reply?'
                          }
                          value={interactionContent}
                          onChange={(e) =>
                            setInteractionContent(e.target.value)
                          }
                          rows={3}
                        />
                      </div>
                      <Button
                        onClick={() => addInteractionMutation.mutate()}
                        disabled={
                          addInteractionMutation.isPending ||
                          !interactionContent.trim()
                        }
                        className="w-full"
                        variant={
                          interactionRole === 'me' ? 'default' : 'secondary'
                        }
                      >
                        {addInteractionMutation.isPending ? (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : interactionRole === 'me' ? (
                          <Send className="mr-2 h-4 w-4" />
                        ) : (
                          <MessageSquare className="mr-2 h-4 w-4" />
                        )}
                        {interactionRole === 'me'
                          ? 'Log Sent Message'
                          : 'Log Received Reply'}
                      </Button>
                    </CardContent>
                  </Card>

                  {/* Interaction History */}
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <History className="h-5 w-5" />
                        Interaction History
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      {loadingInteractions ? (
                        <div className="flex items-center justify-center p-4">
                          <Loader2 className="h-6 w-6 animate-spin" />
                        </div>
                      ) : interactions?.data?.length === 0 ? (
                        <p className="p-4 text-center text-muted-foreground">
                          No interactions yet
                        </p>
                      ) : (
                        <div className="space-y-3">
                          {interactions?.data?.map(
                            (interaction: InteractionLog) => (
                              <div
                                key={interaction.id}
                                className="flex gap-3 rounded-lg border p-3"
                              >
                                <div className="mt-1 flex-shrink-0">
                                  {channelIcons[interaction.channel] || (
                                    <MessageSquare className="h-4 w-4" />
                                  )}
                                </div>
                                <div className="min-w-0 flex-1">
                                  <div className="mb-1 flex items-center gap-2">
                                    <Badge
                                      variant="outline"
                                      className="text-xs"
                                    >
                                      {interaction.direction}
                                    </Badge>
                                    <span className="text-xs text-muted-foreground">
                                      {formatDistanceToNow(
                                        new Date(interaction.timestamp),
                                        { addSuffix: true }
                                      )}
                                    </span>
                                    {interaction.sentiment && (
                                      <Badge
                                        variant="outline"
                                        className={`text-xs ${
                                          interaction.sentiment === 'positive'
                                            ? 'text-green-600'
                                            : interaction.sentiment ===
                                                'negative'
                                              ? 'text-red-600'
                                              : ''
                                        }`}
                                      >
                                        {interaction.sentiment}
                                      </Badge>
                                    )}
                                  </div>
                                  <p className="line-clamp-2 text-sm">
                                    {interaction.content}
                                  </p>
                                </div>
                              </div>
                            )
                          )}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>

                {/* Draft Tab */}
                <TabsContent value="draft">
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <MessageSquare className="h-5 w-5" />
                          Generate Draft
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-7 text-xs"
                          onClick={() =>
                            setDraftLanguage((lang) =>
                              lang === 'vi' ? 'en' : 'vi'
                            )
                          }
                        >
                          {draftLanguage === 'vi'
                            ? '🇻🇳 Vietnamese'
                            : '🇬🇧 English'}
                        </Button>
                      </CardTitle>
                      <CardDescription>
                        Generate a personalized message based on context and
                        history
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <Button
                        onClick={() =>
                          generateDraftMutation.mutate(selectedContactId)
                        }
                        disabled={generateDraftMutation.isPending}
                        className="w-full"
                      >
                        {generateDraftMutation.isPending ? (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                          <MessageSquare className="mr-2 h-4 w-4" />
                        )}
                        Generate Draft
                      </Button>

                      {generateDraftMutation.data?.data && (
                        <div className="space-y-4 border-t pt-4">
                          <div className="grid grid-cols-2 gap-4">
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Scenario
                              </p>
                              <p className="font-medium">
                                {generateDraftMutation.data.data.scenario}
                              </p>
                            </div>
                            <div className="rounded-lg border p-3">
                              <p className="text-xs text-muted-foreground">
                                Context Level
                              </p>
                              <p className="font-medium">
                                {generateDraftMutation.data.data.context_level}
                              </p>
                            </div>
                          </div>
                          <div className="rounded-lg border bg-muted/50 p-4">
                            <p className="mb-2 text-xs text-muted-foreground">
                              Generated Draft
                            </p>
                            <p className="whitespace-pre-wrap text-sm">
                              {generateDraftMutation.data.data.draft}
                            </p>
                          </div>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>

                {/* Update Tab */}
                <TabsContent value="update">
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <Send className="h-5 w-5" />
                        Update Outreach
                      </CardTitle>
                      <CardDescription>
                        Record an outreach event (sent, replied, no_reply, etc.)
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label>Event</Label>
                          <Select
                            value={updateEvent}
                            onValueChange={setUpdateEvent}
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="sent">Sent</SelectItem>
                              <SelectItem value="replied">Replied</SelectItem>
                              <SelectItem value="no_reply">No Reply</SelectItem>
                              <SelectItem value="meeting_booked">
                                Meeting Booked
                              </SelectItem>
                              <SelectItem value="meeting_confirmed">
                                Meeting Confirmed
                              </SelectItem>
                              <SelectItem value="no_confirmation">
                                No Confirmation
                              </SelectItem>
                              <SelectItem value="meeting_done">
                                Meeting Done
                              </SelectItem>
                              <SelectItem value="drop">Drop</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="space-y-2">
                          <Label>Channel</Label>
                          <Select
                            value={updateChannel}
                            onValueChange={setUpdateChannel}
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="LinkedIn">LinkedIn</SelectItem>
                              <SelectItem value="Email">Email</SelectItem>
                              <SelectItem value="Call">Call</SelectItem>
                              <SelectItem value="Meeting">Meeting</SelectItem>
                              <SelectItem value="Note">Note</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      </div>

                      {updateEvent === 'replied' && (
                        <div className="space-y-2">
                          <Label>Sentiment</Label>
                          <Select
                            value={updateSentiment}
                            onValueChange={setUpdateSentiment}
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="positive">Positive</SelectItem>
                              <SelectItem value="neutral">Neutral</SelectItem>
                              <SelectItem value="negative">Negative</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                      )}

                      <div className="space-y-2">
                        <Label>Content (optional)</Label>
                        <Textarea
                          placeholder="Message content..."
                          value={updateContent}
                          onChange={(e) => setUpdateContent(e.target.value)}
                          rows={4}
                        />
                      </div>

                      <Button
                        onClick={() => updateOutreachMutation.mutate()}
                        disabled={updateOutreachMutation.isPending}
                        className="w-full"
                      >
                        {updateOutreachMutation.isPending ? (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                          <Send className="mr-2 h-4 w-4" />
                        )}
                        Update Outreach
                      </Button>

                      {updateOutreachMutation.data?.data && (
                        <div className="flex items-center justify-center gap-2 rounded-lg border bg-green-50 p-4">
                          <Badge
                            className={
                              stateColors[
                                updateOutreachMutation.data.data.previous_state
                              ]
                            }
                          >
                            {updateOutreachMutation.data.data.previous_state}
                          </Badge>
                          <ArrowRight className="h-4 w-4" />
                          <Badge
                            className={
                              stateColors[
                                updateOutreachMutation.data.data.new_state
                              ]
                            }
                          >
                            {updateOutreachMutation.data.data.new_state}
                          </Badge>
                          <span className="ml-2 text-sm text-muted-foreground">
                            Cadence step:{' '}
                            {nextStepLabels[
                              updateOutreachMutation.data.data
                                .next_step as NextStep
                            ] || updateOutreachMutation.data.data.next_step}
                          </span>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>

                {/* Meeting Tab */}
                <TabsContent value="meeting" className="space-y-4">
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <Calendar className="h-5 w-5" />
                        Create Meeting
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="space-y-2">
                        <Label>Title</Label>
                        <Input
                          placeholder="Meeting title..."
                          value={meetingTitle}
                          onChange={(e) => setMeetingTitle(e.target.value)}
                        />
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-2">
                          <Label>Time</Label>
                          <Input
                            type="datetime-local"
                            value={meetingTime}
                            onChange={(e) => setMeetingTime(e.target.value)}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>Duration (minutes)</Label>
                          <Input
                            type="number"
                            value={meetingDuration}
                            onChange={(e) =>
                              setMeetingDuration(parseInt(e.target.value))
                            }
                            min={15}
                            step={15}
                          />
                        </div>
                      </div>
                      <Button
                        onClick={() => createMeetingMutation.mutate()}
                        disabled={
                          createMeetingMutation.isPending || !meetingTime
                        }
                        className="w-full"
                      >
                        {createMeetingMutation.isPending ? (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                          <Calendar className="mr-2 h-4 w-4" />
                        )}
                        Create Meeting
                      </Button>
                    </CardContent>
                  </Card>

                  {/* Meetings List */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Meetings</CardTitle>
                    </CardHeader>
                    <CardContent>
                      {loadingMeetings ? (
                        <div className="flex items-center justify-center p-4">
                          <Loader2 className="h-6 w-6 animate-spin" />
                        </div>
                      ) : meetings?.data?.length === 0 ? (
                        <p className="p-4 text-center text-muted-foreground">
                          No meetings scheduled
                        </p>
                      ) : (
                        <div className="space-y-3">
                          {meetings?.data?.map((meeting: Meeting) => (
                            <div
                              key={meeting.id}
                              className="space-y-3 rounded-lg border p-4"
                            >
                              <div className="flex items-center justify-between">
                                <span className="font-medium">
                                  {meeting.title || 'Meeting'}
                                </span>
                                <Badge
                                  variant={
                                    meeting.status === 'completed'
                                      ? 'default'
                                      : meeting.status === 'cancelled'
                                        ? 'destructive'
                                        : 'outline'
                                  }
                                >
                                  {meeting.status}
                                </Badge>
                              </div>
                              <div className="flex items-center gap-4 text-sm text-muted-foreground">
                                <span className="flex items-center gap-1">
                                  <Clock className="h-3 w-3" />
                                  {new Date(meeting.time).toLocaleString()}
                                </span>
                                <span>{meeting.duration_minutes} min</span>
                                <span>{meeting.channel}</span>
                              </div>
                              {meeting.note && (
                                <p className="text-sm text-muted-foreground">
                                  {meeting.note}
                                </p>
                              )}

                              {/* Meeting Actions */}
                              <div className="flex flex-wrap items-center gap-2 border-t pt-2">
                                {/* Generate Meeting Prep - only show when scheduled and no prep yet */}
                                {meeting.status === 'scheduled' &&
                                  !meeting.meeting_prep && (
                                    <Button
                                      variant="outline"
                                      size="sm"
                                      className="border-amber-200 text-amber-600 hover:bg-amber-50"
                                      onClick={() =>
                                        generateMeetingPrepMutation.mutate(
                                          meeting.id
                                        )
                                      }
                                      disabled={
                                        generateMeetingPrepMutation.isPending
                                      }
                                    >
                                      {generateMeetingPrepMutation.isPending ? (
                                        <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                                      ) : (
                                        <Sparkles className="mr-1 h-4 w-4" />
                                      )}
                                      Generate Meeting Prep
                                    </Button>
                                  )}

                                {/* View Meeting Prep - only show when has prep */}
                                {meeting.meeting_prep && (
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    className="border-green-200 text-green-600 hover:bg-green-50"
                                    onClick={() => {
                                      setSelectedMeeting(meeting);
                                      setMeetingPrepDialogOpen(true);
                                    }}
                                  >
                                    <FileSearch className="mr-1 h-4 w-4" />
                                    View Meeting Prep
                                  </Button>
                                )}

                                {/* Mark as Completed - only show when scheduled */}
                                {meeting.status === 'scheduled' && (
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() =>
                                      updateMeetingMutation.mutate({
                                        meetingId: meeting.id,
                                        status: 'completed',
                                      })
                                    }
                                    disabled={updateMeetingMutation.isPending}
                                  >
                                    <CheckCircle className="mr-1 h-4 w-4" />
                                    Mark as Completed
                                  </Button>
                                )}

                                {/* Add/Edit Content - only show when completed */}
                                {meeting.status === 'completed' && (
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => {
                                      setSelectedMeeting(meeting);
                                      setMeetingContentInput(
                                        meeting.meeting_content || ''
                                      );
                                      setMeetingContentDialogOpen(true);
                                    }}
                                  >
                                    <FileText className="mr-1 h-4 w-4" />
                                    {meeting.meeting_content
                                      ? 'Edit Content'
                                      : 'Add Content'}
                                  </Button>
                                )}

                                {/* Delete Meeting */}
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="border-red-200 text-red-600 hover:bg-red-50"
                                  onClick={() => {
                                    if (
                                      confirm(
                                        'Are you sure you want to delete this meeting?'
                                      )
                                    ) {
                                      deleteMeetingMutation.mutate(meeting.id);
                                    }
                                  }}
                                  disabled={deleteMeetingMutation.isPending}
                                >
                                  {deleteMeetingMutation.isPending ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                  ) : (
                                    <Trash2 className="h-4 w-4" />
                                  )}
                                </Button>
                              </div>

                              {/* Show meeting content preview if exists */}
                              {meeting.meeting_content && (
                                <div className="rounded bg-muted/50 p-3 text-sm">
                                  <p className="mb-1 text-xs text-muted-foreground">
                                    Meeting Content:
                                  </p>
                                  <p className="line-clamp-2">
                                    {meeting.meeting_content}
                                  </p>
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                    </CardContent>
                  </Card>

                  {/* Meeting Content Dialog */}
                  <Dialog
                    open={meetingContentDialogOpen}
                    onOpenChange={setMeetingContentDialogOpen}
                  >
                    <DialogContent className="max-w-2xl">
                      <DialogHeader>
                        <DialogTitle>Meeting Content</DialogTitle>
                        <DialogDescription>
                          Enter meeting notes, content, or transcript for AI to
                          generate Meeting Brief
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4 py-4">
                        <Textarea
                          placeholder="Enter meeting content, notes, or transcript..."
                          value={meetingContentInput}
                          onChange={(e) =>
                            setMeetingContentInput(e.target.value)
                          }
                          className="min-h-[200px]"
                        />
                      </div>
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="outline"
                          onClick={() => setMeetingContentDialogOpen(false)}
                        >
                          Cancel
                        </Button>
                        <Button
                          onClick={() => {
                            if (selectedMeeting) {
                              updateMeetingMutation.mutate({
                                meetingId: selectedMeeting.id,
                                meeting_content: meetingContentInput,
                              });
                            }
                          }}
                          disabled={
                            !meetingContentInput ||
                            updateMeetingMutation.isPending
                          }
                        >
                          {updateMeetingMutation.isPending ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          ) : (
                            <FileText className="mr-2 h-4 w-4" />
                          )}
                          Save
                        </Button>
                      </div>
                    </DialogContent>
                  </Dialog>

                  {/* Meeting Prep Dialog */}
                  <Dialog
                    open={meetingPrepDialogOpen}
                    onOpenChange={setMeetingPrepDialogOpen}
                  >
                    <DialogContent className="max-h-[80vh] max-w-2xl overflow-y-auto">
                      <DialogHeader>
                        <DialogTitle>Meeting Prep</DialogTitle>
                        <DialogDescription>
                          Preparation document for meeting:{' '}
                          {selectedMeeting?.title || 'Meeting'} -{' '}
                          {selectedMeeting &&
                            new Date(selectedMeeting.time).toLocaleString()}
                        </DialogDescription>
                      </DialogHeader>
                      <div className="py-4">
                        <div className="prose prose-sm max-w-none whitespace-pre-wrap text-sm">
                          {selectedMeeting?.meeting_prep}
                        </div>
                      </div>
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="outline"
                          onClick={() => {
                            if (selectedMeeting?.meeting_prep) {
                              navigator.clipboard.writeText(
                                selectedMeeting.meeting_prep
                              );
                              toast.success('Meeting Prep copied to clipboard');
                            }
                          }}
                        >
                          <Copy className="mr-2 h-4 w-4" />
                          Copy
                        </Button>
                        <Button onClick={() => setMeetingPrepDialogOpen(false)}>
                          Close
                        </Button>
                      </div>
                    </DialogContent>
                  </Dialog>
                </TabsContent>

                {/* Feedback Tab */}
                <TabsContent value="feedback" className="space-y-4">
                  {/* Overall Stats */}
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center justify-between">
                        <span className="flex items-center gap-2">
                          <BarChart3 className="h-5 w-5" />
                          Feedback Statistics
                        </span>
                        <Button
                          variant="outline"
                          size="icon"
                          onClick={() => refetchFeedbackStats()}
                          disabled={loadingFeedbackStats}
                        >
                          <RefreshCw
                            className={`h-4 w-4 ${loadingFeedbackStats ? 'animate-spin' : ''}`}
                          />
                        </Button>
                      </CardTitle>
                      <CardDescription>
                        Track AI suggestions vs BD actions and outcomes
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      {loadingFeedbackStats ? (
                        <div className="flex items-center justify-center p-8">
                          <Loader2 className="h-8 w-8 animate-spin" />
                        </div>
                      ) : feedbackStats?.data ? (
                        <div className="space-y-6">
                          {/* Key Metrics */}
                          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                            <div className="rounded-lg border bg-blue-50 p-4">
                              <p className="text-xs text-muted-foreground">
                                Total Suggestions
                              </p>
                              <p className="text-2xl font-bold text-blue-600">
                                {feedbackStats.data.total_suggestions}
                              </p>
                            </div>
                            <div className="rounded-lg border bg-green-50 p-4">
                              <p className="text-xs text-muted-foreground">
                                Reply Rate
                              </p>
                              <p className="text-2xl font-bold text-green-600">
                                {feedbackStats.data.overall_reply_rate?.toFixed(
                                  1
                                ) || 0}
                                %
                              </p>
                            </div>
                            <div className="rounded-lg border bg-purple-50 p-4">
                              <p className="text-xs text-muted-foreground">
                                Meeting Rate
                              </p>
                              <p className="text-2xl font-bold text-purple-600">
                                {feedbackStats.data.overall_meeting_rate?.toFixed(
                                  1
                                ) || 0}
                                %
                              </p>
                            </div>
                            <div className="rounded-lg border bg-orange-50 p-4">
                              <p className="text-xs text-muted-foreground">
                                Draft Usage
                              </p>
                              <p className="text-2xl font-bold text-orange-600">
                                {feedbackStats.data.draft_usage_rate?.toFixed(
                                  1
                                ) || 0}
                                %
                              </p>
                            </div>
                          </div>

                          {/* Action Breakdown */}
                          <div>
                            <h4 className="mb-3 flex items-center gap-2 text-sm font-medium">
                              <Target className="h-4 w-4" />
                              Action Breakdown
                            </h4>
                            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                              <div className="rounded-lg border p-3">
                                <p className="text-xs text-muted-foreground">
                                  Used Draft
                                </p>
                                <p className="text-lg font-semibold">
                                  {feedbackStats.data.draft_used_count || 0}
                                </p>
                              </div>
                              <div className="rounded-lg border p-3">
                                <p className="text-xs text-muted-foreground">
                                  Modified Draft
                                </p>
                                <p className="text-lg font-semibold">
                                  {feedbackStats.data.draft_modified_count || 0}
                                </p>
                              </div>
                              <div className="rounded-lg border p-3">
                                <p className="text-xs text-muted-foreground">
                                  Wrote Own
                                </p>
                                <p className="text-lg font-semibold">
                                  {feedbackStats.data.wrote_own_count || 0}
                                </p>
                              </div>
                              <div className="rounded-lg border p-3">
                                <p className="text-xs text-muted-foreground">
                                  Skipped
                                </p>
                                <p className="text-lg font-semibold">
                                  {feedbackStats.data.skipped_count || 0}
                                </p>
                              </div>
                            </div>
                          </div>

                          {/* Outcome Stats */}
                          <div>
                            <h4 className="mb-3 flex items-center gap-2 text-sm font-medium">
                              <TrendingUp className="h-4 w-4" />
                              Outcomes
                            </h4>
                            <div className="grid grid-cols-3 gap-3">
                              <div className="rounded-lg border p-3">
                                <p className="text-xs text-muted-foreground">
                                  Sent
                                </p>
                                <p className="text-lg font-semibold">
                                  {feedbackStats.data.total_sent || 0}
                                </p>
                              </div>
                              <div className="rounded-lg border p-3">
                                <p className="text-xs text-muted-foreground">
                                  Replied
                                </p>
                                <p className="text-lg font-semibold text-green-600">
                                  {feedbackStats.data.total_replied || 0}
                                </p>
                              </div>
                              <div className="rounded-lg border p-3">
                                <p className="text-xs text-muted-foreground">
                                  Meetings
                                </p>
                                <p className="text-lg font-semibold text-purple-600">
                                  {feedbackStats.data.total_meetings || 0}
                                </p>
                              </div>
                            </div>
                          </div>

                          {/* Insights */}
                          {feedbackStats.data.insights &&
                            feedbackStats.data.insights.length > 0 && (
                              <div>
                                <h4 className="mb-3 flex items-center gap-2 text-sm font-medium">
                                  <Lightbulb className="h-4 w-4 text-yellow-500" />
                                  AI Insights
                                </h4>
                                <div className="space-y-2">
                                  {feedbackStats.data.insights.map(
                                    (insight: string, index: number) => (
                                      <div
                                        key={index}
                                        className="rounded-lg border bg-yellow-50 p-3 text-sm"
                                      >
                                        {insight}
                                      </div>
                                    )
                                  )}
                                </div>
                              </div>
                            )}
                        </div>
                      ) : (
                        <p className="p-4 text-center text-muted-foreground">
                          No feedback data available yet. Start generating
                          drafts and recording outcomes!
                        </p>
                      )}
                    </CardContent>
                  </Card>

                  {/* Scenario Performance */}
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <Target className="h-5 w-5" />
                        Scenario Performance
                      </CardTitle>
                      <CardDescription>
                        Performance breakdown by scenario type
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      {loadingScenarioStats ? (
                        <div className="flex items-center justify-center p-4">
                          <Loader2 className="h-6 w-6 animate-spin" />
                        </div>
                      ) : scenarioStats?.data &&
                        scenarioStats.data.length > 0 ? (
                        <div className="space-y-3">
                          {scenarioStats.data.map((stat: ScenarioStats) => (
                            <div
                              key={`${stat.scenario}-${stat.context_level}`}
                              className="rounded-lg border p-4"
                            >
                              <div className="mb-3 flex items-center justify-between">
                                <div>
                                  <span className="font-medium">
                                    {stat.scenario}
                                  </span>
                                  <Badge
                                    variant="outline"
                                    className="ml-2 text-xs"
                                  >
                                    {stat.context_level}
                                  </Badge>
                                </div>
                                <span className="text-sm text-muted-foreground">
                                  {stat.total_suggested} suggestions
                                </span>
                              </div>
                              <div className="grid grid-cols-4 gap-2 text-sm">
                                <div>
                                  <p className="text-xs text-muted-foreground">
                                    Reply Rate
                                  </p>
                                  <p className="font-medium text-green-600">
                                    {stat.reply_rate?.toFixed(1) || 0}%
                                  </p>
                                </div>
                                <div>
                                  <p className="text-xs text-muted-foreground">
                                    Meeting Rate
                                  </p>
                                  <p className="font-medium text-purple-600">
                                    {stat.meeting_rate?.toFixed(1) || 0}%
                                  </p>
                                </div>
                                <div>
                                  <p className="text-xs text-muted-foreground">
                                    Draft Usage
                                  </p>
                                  <p className="font-medium text-orange-600">
                                    {stat.draft_usage_rate?.toFixed(1) || 0}%
                                  </p>
                                </div>
                                <div>
                                  <p className="text-xs text-muted-foreground">
                                    Avg Reply Days
                                  </p>
                                  <p className="font-medium">
                                    {stat.avg_days_to_reply?.toFixed(1) || '-'}
                                  </p>
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="p-4 text-center text-muted-foreground">
                          No scenario data available yet
                        </p>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>
              </Tabs>
            ) : (
              <Card>
                <CardContent className="flex flex-col items-center justify-center p-12">
                  <Users className="mb-4 h-12 w-12 text-muted-foreground" />
                  <p className="text-lg font-medium">Select a Contact</p>
                  <p className="text-center text-muted-foreground">
                    Choose a contact from the suggestions or contact list to
                    view and manage outreach
                  </p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </ContentLayout>
  );
}
