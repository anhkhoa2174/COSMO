import React from 'react';
import { PlusCircle } from 'lucide-react';
import type { ButtonProps } from '@/components/ui/button';
import { MainButton } from './main-button';

export interface AddButtonProps extends Omit<ButtonProps, 'asChild'> {
  text?: string;
}

const AddButton = React.forwardRef<HTMLButtonElement, AddButtonProps>(
  ({ text = 'Add', ...props }, ref) => {
    return <MainButton ref={ref} icon={PlusCircle} text={text} {...props} />;
  }
);

AddButton.displayName = 'AddButton';

export { AddButton };
