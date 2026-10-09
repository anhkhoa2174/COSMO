'use client';

import { BackButton } from '@/components/buttons/back-button';
import { ContentLayout } from '@/components/nav/content-layout';
import DescriptionText from '@/components/tour/DescriptionText';
import { Spinner } from '@/components/ui/spinner';
import {
  createTourCustomContent,
  UpdateOnboarding,
  useTour,
} from '@/hooks/use-tour';
import CampaignApi from '@/network/client/campaign';
import { useUser } from '@/hooks/use-user';
import classes from '@/styles/Campaign.module.css';
import { Box, Card, Center, SimpleGrid, Text, Title } from '@mantine/core';
import { useMutation } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { toast } from 'sonner';

export default function Playbook() {
  const router = useRouter();
  const { user, invalidate } = useUser();

  const { mutate, isPending } = useMutation({
    mutationFn: CampaignApi.create,
    onSuccess: (data) => router.push(`/campaigns/${data.data.id}`),
    onError: (error) => {
      toast.error(error.message || 'Failed to create campaign');
    },
  });

  const { start, reset } = useTour([
    {
      element: '#campaigns-playbook-0',
      popover: {
        title: '',
        side: 'left',
        description: '',
        customContent: createTourCustomContent({
          isNextSubmit: true,
          title: 'Choose playbook',
          description: (
            <DescriptionText
              title="Select a Playbook"
              description="Pick a message sequence or automation flow to guide how your campaign unfolds."
            />
          ),
          imageSrc: '',
          user,
          onNext: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onPrev: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          onFinish: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'completed',
                },
              },
              user
            );
          },
          onSkipAll: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'skipped',
                },
              },
              user
            );
          },
          onNavigate: (step: number) => {
            UpdateOnboarding(
              {
                campaigns: {
                  onboarding: true,
                  step: step,
                  status: 'in_progress',
                },
              },
              user
            );
          },
          update: invalidate,
        }),
      },
    },
  ]);

  useEffect(() => {
    if (user?.ui_metadata?.campaigns?.onboarding) {
      start();
    }
    return () => {
      reset();
    };
  }, [user]);

  return (
    <ContentLayout
      variant="editor"
      title="Campaign Playbooks"
      leftSection={<BackButton href="/campaigns" />}
    >
      <Center className={classes.playbookContainer}>
        <Spinner show={isPending} withOverlay />
        <Box>
          <Title order={3} ta="center">
            Select a campaign playbook to start
          </Title>
          <SimpleGrid cols={3} mt={48} spacing={48}>
            {playbooks.map((playbook, index) => (
              <Card
                key={`playbook-${index}`}
                id={`campaigns-playbook-${index}`}
                classNames={{ root: classes.playbookRoot }}
                onClick={() =>
                  mutate({
                    name: playbook.title,
                    playbook: playbook.type,
                  })
                }
              >
                <Text fz={40}>{playbook.emoji}</Text>
                <Text fw={600} size="md">
                  {playbook.title}
                </Text>
                <Text fw={500} c="dark.2" mt="md">
                  {playbook.description}
                </Text>
              </Card>
            ))}
          </SimpleGrid>
        </Box>
      </Center>
    </ContentLayout>
  );
}

const playbooks = [
  {
    emoji: '🔊',
    title: 'Revive dormant leads',
    description: 'Reach out to dormant leads in your CRM',
    type: 'revive_dormant_leads',
  },
  {
    emoji: '🎯',
    title: 'Upsell to existing customers',
    description: 'Upsell new product to existing customer base',
    type: 'upsell_to_existing_customers',
  },
  {
    emoji: '💌',
    title: 'Event Invite',
    description:
      'Invite people to your event or webinar with a personalized email',
    type: 'event_invite',
  },
  {
    emoji: '🎁',
    title: 'Content Offering',
    description: 'Offer ebook or new content to warm up your leads',
    type: 'content_offering',
  },
  {
    emoji: '🗂️',
    title: 'Webinar follow up',
    description: 'Follow up with contacts post webinar or event',
    type: 'webinar_follow_up',
  },
  {
    emoji: '🔎',
    title: 'Click Start from Scratch',
    description: 'Create your own AI campaign',
    type: 'click_start_from_scratch',
  },
];
