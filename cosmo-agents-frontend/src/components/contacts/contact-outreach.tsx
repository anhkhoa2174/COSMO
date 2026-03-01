'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  Sparkles,
  Copy,
  Check,
  Loader2,
  RefreshCw,
  MessageSquarePlus,
  Send,
  MessageCircle,
  Calendar,
  XCircle,
  Plus,
  Clock,
  FileText,
  FileSearch,
} from 'lucide-react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import OutreachApi, { type GenerateDraftResponse, type ConversationState, type Meeting } from '@/network/client/outreach';

interface ContactOutreachProps {
  contactId: string;
}

const stateColors: Record<string, string> = {
  COLD: 'bg-blue-100 text-blue-800',
  NO_REPLY: 'bg-yellow-100 text-yellow-800',
  REPLIED: 'bg-green-100 text-green-800',
  POST_MEETING: 'bg-purple-100 text-purple-800',
  DROPPED: 'bg-gray-100 text-gray-800',
};

const nextStepLabels: Record<string, string> = {
  SEND: 'Send Message',
  FOLLOW_UP: 'Follow Up',
  SET_MEETING: 'Set Meeting',
  WAIT: 'Wait',
  DROP: 'Drop',
};

export function ContactOutreach({ contactId }: ContactOutreachProps) {
  const queryClient = useQueryClient();
  const [copied, setCopied] = useState(false);
  const [editedDraft, setEditedDraft] = useState('');
  const [meetingDialogOpen, setMeetingDialogOpen] = useState(false);
  const [meetingTitle, setMeetingTitle] = useState('');
  const [meetingTime, setMeetingTime] = useState('');
  const [meetingDuration, setMeetingDuration] = useState(30);
  const [meetingChannel, setMeetingChannel] = useState('Zoom');

  // Meeting prep states
  const [meetingContentDialogOpen, setMeetingContentDialogOpen] = useState(false);
  const [selectedMeeting, setSelectedMeeting] = useState<Meeting | null>(null);
  const [meetingContent, setMeetingContent] = useState('');
  const [meetingPrepDialogOpen, setMeetingPrepDialogOpen] = useState(false);

  // Generate draft
  const { data: draftResponse, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['contact-draft', contactId],
    queryFn: () => OutreachApi.generateDraft(contactId),
    staleTime: 60_000,
    enabled: false, // Don't auto-fetch
  });

  const draft = draftResponse?.data;
  const currentState = draft?.state?.conversation_state;

  // Fetch meetings for this contact
  const { data: meetingsResponse } = useQuery({
    queryKey: ['contact-meetings', contactId],
    queryFn: () => OutreachApi.getMeetings(contactId),
    staleTime: 30_000,
  });

  const meetings = meetingsResponse?.data || [];

  // Update outreach state mutation
  const updateStateMutation = useMutation({
    mutationFn: (event: 'sent' | 'replied' | 'no_reply' | 'meeting_booked' | 'meeting_confirmed' | 'no_confirmation' | 'meeting_done' | 'drop') =>
      OutreachApi.updateOutreach(contactId, event),
    onSuccess: (response) => {
      const data = response.data;
      toast.success(`State updated: ${data?.previous_state} → ${data?.new_state}`);
      // Invalidate and refetch draft to get new state
      queryClient.invalidateQueries({ queryKey: ['contact-draft', contactId] });
      refetch();
    },
    onError: () => {
      toast.error('Failed to update state');
    },
  });

  // Create meeting mutation
  const createMeetingMutation = useMutation({
    mutationFn: () => {
      const meetingTimeISO = meetingTime ? new Date(meetingTime).toISOString() : '';
      return OutreachApi.createMeeting({
        contact_id: contactId,
        title: meetingTitle || 'Meeting',
        time: meetingTimeISO,
        duration_minutes: meetingDuration,
        channel: meetingChannel,
      });
    },
    onSuccess: () => {
      toast.success('Meeting created!');
      setMeetingDialogOpen(false);
      setMeetingTitle('');
      setMeetingTime('');
      queryClient.invalidateQueries({ queryKey: ['contact-meetings', contactId] });
      queryClient.invalidateQueries({ queryKey: ['contact-draft', contactId] });
      refetch();
    },
    onError: () => {
      toast.error('Failed to create meeting');
    },
  });

  // Update meeting content mutation
  const updateMeetingContentMutation = useMutation({
    mutationFn: (data: { meetingId: string; content: string }) =>
      OutreachApi.updateMeeting(data.meetingId, { meeting_content: data.content }),
    onSuccess: () => {
      toast.success('Nội dung cuộc họp đã được lưu!');
      setMeetingContentDialogOpen(false);
      setMeetingContent('');
      setSelectedMeeting(null);
      queryClient.invalidateQueries({ queryKey: ['contact-meetings', contactId] });
    },
    onError: () => {
      toast.error('Lưu nội dung thất bại');
    },
  });

  // Generate meeting prep mutation
  const generateMeetingPrepMutation = useMutation({
    mutationFn: (meetingId: string) => OutreachApi.generateMeetingPrep(meetingId),
    onSuccess: () => {
      toast.success('Meeting Prep đã được tạo!');
      queryClient.invalidateQueries({ queryKey: ['contact-meetings', contactId] });
    },
    onError: () => {
      toast.error('Tạo Meeting Prep thất bại');
    },
  });

  // Copy to clipboard
  const handleCopy = async () => {
    const textToCopy = editedDraft || draft?.draft || '';
    await navigator.clipboard.writeText(textToCopy);
    setCopied(true);
    toast.success('Copied to clipboard');
    setTimeout(() => setCopied(false), 2000);
  };

  // Generate draft
  const handleGenerate = () => {
    setEditedDraft('');
    refetch();
  };

  // Update edited draft when new draft is received
  const currentDraft = editedDraft || draft?.draft || '';

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center justify-between text-base">
          <div className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-amber-500" />
            <span>Message Draft</span>
          </div>
          {draft?.state && (
            <div className="flex items-center gap-2">
              <Badge className={stateColors[draft.state.conversation_state] || stateColors.COLD}>
                {draft.state.conversation_state}
              </Badge>
              <Badge variant="outline">
                {nextStepLabels[draft.state.next_step] || draft.state.next_step}
              </Badge>
            </div>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Notes Context */}
        {draft?.notes && draft.notes.length > 0 && (
          <div className="rounded-lg bg-amber-50 border border-amber-200 p-3">
            <p className="text-xs font-medium text-amber-800 mb-2">
              Context from team notes:
            </p>
            <div className="space-y-1">
              {draft.notes.slice(0, 3).map((note) => (
                <p key={note.id} className="text-xs text-amber-700 truncate">
                  {format(new Date(note.timestamp), 'dd/MM')} - {note.content.slice(0, 100)}
                  {note.content.length > 100 && '...'}
                </p>
              ))}
            </div>
          </div>
        )}

        {/* Draft Content */}
        {!draft && !isLoading && !isFetching ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <MessageSquarePlus className="h-10 w-10 text-gray-300 mb-3" />
            <p className="text-sm text-gray-500 mb-4">
              Generate a personalized message draft based on contact info and team notes
            </p>
            <Button onClick={handleGenerate} disabled={isLoading}>
              <Sparkles className="h-4 w-4 mr-2" />
              Generate Draft
            </Button>
          </div>
        ) : isLoading || isFetching ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
            <span className="ml-2 text-sm text-gray-500">Generating...</span>
          </div>
        ) : (
          <>
            <Textarea
              value={currentDraft}
              onChange={(e) => setEditedDraft(e.target.value)}
              className="min-h-[150px] text-sm"
              placeholder="Message draft will appear here..."
            />

            {/* Scenario Info */}
            {draft && (
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <span>Scenario: {draft.scenario}</span>
                <span>|</span>
                <span>Context: {draft.context_level}</span>
              </div>
            )}

            {/* Actions */}
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleGenerate}
                disabled={isFetching}
              >
                {isFetching ? (
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                ) : (
                  <RefreshCw className="h-4 w-4 mr-2" />
                )}
                Regenerate
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopy}
                disabled={!currentDraft}
              >
                {copied ? (
                  <Check className="h-4 w-4 mr-2" />
                ) : (
                  <Copy className="h-4 w-4 mr-2" />
                )}
                {copied ? 'Copied!' : 'Copy'}
              </Button>
            </div>

            {/* State Triggers */}
            <Separator className="my-3" />
            <div className="space-y-2">
              <p className="text-xs font-medium text-gray-500">Update Status:</p>
              <div className="flex flex-wrap gap-2">
                {/* Mark as Sent - only show when COLD or NO_REPLY */}
                {(currentState === 'COLD' || currentState === 'NO_REPLY') && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => updateStateMutation.mutate('sent')}
                    disabled={updateStateMutation.isPending}
                    className="text-blue-600 border-blue-200 hover:bg-blue-50"
                  >
                    {updateStateMutation.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin mr-1" />
                    ) : (
                      <Send className="h-3 w-3 mr-1" />
                    )}
                    Sent
                  </Button>
                )}

                {/* Mark as Replied - show when NO_REPLY */}
                {currentState === 'NO_REPLY' && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => updateStateMutation.mutate('replied')}
                    disabled={updateStateMutation.isPending}
                    className="text-green-600 border-green-200 hover:bg-green-50"
                  >
                    {updateStateMutation.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin mr-1" />
                    ) : (
                      <MessageCircle className="h-3 w-3 mr-1" />
                    )}
                    Replied
                  </Button>
                )}

                {/* Mark Meeting Booked - show when REPLIED */}
                {currentState === 'REPLIED' && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => updateStateMutation.mutate('meeting_booked')}
                    disabled={updateStateMutation.isPending}
                    className="text-purple-600 border-purple-200 hover:bg-purple-50"
                  >
                    {updateStateMutation.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin mr-1" />
                    ) : (
                      <Calendar className="h-3 w-3 mr-1" />
                    )}
                    Meeting Booked
                  </Button>
                )}

                {/* Mark as Dropped - always available except DROPPED */}
                {currentState !== 'DROPPED' && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      if (confirm('Drop this contact from outreach?')) {
                        updateStateMutation.mutate('drop');
                      }
                    }}
                    disabled={updateStateMutation.isPending}
                    className="text-gray-500 border-gray-200 hover:bg-gray-50"
                  >
                    {updateStateMutation.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin mr-1" />
                    ) : (
                      <XCircle className="h-3 w-3 mr-1" />
                    )}
                    Drop
                  </Button>
                )}
              </div>
            </div>

            {/* Meetings Section */}
            <Separator className="my-3" />
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium text-gray-500">Meetings ({meetings.length})</p>
                <Dialog open={meetingDialogOpen} onOpenChange={setMeetingDialogOpen}>
                  <DialogTrigger asChild>
                    <Button variant="outline" size="sm" className="h-7 text-xs">
                      <Plus className="h-3 w-3 mr-1" />
                      New Meeting
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>Schedule Meeting</DialogTitle>
                      <DialogDescription>
                        Create a new meeting with this contact
                      </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
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
                          <Label>Date & Time</Label>
                          <Input
                            type="datetime-local"
                            value={meetingTime}
                            onChange={(e) => setMeetingTime(e.target.value)}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>Duration (min)</Label>
                          <Input
                            type="number"
                            value={meetingDuration}
                            onChange={(e) => setMeetingDuration(parseInt(e.target.value) || 30)}
                            min={15}
                            step={15}
                          />
                        </div>
                      </div>
                      <div className="space-y-2">
                        <Label>Channel</Label>
                        <Select value={meetingChannel} onValueChange={setMeetingChannel}>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="Zoom">Zoom</SelectItem>
                            <SelectItem value="Google Meet">Google Meet</SelectItem>
                            <SelectItem value="Teams">Microsoft Teams</SelectItem>
                            <SelectItem value="Phone">Phone Call</SelectItem>
                            <SelectItem value="In-person">In-person</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    <div className="flex justify-end gap-2">
                      <Button variant="outline" onClick={() => setMeetingDialogOpen(false)}>
                        Cancel
                      </Button>
                      <Button
                        onClick={() => createMeetingMutation.mutate()}
                        disabled={!meetingTime || createMeetingMutation.isPending}
                      >
                        {createMeetingMutation.isPending ? (
                          <Loader2 className="h-4 w-4 animate-spin mr-2" />
                        ) : (
                          <Calendar className="h-4 w-4 mr-2" />
                        )}
                        Create
                      </Button>
                    </div>
                  </DialogContent>
                </Dialog>
              </div>

              {/* Meetings List */}
              {meetings.length > 0 && (
                <div className="space-y-2 max-h-[200px] overflow-y-auto">
                  {meetings.map((meeting: Meeting) => (
                    <div
                      key={meeting.id}
                      className="p-2 rounded border bg-white text-xs space-y-2"
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Calendar className="h-3 w-3 text-purple-500" />
                          <span className="font-medium">{meeting.title || 'Meeting'}</span>
                        </div>
                        <div className="flex items-center gap-2 text-gray-500">
                          <Clock className="h-3 w-3" />
                          <span>{format(new Date(meeting.time), 'dd/MM HH:mm')}</span>
                          <Badge
                            variant={meeting.status === 'completed' ? 'default' : 'outline'}
                            className="text-[10px] h-5"
                          >
                            {meeting.status}
                          </Badge>
                        </div>
                      </div>

                      {/* Meeting Prep Actions - show for scheduled meetings */}
                      {meeting.status === 'scheduled' && (
                        <div className="flex items-center gap-2 pt-1 border-t">
                          {/* Generate Prep Button - only show when no prep yet */}
                          {!meeting.meeting_prep && (
                            <Button
                              variant="outline"
                              size="sm"
                              className="h-6 text-[10px] text-amber-600 border-amber-200 hover:bg-amber-50"
                              onClick={() => generateMeetingPrepMutation.mutate(meeting.id)}
                              disabled={generateMeetingPrepMutation.isPending}
                            >
                              {generateMeetingPrepMutation.isPending ? (
                                <Loader2 className="h-3 w-3 animate-spin mr-1" />
                              ) : (
                                <Sparkles className="h-3 w-3 mr-1" />
                              )}
                              Tạo Meeting Prep
                            </Button>
                          )}

                          {/* View Prep Button - only show when has prep */}
                          {meeting.meeting_prep && (
                            <Button
                              variant="outline"
                              size="sm"
                              className="h-6 text-[10px] text-green-600 border-green-200 hover:bg-green-50"
                              onClick={() => {
                                setSelectedMeeting(meeting);
                                setMeetingPrepDialogOpen(true);
                              }}
                            >
                              <FileSearch className="h-3 w-3 mr-1" />
                              Xem Prep
                            </Button>
                          )}
                        </div>
                      )}

                      {/* Meeting Content Actions - show for completed meetings */}
                      {meeting.status === 'completed' && (
                        <div className="flex items-center gap-2 pt-1 border-t">
                          {/* View Prep Button - only show when has prep */}
                          {meeting.meeting_prep && (
                            <Button
                              variant="outline"
                              size="sm"
                              className="h-6 text-[10px] text-green-600 border-green-200 hover:bg-green-50"
                              onClick={() => {
                                setSelectedMeeting(meeting);
                                setMeetingPrepDialogOpen(true);
                              }}
                            >
                              <FileSearch className="h-3 w-3 mr-1" />
                              Xem Prep
                            </Button>
                          )}

                          {/* Add/Edit Content Button */}
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-6 text-[10px]"
                            onClick={() => {
                              setSelectedMeeting(meeting);
                              setMeetingContent(meeting.meeting_content || '');
                              setMeetingContentDialogOpen(true);
                            }}
                          >
                            <FileText className="h-3 w-3 mr-1" />
                            {meeting.meeting_content ? 'Sửa nội dung' : 'Thêm nội dung'}
                          </Button>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {/* Meeting Content Dialog */}
              <Dialog open={meetingContentDialogOpen} onOpenChange={setMeetingContentDialogOpen}>
                <DialogContent className="max-w-2xl">
                  <DialogHeader>
                    <DialogTitle>Nội dung cuộc họp</DialogTitle>
                    <DialogDescription>
                      Nhập nội dung, ghi chú hoặc transcript của cuộc họp
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    <div className="space-y-2">
                      <Label>Nội dung cuộc họp</Label>
                      <Textarea
                        placeholder="Nhập nội dung cuộc họp, ghi chú hoặc transcript..."
                        value={meetingContent}
                        onChange={(e) => setMeetingContent(e.target.value)}
                        className="min-h-[200px]"
                      />
                    </div>
                  </div>
                  <div className="flex justify-end gap-2">
                    <Button variant="outline" onClick={() => setMeetingContentDialogOpen(false)}>
                      Hủy
                    </Button>
                    <Button
                      onClick={() => {
                        if (selectedMeeting) {
                          updateMeetingContentMutation.mutate({
                            meetingId: selectedMeeting.id,
                            content: meetingContent,
                          });
                        }
                      }}
                      disabled={!meetingContent || updateMeetingContentMutation.isPending}
                    >
                      {updateMeetingContentMutation.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      ) : (
                        <FileText className="h-4 w-4 mr-2" />
                      )}
                      Lưu
                    </Button>
                  </div>
                </DialogContent>
              </Dialog>

              {/* Meeting Prep Dialog */}
              <Dialog open={meetingPrepDialogOpen} onOpenChange={setMeetingPrepDialogOpen}>
                <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
                  <DialogHeader>
                    <DialogTitle>Meeting Prep</DialogTitle>
                    <DialogDescription>
                      Tài liệu chuẩn bị: {selectedMeeting?.title || 'Meeting'} - {selectedMeeting && format(new Date(selectedMeeting.time), 'dd/MM/yyyy HH:mm')}
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
                          navigator.clipboard.writeText(selectedMeeting.meeting_prep);
                          toast.success('Đã copy Meeting Prep');
                        }
                      }}
                    >
                      <Copy className="h-4 w-4 mr-2" />
                      Copy
                    </Button>
                    <Button onClick={() => setMeetingPrepDialogOpen(false)}>
                      Đóng
                    </Button>
                  </div>
                </DialogContent>
              </Dialog>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
