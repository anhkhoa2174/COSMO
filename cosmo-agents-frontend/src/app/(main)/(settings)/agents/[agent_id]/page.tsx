'use client';

import { IconBrandGoogle } from '@/assets/icons';
import { BackButton } from '@/components/buttons/back-button';
import { MainButton } from '@/components/buttons/main-button';
import { PlateEditor } from '@/components/editor/plate-editor';
import { ContentLayout } from '@/components/nav/content-layout';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { gmailRedirectUri } from '@/helpers/env';
import AgentApi, { getAgent } from '@/network/client/agent';
import AuthApi from '@/network/client/auth';
import { useUser } from '@/hooks/use-user';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  AlertCircle,
  Calendar,
  Link,
  Mail,
  MessageSquare,
  User,
} from 'lucide-react';
import { useEffect, useState } from 'react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { toast } from 'sonner';

export default function AgentDetailPage({
  params,
}: {
  params: { agent_id: string };
}) {
  const queryClient = useQueryClient();
  const [pendingItems, setPendingItems] = useState({});

  const { data, isLoading, isSuccess, isError, refetch, error } = useQuery({
    queryKey: ['agents', params.agent_id],
    queryFn: () => getAgent(params.agent_id),
  });

  const agent = data?.data.entity;
  const inbox = data?.data.inbox_detail;

  const [emailSignature, setEmailSignature] = useState('');
  const [emailLimit, setEmailLimit] = useState(0);
  const [trigger, setTrigger] = useState(0);

  useEffect(() => {
    if (isSuccess) {
      setEmailSignature(data.data.entity.signature || '');
      setTrigger((prev) => prev + 1);
      setEmailLimit(data.data.entity.daily_limit);
    }
  }, [isSuccess]);

  const { user } = useUser();

  const { mutate: update, isPending } = useMutation({
    mutationFn: ({ key, ...data }: any) =>
      AgentApi.update(params.agent_id, data),
    onMutate({ key }) {
      setPendingItems((prev) => ({ ...prev, [key]: true }));
    },
    onSuccess(_, { key, ...data }) {
      toast.success('Agent updated successfully');
      setPendingItems((prev) => ({ ...prev, [key]: false }));
      queryClient.setQueryData(['agents', params.agent_id], (oldData: any) => ({
        ...oldData,
        data: {
          ...oldData.data,
          entity: {
            ...oldData.data.entity,
            ...(key === 'DAILY_LIMIT' ? { daily_limit: emailLimit } : {}),
            ...(key === 'SIGNATURE' ? { signature: emailSignature } : {}),
            ...(key === 'STATUS' ? { status: data.status } : {}),
          },
        },
      }));
    },
    onError(err: any, { key }) {
      toast.error(err.message);
      setPendingItems((prev) => ({ ...prev, [key]: false }));
    },
  });

  const mergeTag = {
    '{organization_name}': user?.organizations[0]?.name || '',
    '{organization_company_url}': user?.organizations[0]?.company_url || '',
    '{sender_email}': agent?.email || '',
    '{sender_name}': agent?.name || '',
  };

  const rightSection = emailSignature !== agent?.signature && (
    <MainButton
      text="Save changes"
      loading={pendingItems['SIGNATURE']}
      onClick={() => update({ key: 'SIGNATURE', signature: emailSignature })}
    />
  );

  const [displayValue, setDisplayValue] = useState('0');

  // Update display value whenever the actual value changes
  useEffect(() => {
    // Convert to string but ensure no leading zeros
    setDisplayValue(emailLimit.toString());
  }, [emailLimit]);

  const handleValueChange = (e) => {
    const min = 0;
    const max = agent?.max_daily_limit || 0;

    // Get the raw input value
    const inputValue = e.target.value;

    // Handle empty input
    if (inputValue === '') {
      setEmailLimit(min);
      return;
    }

    // Parse as base-10 number to prevent octal interpretation
    let newValue = parseInt(inputValue, 10);

    // Handle NaN case
    if (isNaN(newValue)) {
      setEmailLimit(min);
      return;
    }

    // Auto-constrain to min/max
    if (newValue > max) {
      newValue = max;
    } else if (newValue < min) {
      newValue = min;
    }

    setEmailLimit(newValue);
  };

  return (
    <ContentLayout
      title="AI Agent Settings"
      className="container mx-auto"
      leftSection={<BackButton href="/agents" />}
      rightSection={rightSection}
    >
      {isError && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error: {error.name}</AlertTitle>
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      )}
      {isLoading ? (
        <Skeleton className="h-[160px] rounded-xl" />
      ) : agent && inbox ? (
        <>
          <Card>
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
                    <div className="flex items-center gap-2">
                      <p className="text-base font-semibold">{agent.name}</p>
                      <Badge
                        variant={
                          agent.status === 'active' ? 'success' : 'destructive'
                        }
                        className="capitalize"
                      >
                        {agent.status}
                      </Badge>
                    </div>
                    <p className="text-muted-foreground">{agent.email}</p>
                  </div>
                </div>
                {agent.status === 'active' ? (
                  <MainButton
                    variant="outline"
                    text="Deactivate"
                    onClick={() =>
                      update({ key: 'STATUS', status: 'inactive' })
                    }
                    loading={pendingItems['STATUS']}
                  />
                ) : agent.status === 'inactive' ? (
                  <MainButton
                    variant="outline"
                    text="Activate"
                    onClick={() => update({ key: 'STATUS', status: 'active' })}
                    loading={pendingItems['STATUS']}
                  />
                ) : (
                  <ReAuthorizeButton onSuccess={refetch} />
                )}
              </div>
              <Separator className="my-4" />
              <div>
                <p className="text-base font-semibold">
                  AI Agent Configuration
                </p>
                <p className="text-muted-foreground">
                  Manage inbox details, email limitations, and signature
                  settings
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <MessageSquare className="h-5 w-5 text-accent-foreground" />{' '}
                Inbox Details
              </CardTitle>
            </CardHeader>
            <Separator />
            <CardContent className="pt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="flex flex-col gap-2">
                  <p className="font-medium">Connected date</p>
                  <div className="flex items-center gap-2 rounded-md border bg-zinc-100 p-2 text-sm text-muted-foreground">
                    <Calendar className="h-4 w-4" />{' '}
                    {format(new Date(inbox.connected_date), 'PPpp')}
                  </div>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="font-medium">Email provider</p>
                  <div className="flex items-center gap-2 rounded-md border bg-zinc-100 p-2 text-sm text-muted-foreground">
                    <IconBrandGoogle className="h-4 w-4" />{' '}
                    <span className="capitalize">{inbox.email_provider}</span>
                  </div>
                </div>
                <div className="col-span-2 flex flex-col gap-2">
                  <p className="font-medium">Added by</p>
                  <div className="flex items-center gap-2 rounded-md border bg-zinc-100 p-2 text-sm text-muted-foreground">
                    <User className="h-4 w-4" /> {inbox.added_by} &lt;
                    {inbox.connected_inbox}&gt;
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Mail className="h-5 w-5 text-accent-foreground" /> Email
                Limitations
              </CardTitle>
            </CardHeader>
            <Separator />
            <CardContent className="pt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2 flex flex-col gap-2">
                  <div className="flex items-center justify-between gap-2">
                    <p className="font-medium">Daily limit</p>
                    <p className="font-medium text-accent-foreground">
                      {(agent.daily_limit * 100) / agent.max_daily_limit} %
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Progress
                      value={(agent.daily_limit * 100) / agent.max_daily_limit}
                      className="[&>div]:bg-accent-foreground"
                    />
                    <p className="flex-shrink-0 text-muted-foreground">
                      {agent.daily_limit} / {agent.max_daily_limit}
                    </p>
                  </div>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="font-medium">Adjust daily sending limit</p>
                  <div className="flex items-center gap-2">
                    <Input
                      type="number"
                      min={0}
                      max={agent.max_daily_limit}
                      step="any"
                      value={displayValue}
                      onChange={handleValueChange}
                    />
                    <MainButton
                      text="Apply"
                      variant="secondary"
                      disabled={emailLimit === agent.daily_limit}
                      loading={pendingItems['DAILY_LIMIT']}
                      onClick={() =>
                        update({ key: 'DAILY_LIMIT', daily_limit: emailLimit })
                      }
                    />
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Link className="h-5 w-5 text-accent-foreground" /> Email
                Signature
              </CardTitle>
            </CardHeader>
            <Separator />
            <CardContent className="pt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="flex flex-col gap-2">
                  <p className="font-medium">Edit Signature</p>
                  <div className="overflow-hidden rounded-lg border">
                    <PlateEditor
                      value={emailSignature}
                      onChange={setEmailSignature}
                      deps={[trigger]}
                    />
                  </div>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="font-medium">Signature Preview</p>
                  <div className="overflow-hidden rounded-lg border p-4 text-base leading-loose">
                    <Markdown
                      remarkPlugins={[remarkGfm]}
                      rehypePlugins={[rehypeRaw]}
                    >
                      {Object.entries(mergeTag).reduce(
                        (sign, [key, value]) =>
                          sign.replace(new RegExp(key, 'g'), value),
                        emailSignature
                      )}
                    </Markdown>
                  </div>
                </div>
                <div className="col-span-2 text-muted-foreground">
                  Available variables: &#123;sender_name&#125;,
                  &#123;sender_email&#125;, &#123;organization_name&#125;,
                  &#123;organization_company_url&#125;
                </div>
              </div>
            </CardContent>
          </Card>
        </>
      ) : null}
    </ContentLayout>
  );
}

