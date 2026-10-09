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
    // Chưa có nội dung để gõ thì KHÔNG đụng vào ô soạn thảo. Trước đây nhánh
    // này gọi setText('') và xoá sạch nội dung mẫu của template ngay khi mở
    // trình soạn, làm nội dung hiện ra được một đoạn rồi cụt.
    if (!text) return;
    if (!isActive) return;
    if (currentText.current === text) return;

    const cancel = fakeStream({
      data: text,
      callback: (data) => {
        setText(data);
      },
      step,
      time: speed,
      onFinished: handleFinished,
    });

    return cancel;
    // currentText là ref nên không bao giờ đổi — để trong deps chỉ gây hiểu nhầm.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [text, isActive, ...dependencies]);
}
