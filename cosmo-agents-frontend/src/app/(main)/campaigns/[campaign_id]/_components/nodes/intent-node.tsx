import { cn } from '@/lib/utils';
import { Handle, Position, type NodeProps } from '@xyflow/react';

import { type IntentNode } from './index';

export function IntentNode({ data, selected }: NodeProps<IntentNode>) {
  const { label } = data;

  return (
    <>
      <Handle
        id="intent-node-handle-left"
        type="target"
        position={Position.Left}
      />
      <div
        className={cn(
          'w-[256px] cursor-pointer rounded-lg bg-zinc-100 p-2 shadow',
          selected
            ? 'border border-t-4 border-accent-foreground'
            : 'hover:border hover:border-accent-foreground'
        )}
      >
        {label}
      </div>
      <Handle
        id="intent-node-handle-right"
        type="source"
        position={Position.Right}
      />
    </>
  );
}