function ReAuthorizeButton({ onSuccess }: { onSuccess: () => void }) {
  const getGmailAuth = async () => {
    try {
      const response = await AuthApi.getGmailAuthURL(gmailRedirectUri);
      const width = 600;
      const height = 600;
      const left = (window.screen.width - width) / 2;
      const top = (window.screen.height - height) / 2;
      const feature = `width=${width},height=${height},left=${left},top=${top},status=yes,toolbar=no,menubar=no,location=yes`;
      const authWindow = window.open(response.data.url, '_blank', feature);

      if (!authWindow) {
        throw new Error(
          'Popup was blocked. Please allow popups for this site.'
        );
      }

      const timeout = setTimeout(() => {
        authWindow.close();
        toast.error('Authentication timeout. Please try again.');
      }, 120000);

      const handleMessage = (
        event: MessageEvent<{ type: string; data: any }>
      ) => {
        if (event.origin !== window.location.origin) return;

        if (event.data?.type === 'GMAIL_AUTH') {
          const data = event.data.data;
          clearTimeout(timeout);
          authWindow.close();
          window.removeEventListener('message', handleMessage);

          if (!data) {
            toast.error('Authentication failed. Please try again.');
            return;
          }

          toast.success('Re-authorized agent successfully');
        }
      };

      window.addEventListener('message', handleMessage);

      const checkWindow = setInterval(() => {
        if (authWindow.closed) {
          clearInterval(checkWindow);
          clearTimeout(timeout);
          window.removeEventListener('message', handleMessage);
          onSuccess(); // Although getting error and authWindow is closed after 5s, the agent is still re-authorized => Need to put onSuccess here
        }
      }, 500);
    } catch (error: any) {
      toast.error(error.message || 'Failed to authenticate with Google');
    }
  };

  return (
    <MainButton text="Re-authorize" onClick={getGmailAuth} variant="outline" />
  );
}
