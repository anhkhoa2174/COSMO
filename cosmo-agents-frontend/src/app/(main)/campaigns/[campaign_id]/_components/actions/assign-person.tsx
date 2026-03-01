import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Switch } from '@/components/ui/switch';
import { AddButton } from '@/components/buttons/add-button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Spinner } from '@/components/ui/spinner';
import { CampaignConfig } from '@/models/campaign';
import type { EmailIntent } from '@/models/email';
import CampaignApi from '@/network/client/campaign';
import OrganizationApi from '@/network/client/organization';
import { useCampaign, useCampaignSupport } from '../../use-campaign';
import { SheetHeader } from '../sheet-header';
import { Skeleton } from '@/components/ui/skeleton';

type ConfigPayload = {
  user_id: string;
};

interface AssignPersonProps {
  intentType: EmailIntent;
  closeSheet: () => void;
  onSuccess: () => void;
}

export default function AssignPerson({
  intentType,
  closeSheet,
  onSuccess,
}: AssignPersonProps) {
  const router = useRouter();

  const [campaign, setCampaign] = useCampaign();
  const existOption = campaign.cmetadata.config.find(
    (c) => c.intent_type === intentType
  );
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();

  const [option, setOption] = useState<CampaignConfig<ConfigPayload>>(
    existOption || {
      intent_type: intentType,
      who: 'Assign to a person',
      payload: { user_id: campaignSupport.members[0]?.id || '' },
    }
  );

  const {
    data: members,
    isLoading,
    isSuccess,
  } = useQuery({
    queryKey: ['members', campaign.organization_id],
    queryFn: () =>
      OrganizationApi.searchMember(campaign.organization_id, { filter: {} }),
  });
  useEffect(() => {
    if (isSuccess) {
      setCampaignSupport((prev) => ({
        ...prev,
        members: members.data.list.map((l) => l.entity),
      }));
      if (!existOption) {
        setOption((prev) => ({
          ...prev,
          payload: { user_id: members.data.list[0].entity.id },
        }));
      }
    }
  }, [isSuccess]);
  const {
    mutateAsync: assignCampaignIntent,
    isPending: isAssignCampaignIntentPending,
  } = useMutation({
    mutationFn: () => {
      const copy = [...(campaign.cmetadata.config || [])];
      const payload = {
        config: copy.filter((c) => c.intent_type !== intentType).concat(option),
      };
      return CampaignApi.assign(campaign.id, payload);
    },
    onSuccess: () => {
      const copy = [...(campaign.cmetadata.config || [])];
      const payload = {
        config: copy.filter((c) => c.intent_type !== intentType).concat(option),
      };
      setCampaign((prev) => ({
        ...prev,
        cmetadata: { ...prev.cmetadata, ...payload },
      }));
      onSuccess();
      closeSheet();
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    },
  });

  const hasChanged =
    !existOption || existOption.payload.user_id !== option.payload.user_id;

  const handleSave = () => {
    assignCampaignIntent();
  };

  const leftSection = (
    <AddButton
      text="New"
      onClick={() => router.push('/team-members')}
      variant="outline"
    />
  );

  return (
    <div className="flex h-full flex-col">
      <SheetHeader
        title="Assign To A Person"
        hasChanged={hasChanged}
        onSubmit={handleSave}
        onClose={closeSheet}
        leftSection={leftSection}
      />
      <Separator />
      <div className="h-96 flex-grow">
        <Spinner
          show={isAssignCampaignIntentPending}
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
              {campaignSupport.members.map((member) => (
                <Card key={member.id}>
                  <CardContent className="pt-4">
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-4">
                        <Avatar className="h-12 w-12">
                          <AvatarImage
                            src={member.picture || ''}
                            alt={member.name}
                          />
                          <AvatarFallback>
                            {member.email.slice(0, 2).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <div className="max-w-[200px]">
                          <div className="flex items-center gap-2">
                            <p className="truncate text-base font-semibold">
                              {member.name}
                            </p>
                          </div>
                          <p className="truncate text-muted-foreground">
                            {member.email}
                          </p>
                        </div>
                      </div>
                      <Switch
                        checked={option.payload.user_id === member.id}
                        onCheckedChange={() =>
                          setOption((prev) => ({
                            ...prev,
                            payload: { user_id: member.id },
                          }))
                        }
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
