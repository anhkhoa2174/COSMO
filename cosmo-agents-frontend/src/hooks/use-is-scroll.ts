import React, { useEffect } from 'react';

export default function useIsScroll(
  refElement: React.RefObject<HTMLDivElement>
) {
  const [isScroll, setIsScroll] = React.useState(false);
  useEffect(() => {
    if (!refElement.current) return;
    const handleScroll = () => {
      if (refElement.current) {
        setIsScroll(refElement.current.scrollTop > 0);
      }
    };
    const ref = refElement.current;
    ref.addEventListener('scroll', handleScroll);
    handleScroll();
    return () => {
      ref.removeEventListener('scroll', handleScroll);
    };
  }, [refElement]);
  return isScroll;
}
