'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Send, MessageSquare, Loader2, Pencil, Trash2, Check, X } from 'lucide-react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { ScrollArea } from '@/components/ui/scroll-area';
import OutreachApi, { type InteractionLog } from '@/network/client/outreach';

interface ContactNotesProps {
  contactId: string;
}

export function ContactNotes({ contactId }: ContactNotesProps) {
  const queryClient = useQueryClient();
  const [newNote, setNewNote] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editContent, setEditContent] = useState('');

  // Fetch notes
  const { data: notesResponse, isLoading } = useQuery({
    queryKey: ['contact-notes', contactId],
    queryFn: () => OutreachApi.getNotes(contactId, 50),
    staleTime: 30_000,
  });

  const notes = notesResponse?.data || [];

  // Add note mutation
  const addNoteMutation = useMutation({
    mutationFn: (content: string) => OutreachApi.addNote(contactId, content),
    onSuccess: () => {
      toast.success('Note added');
      setNewNote('');
      queryClient.invalidateQueries({ queryKey: ['contact-notes', contactId] });
    },
    onError: () => {
      toast.error('Failed to add note');
    },
  });

  // Update note mutation
  const updateNoteMutation = useMutation({
    mutationFn: ({ noteId, content }: { noteId: string; content: string }) =>
      OutreachApi.updateNote(contactId, noteId, content),
    onSuccess: () => {
      toast.success('Note updated');
      setEditingId(null);
      setEditContent('');
      queryClient.invalidateQueries({ queryKey: ['contact-notes', contactId] });
    },
    onError: () => {
      toast.error('Failed to update note');
    },
  });

  // Delete note mutation
  const deleteNoteMutation = useMutation({
    mutationFn: (noteId: string) => OutreachApi.deleteNote(contactId, noteId),
    onSuccess: () => {
      toast.success('Note deleted');
      queryClient.invalidateQueries({ queryKey: ['contact-notes', contactId] });
    },
    onError: () => {
      toast.error('Failed to delete note');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newNote.trim()) return;
    addNoteMutation.mutate(newNote.trim());
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const handleEdit = (note: InteractionLog) => {
    setEditingId(note.id);
    setEditContent(note.content);
  };

  const handleSaveEdit = () => {
    if (!editingId || !editContent.trim()) return;
    updateNoteMutation.mutate({ noteId: editingId, content: editContent.trim() });
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setEditContent('');
  };

  const handleDelete = (noteId: string) => {
    if (confirm('Delete this note?')) {
      deleteNoteMutation.mutate(noteId);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 mb-3">
        <MessageSquare className="h-4 w-4 text-gray-500" />
        <h3 className="font-semibold text-gray-900">Team Notes</h3>
        <span className="text-xs text-gray-500">({notes.length})</span>
      </div>

      {/* Notes List */}
      <ScrollArea className="flex-1 max-h-[300px] mb-3">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
          </div>
        ) : notes.length === 0 ? (
          <div className="text-center py-8 text-gray-500 text-sm">
            No notes yet. Add one below.
          </div>
        ) : (
          <div className="space-y-3 pr-3">
            {notes.map((note: InteractionLog) => (
              <div
                key={note.id}
                className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm group"
              >
                {editingId === note.id ? (
                  <>
                    <Textarea
                      value={editContent}
                      onChange={(e) => setEditContent(e.target.value)}
                      className="min-h-[60px] text-sm mb-2"
                      autoFocus
                    />
                    <div className="flex justify-end gap-1">
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-7 w-7"
                        onClick={handleCancelEdit}
                      >
                        <X className="h-3 w-3" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-7 w-7 text-green-600"
                        onClick={handleSaveEdit}
                        disabled={updateNoteMutation.isPending}
                      >
                        {updateNoteMutation.isPending ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <Check className="h-3 w-3" />
                        )}
                      </Button>
                    </div>
                  </>
                ) : (
                  <>
                    <div className="flex justify-between items-start">
                      <p className="text-sm text-gray-700 whitespace-pre-wrap break-words flex-1">
                        {note.content}
                      </p>
                      <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity ml-2">
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-6 w-6"
                          onClick={() => handleEdit(note)}
                        >
                          <Pencil className="h-3 w-3" />
                        </Button>
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-6 w-6 text-red-500"
                          onClick={() => handleDelete(note.id)}
                          disabled={deleteNoteMutation.isPending}
                        >
                          <Trash2 className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                    <p className="text-xs text-gray-400 mt-2">
                      {format(new Date(note.timestamp), 'dd/MM/yyyy HH:mm')}
                    </p>
                  </>
                )}
              </div>
            ))}
          </div>
        )}
      </ScrollArea>

      {/* Add Note Form */}
      <form onSubmit={handleSubmit} className="flex gap-2">
        <Textarea
          value={newNote}
          onChange={(e) => setNewNote(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Add a note... (Enter to send, Shift+Enter for new line)"
          className="min-h-[60px] resize-none text-sm"
          disabled={addNoteMutation.isPending}
        />
        <Button
          type="submit"
          size="icon"
          disabled={!newNote.trim() || addNoteMutation.isPending}
          className="shrink-0"
        >
          {addNoteMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Send className="h-4 w-4" />
          )}
        </Button>
      </form>
    </div>
  );
}
