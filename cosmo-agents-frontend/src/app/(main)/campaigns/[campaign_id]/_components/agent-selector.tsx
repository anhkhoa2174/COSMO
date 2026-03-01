import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { AddButton } from '@/components/buttons/add-button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Spinner } from '@/components/ui/spinner';
import { getValuable } from '@/lib/utils';
import AgentApi from '@/network/client/agent';
import CampaignApi from '@/network/client/campaign';
import { useCampaign } from '../use-campaign';
import { SheetHeader } from './sheet-header';
import { Skeleton } from '@/components/ui/skeleton';

interface AgentSelectorProps {
  closeSheet: () => void;
}

export default function AgentSelector({ closeSheet }: AgentSelectorProps) {
  const router = useRouter();

  const [campaign, setCampaign] = useCampaign();
  const [agentId, setAgentId] = useState(campaign.agent_id);

  const { data, isLoading } = useQuery({
    queryKey: ['agents'],
    queryFn: () => AgentApi.list({ filter_: {} }),
  });
  const updateCampaignMutation = useMutation({
    mutationFn: () =>
      CampaignApi.update(campaign.id, getValuable({ agent_id: agentId })),
    onSuccess: () => {
      setCampaign((prev) => ({ ...prev, agent_id: agentId }));
    },
    onError: (err: any) => {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    },
  });

  const shouldAssign = agentId !== campaign.agent_id;

  const handleSaveAndClose = async () => {
    if (shouldAssign) {
      await updateCampaignMutation.mutateAsync();
    }
    closeSheet();
  };

  const leftSection = (
    <AddButton
      variant="outline"
      text="New"
      onClick={() => router.push('/agents')}
    />
  );

  return (
    <div className="flex h-full flex-col">
      <SheetHeader
        title="AI Inbox"
        hasChanged={shouldAssign}
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
                  You can't change the AI Inbox for a campaign that is not in
                  draft mode
                </p>
              )}
              {data?.data.list.length === 0 && (
                <p className="text-center">No agent found</p>
              )}
              {data?.data.list.map(({ entity: agent }) => (
                <Card key={agent.id}>
                  <CardContent className="pt-4">
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-4">
                        <Avatar className="h-12 w-12">
                          <AvatarImage src={agent.picture} alt={agent.name} />
                          <AvatarFallback>
                            {agent.email.slice(0, 2).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <div className="max-w-[200px]">
                          <div className="flex items-center gap-2">
                            <p className="truncate text-base font-semibold">
                              {agent.name}
                            </p>
                            <Badge
                              variant={
                                agent.status === 'active'
                                  ? 'success'
                                  : 'destructive'
                              }
                              className="capitalize"
                            >
                              {agent.status}
                            </Badge>
                          </div>
                          <p className="truncate text-muted-foreground">
                            {agent.email}
                          </p>
                        </div>
                      </div>
                      <Switch
                        checked={agentId === agent.id}
                        onCheckedChange={() => setAgentId(agent.id)}
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
