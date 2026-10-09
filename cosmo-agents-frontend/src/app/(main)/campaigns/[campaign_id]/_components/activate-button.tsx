'use client';

import { useEffect, useState } from 'react';
import { CheckCircle2, XCircle } from 'lucide-react';
import { toast } from 'sonner';
import { Alert, Button, Group, Modal, Stack, Text } from '@mantine/core';
import StatusButton from './status-button';
import { Spinner } from '@/components/ui/spinner';
import { useDisclosure } from '@/hooks/use-disclosure';
import type { CampaignStatus } from '@/models/campaign';
import CampaignApi from '@/network/client/campaign';
import { useCampaign } from '../use-campaign';

const ActivateButton = ({ campaignId }: { campaignId: string }) => {
  const [campaign, setCampaign] = useCampaign();

  const [isLoading, setIsLoading] = useState(false);

  const [opened, { open, close }] = useDisclosure(false);
  const [selectedStatus, setSelectedStatus] = useState<CampaignStatus>(
    campaign?.status || 'draft'
  );

  const newStatus: Record<CampaignStatus, CampaignStatus> = {
    active: 'paused',
    paused: 'active',
    draft: 'active',
    ended: 'draft',
    scheduled: 'draft',
  };

  const handleLive = async () => {
    setIsLoading(true);
    try {
      await CampaignApi.update(campaignId, {
        status: newStatus[selectedStatus],
      });
      toast.success('Update campaign successfully');
      setCampaign({ ...campaign, status: newStatus[selectedStatus] });
    } catch (err: any) {
      toast.error(
        err.error?.message || err.message || 'Oops! Something went wrong'
      );
    } finally {
      close();
      setIsLoading(false);
    }
  };

  const text: Record<CampaignStatus, string> = {
    active:
      'The campaign will be paused, and the email will not be sent to the contact list.',
    paused:
      'The campaign will be continued, and the email will be sent to the contact list.',
    draft:
      'The campaign will be activated, and the email will be sent to the contact list.',
    ended: 'The campaign has ended.',
    scheduled: 'The campaign is scheduled.',
  };

  useEffect(() => {
    setSelectedStatus(campaign?.status || 'draft');
  }, [campaign?.status]);

  const isLaunch = campaign.list_contact_id && campaign.agent_id;

  const prerequisites = [
    {
      condition: campaign.list_contact_id,
      message: 'Complete the entry rules to active the campaign',
      title: 'Entry Rules',
    },
    {
      condition: campaign.agent_id,
      message: 'Select the AI inbox to active the campaign',
      title: 'AI Inbox',
    },
  ];

  return (
    <>
      <StatusButton status={selectedStatus} onClick={open} />
      <Modal
        opened={opened}
        onClose={close}
        centered
        title={
          <Text fw={600} span size="md">
            Confirmation
          </Text>
        }
        withCloseButton={false}
        closeOnEscape={false}
      >
        <Spinner show={isLoading} withOverlay />
        {!isLaunch ? (
          <div>
            <Stack>
              {prerequisites.map((prerequisite) => (
                <Alert
                  key={prerequisite.title}
                  p="sm"
                  radius="md"
                  color={prerequisite.condition ? 'teal' : 'red'}
                  title={prerequisite.title}
                  icon={prerequisite.condition ? <CheckCircle2 /> : <XCircle />}
                >
                  {prerequisite.condition ? 'Completed' : prerequisite.message}
                </Alert>
              ))}
            </Stack>
          </div>
        ) : (
          <>
            <Text>{text[selectedStatus]}</Text>
            <Group justify="flex-end" mt="sm">
              <Button variant="light" onClick={close}>
                Cancel
              </Button>
              <Button variant="light" color="green" onClick={handleLive}>
                Continue
              </Button>
            </Group>
          </>
        )}
      </Modal>
    </>
  );
};

export default ActivateButton;
