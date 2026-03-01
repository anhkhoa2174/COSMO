import { fakeStream } from '@/helpers';
import { useEffect, useRef } from 'react';

export function useTypingEffect({
  text,
  isActive,
  setText,
  speed,
  step,
  dependencies = [],
  onFinished,
}: {
  text: string | undefined;
  isActive: boolean;
  setText: (value: string) => void;
  speed: number;
  step: number;
  dependencies: any[];
  onFinished?: () => void;
}) {
  const currentText = useRef('');

  const handleFinished = () => {
    currentText.current = text || '';
    onFinished?.();
  };

  useEffect(() => {
    if (!text) {
      setText('');
      return;
    }
    if (!isActive) return;
    if (currentText.current === text) return;

    fakeStream({
      data: text,
      callback: (data) => {
        setText(data);
      },
      step,
      time: speed,
      onFinished: handleFinished,
    });
  }, [text, isActive, currentText, ...dependencies]);
}
