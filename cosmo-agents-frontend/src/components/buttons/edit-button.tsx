import React from 'react';
import { Pen } from 'lucide-react';
import type { ButtonProps } from '@/components/ui/button';
import { MainButton } from './main-button';

export interface EditButtonProps extends Omit<ButtonProps, 'asChild'> {
  text?: string;
}

const EditButton = React.forwardRef<HTMLButtonElement, EditButtonProps>(
  ({ text = 'Edit', ...props }, ref) => {
    return <MainButton ref={ref} icon={Pen} text={text} {...props} />;
  }
);

EditButton.displayName = 'EditButton';

export { EditButton };
