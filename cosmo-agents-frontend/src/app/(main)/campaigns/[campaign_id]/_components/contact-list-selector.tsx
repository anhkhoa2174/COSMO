import { AddButton } from '@/components/buttons/add-button';
import { Card, CardContent } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { Spinner } from '@/components/ui/spinner';
import { Switch } from '@/components/ui/switch';
import { Campaign } from '@/models/campaign';
import CampaignApi from '@/network/client/campaign';
import { contactListApi } from '@/network/client/contact-list';
import { useMutation, useQuery } from '@tanstack/react-query';
import { usePathname, useRouter } from 'next/navigation';
import { useState } from 'react';
import { toast } from 'sonner';
import { useCampaign, useCampaignSupport } from '../use-campaign';
import { SheetHeader } from './sheet-header';

interface ContactListSelectorProps {
  closeSheet: () => void;
  onSuccess: () => void;
}

export default function ContactListSelector({
  closeSheet,
  onSuccess,
}: ContactListSelectorProps) {
  const router = useRouter();
  const pathname = usePathname();

  const [campaign, setCampaign] = useCampaign();
  const [, setCampaignSupport] = useCampaignSupport();
  const [contactListId, setContactListId] = useState(campaign.list_contact_id);

  const { data, isLoading } = useQuery({
    queryKey: ['contactLists'],
    queryFn: () => contactListApi.search({ filter_: {} }),
  });
  const updateCampaignMutation = useMutation({
    mutationFn: () =>
      CampaignApi.update(campaign.id, {
        list_contact_id: contactListId as string,
      }),
    onSuccess: () => {
      setCampaign(
        (prev) => ({ ...prev, list_contact_id: contactListId }) as Campaign
      );
      setCampaignSupport((prev) => ({ ...prev, templates: {} }));
      onSuccess();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Failed to update campaign'
      );
    },
  });

  const hasChanged = contactListId !== campaign.list_contact_id;

  const handleSaveAndClose = async () => {
    if (hasChanged) {
      await updateCampaignMutation.mutateAsync();
    }
    closeSheet();
  };

  const leftSection = (
    <AddButton
      text="New"
      onClick={() =>
        router.push(`/contact-lists?next=${encodeURIComponent(pathname)}`)
      }
      variant="outline"
    />
  );

  return (
    <div className="flex h-full flex-col">
      <SheetHeader
        title="Entry Rules"
        hasChanged={hasChanged}
        onSubmit={handleSaveAndClose}
        onClose={closeSheet}
        leftSection={leftSection}
      />
      <Separator />
      <div className="h-96 flex-grow">
        <Spinner
          show={updateCampaignMutation.isPending}
          withOverlay
          label="Saving..."
        />
        <ScrollArea className="h-full p-4" type="always">
          {isLoading ? (
            <div className="flex flex-col space-y-4">
              <Skeleton className="h-[50px] rounded-md" />
              <Skeleton className="h-[50px] rounded-md" />
              <Skeleton className="h-[50px] rounded-md" />
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              {campaign.status !== 'draft' && (
                <p className="mb-4 text-orange-500">
                  You can't change the contact list for a campaign that is not
                  in draft mode
                </p>
              )}
              {data?.data.list.length === 0 && (
                <p className="text-center">No contact list found</p>
              )}
              {data?.data.list
                .filter((l) => l.size > 0)
                .map(({ entity: contactList, size }) => (
                  <Card key={contactList.id}>
                    <CardContent className="pt-4">
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-4">
                          <div className="max-w-[200px]">
                            <p className="truncate text-base font-semibold">
                              [{contactList.source}] {contactList.name}
                            </p>
                            <p className="truncate text-muted-foreground">
                              {size} contacts
                            </p>
                          </div>
                        </div>
                        <Switch
                          checked={contactListId === contactList.id}
                          onCheckedChange={() =>
                            setContactListId(contactList.id)
                          }
                          disabled={campaign.status !== 'draft'}
                        />
                      </div>
                    </CardContent>
                  </Card>
                ))}
            </div>
          )}
        </ScrollArea>
      </div>
    </div>
  );
}
