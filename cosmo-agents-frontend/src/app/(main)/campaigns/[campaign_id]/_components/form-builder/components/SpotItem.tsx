'use client';

import { useDroppable } from '@dnd-kit/core';
import { ReactNode, useEffect, useState } from 'react';

interface SpotItemProps {
  id: string;
  children: ReactNode;
  onDropLeft: () => void;
  onDropRight: () => void;
  onDropTop?: () => void;
  onDropBottom?: () => void;
}

export default function SpotItem({ 
  id, 
  children, 
  onDropLeft, 
  onDropRight, 
  onDropTop, 
  onDropBottom 
}: SpotItemProps) {
  const [isHoveringLeft, setIsHoveringLeft] = useState(false);
  const [isHoveringRight, setIsHoveringRight] = useState(false);
  const [isHoveringTop, setIsHoveringTop] = useState(false);
  const [isHoveringBottom, setIsHoveringBottom] = useState(false);

  const { setNodeRef: setLeftRef, isOver: isOverLeft } = useDroppable({
    id: `${id}-left`,
    data: {
      type: 'spot-left',
      elementId: id,
    },
  });

  const { setNodeRef: setRightRef, isOver: isOverRight } = useDroppable({
    id: `${id}-right`,
    data: {
      type: 'spot-right',
      elementId: id,
    },
  });

  const { setNodeRef: setTopRef, isOver: isOverTop } = useDroppable({
    id: `${id}-top`,
    data: {
      type: 'spot-top',
      elementId: id,
    },
  });

  const { setNodeRef: setBottomRef, isOver: isOverBottom } = useDroppable({
    id: `${id}-bottom`,
    data: {
      type: 'spot-bottom',
      elementId: id,
    },
  });

  // Update hover state based on drag over
  useEffect(() => {
    setIsHoveringLeft(isOverLeft);
    setIsHoveringRight(isOverRight);
    setIsHoveringTop(isOverTop);
    setIsHoveringBottom(isOverBottom);
  }, [isOverLeft, isOverRight, isOverTop, isOverBottom]);

  return (
    <div className="relative w-full">
      {/* Top drop zone - horizontal indicator */}
      {onDropTop && (
        <div
          ref={setTopRef}
          className={`absolute left-0 top-0 w-full transition-all pointer-events-auto ${isOverTop ? 'bg-primary/40' : 'bg-transparent'}`}
          style={{
            zIndex: 20,
            height: '16px',
            opacity: isHoveringTop || isOverTop ? 1 : 0,
            transform: 'translateY(-6px)',
          }}
          onMouseEnter={() => setIsHoveringTop(true)}
          onMouseLeave={() => !isOverTop && setIsHoveringTop(false)}
          onClick={onDropTop}
        >
          {isOverTop && (
            <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-primary text-white text-xs px-1 rounded whitespace-nowrap">
              Insert Above
            </div>
          )}
        </div>
      )}

      {/* The actual form element with side drop zones as overlays */}
      <div className="relative w-full">
        {/* Left drop zone */}
        <div
          ref={setLeftRef}
          className={`absolute left-0 top-0 h-full rounded-l-md transition-all pointer-events-auto ${isOverLeft ? 'bg-primary/40' : 'bg-transparent'}`}
          style={{
            zIndex: 20,
            width: '64px',
            opacity: isHoveringLeft || isOverLeft ? 1 : 0,
            transform: 'translateX(-6px)',
          }}
          onMouseEnter={() => setIsHoveringLeft(true)}
          onMouseLeave={() => !isOverLeft && setIsHoveringLeft(false)}
          onClick={onDropLeft}
        >
          {isOverLeft && (
            <div className="absolute top-1/2 left-0 transform -translate-y-1/2 -translate-x-1/2 bg-primary text-white text-xs px-1 rounded whitespace-nowrap">
              Insert Left
            </div>
          )}
        </div>

        {/* The actual form element */}
        <div className="w-full">{children}</div>

        {/* Right drop zone */}
        <div
          ref={setRightRef}
          className={`absolute right-0 top-0 h-full rounded-r-md transition-all pointer-events-auto ${isOverRight ? 'bg-primary/40' : 'bg-transparent'}`}
          style={{
            zIndex: 20,
            width: '64px',
            opacity: isHoveringRight || isOverRight ? 1 : 0,
            transform: 'translateX(6px)',
          }}
          onMouseEnter={() => setIsHoveringRight(true)}
          onMouseLeave={() => !isOverRight && setIsHoveringRight(false)}
          onClick={onDropRight}
        >
          {isOverRight && (
            <div className="absolute top-1/2 right-0 transform -translate-y-1/2 translate-x-1/2 bg-primary text-white text-xs px-1 rounded whitespace-nowrap">
              Insert Right
            </div>
          )}
        </div>
      </div>

      {/* Bottom drop zone - horizontal indicator */}
      {onDropBottom && (
        <div
          ref={setBottomRef}
          className={`absolute left-0 bottom-0 w-full transition-all pointer-events-auto ${isOverBottom ? 'bg-primary/40' : 'bg-transparent'}`}
          style={{
            zIndex: 20,
            height: '16px',
            transform: 'translateY(4px)',
            opacity: isHoveringBottom || isOverBottom ? 1 : 0,
          }}
          onMouseEnter={() => setIsHoveringBottom(true)}
          onMouseLeave={() => !isOverBottom && setIsHoveringBottom(false)}
          onClick={onDropBottom}
        >
          {isOverBottom && (
            <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-primary text-white text-xs px-1 rounded whitespace-nowrap">
              Insert Below
            </div>
          )}
        </div>
      )}
    </div>
  );
}
