import { Pause, Play, Rocket } from 'lucide-react';
import { MainButton } from '@/components/buttons/main-button';
import { ButtonProps } from '@/components/ui/button';
import type { CampaignStatus } from '@/models/campaign';

interface StatusButtonProps extends ButtonProps {
  status: CampaignStatus;
  onClick: () => void;
}

const StatusButton = (props: StatusButtonProps) => {
  const { status, onClick, ...rest } = props;
  switch (status) {
    case 'paused':
      return <MainButton icon={Play} text="Play" onClick={onClick} {...rest} />;
    case 'active':
      return (
        <MainButton
          icon={Pause}
          variant="destructive"
          text="Pause"
          onClick={onClick}
          {...rest}
        />
      );
    case 'draft':
      return (
        <MainButton icon={Play} text="Launch" onClick={onClick} {...rest} />
      );
    default:
      return null;
  }
};

export default StatusButton;
