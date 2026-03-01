'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { DialogProps } from '@radix-ui/react-dialog';
import { CheckCircle2 } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';
import _ from 'lodash';
import { Bell, Calendar } from 'lucide-react';
import { toast } from 'sonner';
import { Badge, Group, Text } from '@mantine/core';
import { MainButton } from '@/components/buttons/main-button';
import { DatetimePicker } from '@/components/datetime-picker';
import { SelectMenu } from '@/components/SelectMenu';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Separator } from '@/components/ui/separator';
import { Spinner } from '@/components/ui/spinner';
import { getValuable } from '@/lib/utils';
import { MemberSearchResponse } from '@/models/organization';
import CampaignApi from '@/network/client/campaign';
import OrganizationApi from '@/network/client/organization';
import classes from '@/styles/Campaign.module.css';
import { useCampaign, useCampaignSupport } from '../use-campaign';
import { SheetHeader } from './sheet-header';

export default function Settings({ closeSheet }: { closeSheet: () => void }) {
  const router = useRouter();

  const [campaign, setCampaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();

  const [notificationIds, setNotificationIds] = useState<string[]>(
    campaign.notifications.map((n) => n.user_id)
  );
  const [isSettingScheduleDialogOpen, setIsSettingScheduleDialogOpen] = useState(false);

  const { mutateAsync, isPending } = useMutation({
    mutationFn: () =>
      CampaignApi.update(campaign.id, {
        ...getValuable({
          notification_ids: notificationIds,
        }),
      }),
    onSuccess: () => {
      setCampaign((prev) => ({
        ...prev,
        notifications: campaignSupport.members
          .filter((n) => notificationIds.includes(n.id))
          .map((n) => ({ user_id: n.id })),
      }));
      closeSheet();
    },
    onError: (err: any) => {
      toast.error(err.error?.message || err.message || 'Cannot update schedule');
    },
  });

  const hasChanged = useMemo(
    () =>
      !_.isEqual(
        notificationIds,
        campaign.notifications.map((n) => n.user_id)
      ),
    [notificationIds, campaign]
  );

  const handleSave = async () => {
    mutateAsync();
  };

  return (
    <div className="flex flex-col h-full">
      <SheetHeader
        title="Settings"
        hasChanged={hasChanged}
        onSubmit={handleSave}
        onClose={closeSheet}
      />
      <Separator />
      <div className="flex-1 p-4 flex flex-col gap-4">
        <Spinner show={isPending} withOverlay />
        <Group className={classes.listContact} onClick={() => setIsSettingScheduleDialogOpen(true)}>
          <Group>
            <Calendar />
            <Text fw={500}>Schedule</Text>
          </Group>
          {campaign.schedule && (
            <Text span c="teal.7">
              <CheckCircle2 size={16} />
            </Text>
          )}
        </Group>
        <SelectMenu
          value={notificationIds}
          data={campaignSupport.members.map((n) => ({
            label: n.name,
            value: n.id,
          }))}
          onSubmit={setNotificationIds}
          fetchFn={() => OrganizationApi.searchMember(campaign.organization_id, { filter: {} })}
          afterFetch={({ data }: MemberSearchResponse) =>
            setCampaignSupport((prev) => ({
              ...prev,
              members: data.list.map((l) => l.entity),
            }))
          }
          title="Campaign Notification"
          addFn={() => router.push('/team-members')}
        >
          <Group className={classes.listContact}>
            <Group>
              <Bell />
              <Text fw={500}>Notification</Text>
            </Group>
            {!_.isEqual(
              notificationIds,
              campaign.notifications.map((n) => n.user_id)
            ) ? (
              <Badge variant="light" color="yellow">
                Unsaved changes
              </Badge>
            ) : notificationIds.length ? (
              <Text span c="teal.7">
                <CheckCircle2 size={16} />
              </Text>
            ) : (
              <></>
            )}
          </Group>
        </SelectMenu>
        <SettingScheduleDialog
          isOpen={isSettingScheduleDialogOpen}
          setOpen={setIsSettingScheduleDialogOpen}
        />
      </div>
    </div>
  );
}

interface SetScheduleDialogProps extends DialogProps {
  isOpen: boolean;
  setOpen: (open: boolean) => void;
}

function SettingScheduleDialog({ isOpen, setOpen, ...props }: SetScheduleDialogProps) {
  const [campaign, setCampaign] = useCampaign();
  const prevDate = campaign.schedule ? new Date(campaign.schedule) : undefined;

  const [date, setDate] = useState<Date | undefined>(prevDate);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const { mutateAsync, isPending } = useMutation({
    mutationFn: () =>
      CampaignApi.update(campaign.id, {
        ...getValuable({ schedule: date?.toISOString() }),
      }),
    onSuccess: () => {
      setCampaign((prev) => ({
        ...prev,
        schedule: date?.toISOString() || null,
      }));
      setOpen(false);
      toast.success(`Email scheduled for ${date?.toLocaleString()}`);
    },
    onError: (err: any) => {
      toast.error(err.error?.message || err.message || 'Cannot update schedule');
    },
  });

  const hasChanged = useMemo(
    () => date?.toISOString() !== prevDate?.toISOString(),
    [date, prevDate]
  );

  const handleSubmit = async () => {
    if (date && date.getTime() < Date.now()) {
      setErrorMessage('The scheduled time must be in the future');
      return;
    }
    if (date && date.getTime() < Date.now() + 2 * 60 * 60 * 1000) {
      setErrorMessage('The scheduled time must be at least 2 hours from now');
      return;
    }
    await mutateAsync();
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          setDate(prevDate);
          setErrorMessage(null);
        }
        setOpen(open);
      }}
      {...props}
    >
      <DialogContent className="w-96">
        <DialogHeader>
          <DialogTitle>Schedule Send</DialogTitle>
          <DialogDescription>
            Set the date and time to send the email to the contact list
          </DialogDescription>
        </DialogHeader>
        <div>
          <DatetimePicker
            className="mx-auto"
            value={date}
            onChange={setDate}
            defaultMonth={prevDate}
          />
          {errorMessage && <p className="text-red-500 text-sm">{errorMessage}</p>}
        </div>
        <DialogFooter className="justify-between items-center">
          <p className="text-muted-foreground mr-auto">
            {Intl.DateTimeFormat().resolvedOptions().timeZone} (
            {new Date().toLocaleTimeString('en-us', { timeZoneName: 'short' }).split(' ')[2]})
          </p>
          {hasChanged && (
            <div className="flex gap-2">
              <MainButton text="Save" loading={isPending} onClick={handleSubmit} />
              <MainButton
                text="Discard"
                variant="ghost"
                onClick={() => {
                  setOpen(false);
                  setDate(prevDate);
                  setErrorMessage(null);
                }}
              />
            </div>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
