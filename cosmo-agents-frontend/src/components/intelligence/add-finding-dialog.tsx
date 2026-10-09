'use client';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import IntelligenceApi, {
  type ResearchSuggestion,
} from '@/network/client/intelligence';
import { toast } from 'sonner';
import { Loader2 } from 'lucide-react';

interface AddFindingDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  contactId: string;
  suggestion: ResearchSuggestion;
  onSuccess?: () => void;
}

export function AddFindingDialog({
  open,
  onOpenChange,
  contactId,
  suggestion,
  onSuccess,
}: AddFindingDialogProps) {
  const [fieldName, setFieldName] = useState('');
  const [value, setValue] = useState('');
  const [source, setSource] = useState('');

  const addFindingMutation = useMutation({
    mutationFn: () =>
      IntelligenceApi.addResearchFinding(contactId, {
        category: suggestion.category,
        field_name: fieldName,
        value: value,
        source: source || undefined,
        priority: suggestion.priority,
        why_important: suggestion.why_important,
      }),
    onSuccess: () => {
      toast.success('Research finding added successfully!');
      onOpenChange(false);
      // Reset form
      setFieldName('');
      setValue('');
      setSource('');
      onSuccess?.();
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to add finding');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!fieldName || !value) {
      toast.error('Please fill in all required fields');
      return;
    }
    addFindingMutation.mutate();
  };

  // Suggest common field names based on category
  const getFieldSuggestions = (category: string) => {
    const suggestions: Record<string, string[]> = {
      'Company Intelligence': [
        'Company Size',
        'Funding Stage',
        'Revenue',
        'Employee Count',
        'Growth Rate',
      ],
      'Technical Stack': [
        'Tech Stack',
        'Platform',
        'Integration Tools',
        'Development Framework',
      ],
      'Competitive Landscape': [
        'Current Vendor',
        'Competitors',
        'Switching Cost',
        'Contract End Date',
      ],
      'Budget & Authority': [
        'Budget Range',
        'Decision Maker',
        'Approval Process',
        'Fiscal Year',
      ],
      'Timeline & Urgency': [
        'Implementation Timeline',
        'Project Start Date',
        'Urgency Level',
      ],
    };
    return suggestions[category] || ['Custom Field'];
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Add Research Finding</DialogTitle>
          <DialogDescription>
            Add your findings about this contact to help close the deal.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            {/* Category (readonly) */}
            <div className="space-y-2">
              <Label>Category</Label>
              <Input value={suggestion.category} disabled />
            </div>

            {/* Field Name */}
            <div className="space-y-2">
              <Label htmlFor="fieldName">
                Field Name <span className="text-red-500">*</span>
              </Label>
              <Select value={fieldName} onValueChange={setFieldName}>
                <SelectTrigger id="fieldName">
                  <SelectValue placeholder="Select or type custom field name..." />
                </SelectTrigger>
                <SelectContent>
                  {getFieldSuggestions(suggestion.category).map((field) => (
                    <SelectItem key={field} value={field}>
                      {field}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input
                placeholder="Or type custom field name..."
                value={fieldName}
                onChange={(e) => setFieldName(e.target.value)}
                className="mt-2"
              />
            </div>

            {/* Value */}
            <div className="space-y-2">
              <Label htmlFor="value">
                Value <span className="text-red-500">*</span>
              </Label>
              <Textarea
                id="value"
                placeholder="Enter the information you found..."
                value={value}
                onChange={(e) => setValue(e.target.value)}
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                {suggestion.why_important}
              </p>
            </div>

            {/* Source */}
            <div className="space-y-2">
              <Label htmlFor="source">Source (optional)</Label>
              <Input
                id="source"
                placeholder="e.g., LinkedIn, Crunchbase, company website..."
                value={source}
                onChange={(e) => setSource(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                🔍 Suggested: {suggestion.where_to_find}
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={addFindingMutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={addFindingMutation.isPending}>
              {addFindingMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save Finding
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
