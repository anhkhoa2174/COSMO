import { useCallback, useState } from 'react';
import { useDisclosure } from './use-disclosure';

export const useSheet = () => {
  const breakpoint = {
    xs: 'w-[400px] sm:max-w-[400px]',
    sm: 'w-[640px] sm:max-w-[640px]',
    md: 'w-[768px] sm:max-w-[768px]',
    lg: 'w-[1024px] sm:max-w-[1024px]',
    xl: 'w-[1280px] sm:max-w-[1280px]',
    '2xl': 'w-[1536px] sm:max-w-[1536px]',
  };

  const [size, setSize] = useState<'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl'>(
    'xs'
  );
  const [content, setContent] = useState<React.ReactNode | null>(null);
  const [opened, { open, close }] = useDisclosure();

  const handleClose = useCallback(() => {
    setContent(null);
    close();
  }, [close]);

  const handleOpen = useCallback(
    (params: {
      size?: typeof size;
      className?: string;
      content: React.ReactNode;
    }) => {
      setSize(params.size || 'xs');
      setContent(params.content);
      open();
    },
    [open]
  );

  return {
    opened,
    size: breakpoint[size],
    content,
    close: handleClose,
    open: handleOpen,
  };
};
