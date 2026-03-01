import { importHubspot } from '@/network/client/contact';
import { HubspotApi } from '@/network/client/hubspot';
import { useOperationFlow } from '@/network/client/template';
import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import Link from 'next/link';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Alert, AlertDescription, AlertTitle } from '../ui/alert';
import { Info } from 'lucide-react';
import { MainButton } from '../buttons/main-button';

export function ImportFromHubSpotDialog({
  onSuccess,
  open,
  onOpenChange,
  children,
}: {
  onSuccess?: () => void;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  children?: React.ReactNode;
}) {
  const router = useRouter();
  const { data: hubspotInfo } = useQuery({
    queryKey: ['hubspot-me'],
    queryFn: () => HubspotApi.getMe(),
  })
  const accessToken = hubspotInfo?.data?.access_token;
  const refreshToken = hubspotInfo?.data?.refresh_token;

  const [listId, setListId] = useState<string>();
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ['hubspot-lists'],
    queryFn: () => HubspotApi.searchLists(accessToken, refreshToken),
    enabled: !!accessToken && !!refreshToken,
  });

  const importMutation = useOperationFlow(
    () => importHubspot({ list_ids: [listId!] }),
    {
      onSuccess: () => {
        onSuccess?.();
        setIsDialogOpen(false);
      },
      onError: (error) => {
        toast.error(error.message);
      },
    }
  );

  const [searchTerm, setSearchTerm] = useState('');
  const onSearch = (search: string) => {
    setSearchTerm(search.toLowerCase());
  };
  const filteredData =
    data?.lists.filter((list) =>
      list.name.toLowerCase().includes(searchTerm)
    ) || [];

  return (
    <Dialog
      {...(children
        ? { open: isDialogOpen, onOpenChange: setIsDialogOpen }
        : { open, onOpenChange })}
    >
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Choose list from HubSpot</DialogTitle>
          <DialogDescription>
            Import contact list from HubSpot to use in your campaigns
          </DialogDescription>
        </DialogHeader>
        {accessToken ? (
          <div className="space-y-4">
            <p className="text-muted-foreground">
              You can{' '}
              <Link
                href="/hubspot-integration"
                className="text-orange-500"
              >
                go here
              </Link>{' '}
              to modify your HubSpot account or mapping parameters
            </p>
            <Input
              placeholder="Search HubSpot lists ..."
              onChange={(e) => onSearch(e.target.value)}
            />
            {error && <p className="text-destructive">{error?.message}</p>}
            {isLoading ? (
              <div className="italic">HubSpot lists loading...</div>
            ) : data ? (
              <ScrollArea
                className="h-[300px] gap-0 rounded border-b border-l border-r"
                type="always"
              >
                <RadioGroup
                  className="gap-0 space-y-0"
                  value={listId}
                  onValueChange={setListId}
                >
                  {filteredData.map((list) => (
                    <div
                      key={list.listId}
                      className="flex items-center gap-4 border-t p-4"
                    >
                      <RadioGroupItem value={list.listId} />
                      <div>
                        <p className="font-medium">{list.name}</p>
                        <p className="text-muted-foreground">
                          {list.additionalProperties?.hs_list_size} contacts
                        </p>
                      </div>
                    </div>
                  ))}
                </RadioGroup>
              </ScrollArea>
            ) : null}
          </div>
        ) : (
          <Alert>
            <Info className="h-4 w-4" />
            <AlertTitle>Required Permissions</AlertTitle>
            <AlertDescription>
              This integration needs access to contacts and lists in your
              HubSpot account.
            </AlertDescription>
          </Alert>
        )}
        <DialogFooter>
          <MainButton
            variant="outline"
            onClick={() => onOpenChange?.(false) || setIsDialogOpen(false)}
            text="Cancel"
          />
          {accessToken ? (
            <MainButton
              onClick={() => importMutation.mutate()}
              loading={importMutation.isLoading}
              text="Import"
              disabled={!listId}
            />
          ) : (
            <MainButton
              onClick={() => router.push('/hubspot-integration')}
              text="Authorize"
            />
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
