'use client';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Search, Loader2, User } from 'lucide-react';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import IntelligenceApi, {
  type ContactSearchResult,
} from '@/network/client/intelligence';
import { toast } from 'sonner';
import { useRouter } from 'next/navigation';

interface VectorSearchDialogProps {
  trigger?: React.ReactNode;
}

export function VectorSearchDialog({ trigger }: VectorSearchDialogProps) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<ContactSearchResult[]>([]);
  const [isReembedPending, setIsReembedPending] = useState(false);

  const searchMutation = useMutation({
    mutationFn: () => IntelligenceApi.vectorSearch(query, 10, 1.0),
    onSuccess: (response) => {
      const sorted = [...response.data.results].sort(
        (a, b) => (b.similarity ?? 0) - (a.similarity ?? 0)
      );
      setResults(sorted);
      if (response.data.results.length === 0) {
        toast.info('No similar contacts found');
      } else {
        toast.success(`Found ${sorted.length} similar contacts`);
      }
    },
    onError: (error: any) => {
      toast.error(error.message || 'Search failed');
    },
  });

  const handleSearch = () => {
    if (!query.trim()) {
      toast.error('Please enter a search query');
      return;
    }
    searchMutation.mutate();
  };

  const handleReEmbedAll = async () => {
    try {
      setIsReembedPending(true);
      const resp = await IntelligenceApi.reEmbedAllContacts();
      toast.success(
        `Re-embed started for ${resp.data?.total_started ?? 0} contacts`
      );
    } catch (error: any) {
      toast.error(error.message || 'Failed to re-embed');
    } finally {
      setIsReembedPending(false);
    }
  };

  const handleContactClick = (contactId: string) => {
    router.push(`/all-prospects/${contactId}`);
    setOpen(false);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline">
            <Search className="mr-2 h-4 w-4" />
            AI Search
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Vector Semantic Search</DialogTitle>
          <DialogDescription>
            Search contacts using natural language and AI embeddings
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              onClick={handleReEmbedAll}
              disabled={isReembedPending}
            >
              {isReembedPending ? 'Re-embedding...' : 'Re-embed all contacts'}
            </Button>
          </div>

          {/* Search Input */}
          <div className="space-y-2">
            <Label htmlFor="search-query">Search Query</Label>
            <div className="flex gap-2">
              <Input
                id="search-query"
                placeholder="e.g., 'VP of Marketing at tech companies interested in automation'"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleSearch();
                  }
                }}
              />
              <Button
                onClick={handleSearch}
                disabled={searchMutation.isPending}
              >
                {searchMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Search className="h-4 w-4" />
                )}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Try: "Sales leaders at SaaS companies", "Decision makers in
              healthcare", "CTOs interested in AI"
            </p>
          </div>

          {/* Results */}
          {results.length > 0 && (
            <div className="space-y-2">
              <Label>Results ({results.length} contacts found)</Label>
              <div className="max-h-[400px] space-y-2 overflow-y-auto rounded-md border p-3">
                {results.map((result) => (
                  <div
                    key={result.contact_id}
                    className="flex cursor-pointer items-center justify-between rounded-lg border p-3 transition-colors hover:bg-accent"
                    onClick={() => handleContactClick(result.contact_id)}
                  >
                    <div className="flex items-start gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                        <User className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <p className="font-medium">
                          {result.first_name || ''} {result.last_name || ''}
                        </p>
                        {result.job_title && (
                          <p className="text-sm text-muted-foreground">
                            {result.job_title}
                          </p>
                        )}
                        {result.company && (
                          <p className="text-sm text-muted-foreground">
                            {result.company}
                          </p>
                        )}
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <Badge
                        variant={
                          result.similarity >= 0.8
                            ? 'default'
                            : result.similarity >= 0.7
                              ? 'secondary'
                              : 'outline'
                        }
                      >
                        {(result.similarity * 100).toFixed(1)}%
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        Match Score
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Empty State */}
          {searchMutation.isSuccess && results.length === 0 && (
            <div className="rounded-lg border border-dashed p-8 text-center">
              <Search className="mx-auto h-12 w-12 text-muted-foreground" />
              <p className="mt-4 font-medium">No contacts found</p>
              <p className="text-sm text-muted-foreground">
                Try a different search query or ensure contacts have been
                enriched with AI
              </p>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
