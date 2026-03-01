'use client';

import { useEffect, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Separator } from '@/components/ui/separator';
import { IconBrandGoogle } from '@/assets/icons';
import { AddButton } from '@/components/buttons/add-button';
import { EditButton } from '@/components/buttons/edit-button';
import { CreateAgentDialog } from '@/components/forms/create-agent-form';
import { ContentLayout } from '@/components/nav/content-layout';
import { Badge } from '@/components/ui/badge';
import type { Agent } from '@/models/agent';
import AgentApi from '@/network/client/agent';
import { Skeleton } from '@/components/ui/skeleton';
import { useRouter } from 'next/navigation';
import AuthApi from '@/network/client/auth';
import { gmailRedirectUri } from '@/helpers/env';

function AgentsPage() {
  const router = useRouter();
  const popupRef = useRef<Window | null>(null);

  const {
    data: agents,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ['agents'],
    queryFn: () => AgentApi.list({ filter_: {} }),
  });

  const openGmailPopup = async () => {
    try {
      const { data } = await AuthApi.getGmailAuthURL(gmailRedirectUri);
      const feature = `width=600,height=600,left=${(window.screen.width - 600) / 2},top=${(window.screen.height - 600) / 2}`;
      popupRef.current = window.open(data.url, '_blank', feature);
    } catch (err: any) {
      toast.error(err.message);
    }
  };

  useEffect(() => {
    const listener = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;

      if (event.data?.type === 'GMAIL_AUTH') {
        toast.success('Authorized agent successfully');
        refetch();
        if (popupRef.current && !popupRef.current.closed) {
          popupRef.current.close();
          popupRef.current = null;
        }
      }
    };

    window.addEventListener('message', listener);

    return () => {
      window.removeEventListener('message', listener, false);
    };
  }, []);

  const datas = agents?.data?.list.flatMap((item) => item.entity) || [];

  const rightSection = (
    <CreateAgentDialog onGmailAuth={openGmailPopup}>
      <AddButton text="Add agent" />
    </CreateAgentDialog>
  );

  return (
    <ContentLayout title="AI Agents" rightSection={rightSection}>
      {isLoading ? (
        <div className="grid grid-cols-3 gap-4">
          <Skeleton className="h-[200px] rounded-lg" />
          <Skeleton className="h-[200px] rounded-lg" />
          <Skeleton className="h-[200px] rounded-lg" />
        </div>
      ) : (
        <div>
          {datas.length === 0 ? (
            <div className="flex items-center justify-center">
              <p className="text-gray-500">No agent</p>
            </div>
          ) : (
            <div className="grid grid-cols-3 gap-4">
              {datas.map((agent: Agent, index: number) => (
                <Card key={agent.id}>
                  <CardContent className="pt-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <Avatar className="h-12 w-12">
                          <AvatarImage src={agent.picture} alt={agent.name} />
                          <AvatarFallback>
                            {agent.email.slice(0, 2).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <div>
                          <p className="text-base font-semibold">
                            {agent.name}
                          </p>
                          <p className="text-muted-foreground">
                            Agent {index + 1}
                          </p>
                        </div>
                      </div>
                      <EditButton
                        variant="outline"
                        onClick={() => router.push(`/agents/${agent.id}`)}
                      />
                    </div>
                    <Separator className="my-4" />
                    <div className="space-y-2">
                      <div className="flex items-center justify-between gap-2">
                        <div className="text-muted-foreground">Email</div>
                        <div>{agent.email}</div>
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <div className="text-muted-foreground">Status</div>
                        <div>
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
                      </div>
                      <div className="flex items-center justify-between gap-2">
                        <div className="text-muted-foreground">Provider</div>
                        <div className="flex items-center gap-2">
                          <IconBrandGoogle className="h-4 w-4" />{' '}
                          <span className="capitalize">
                            {agent.email_provider}
                          </span>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}
    </ContentLayout>
  );
}

export default AgentsPage;
