'use client';

import { IconBrandGoogle } from '@/assets/icons';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { PropsWithChildren, useState } from 'react';

export function CreateAgentForm({
  onSuccess,
  onGmailAuth
}: {
  onSuccess: () => void;
  onGmailAuth: () => void;
}) {
  const onSubmit = () => {
    onGmailAuth();
    onSuccess();
  };

  return (
    <Button variant="outline" onClick={onSubmit}>
      <IconBrandGoogle />
      Connect using Google
    </Button>
  );
}

export function CreateAgentDialog({
  children,
  onGmailAuth
}: PropsWithChildren<{ onGmailAuth: () => void }>) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>Add a New Agent</DialogTitle>
          <DialogDescription>
            Add a new agent to your system to manage tasks, handle inquiries,
            and support your operations. Provide the necessary details to
            complete the setup.
          </DialogDescription>
        </DialogHeader>
        <CreateAgentForm
          onSuccess={() => setIsDialogOpen(false)}
          onGmailAuth={onGmailAuth}
        />
      </DialogContent>
    </Dialog>
  );
}
