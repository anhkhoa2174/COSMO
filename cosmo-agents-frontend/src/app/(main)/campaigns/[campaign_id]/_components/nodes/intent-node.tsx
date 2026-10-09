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
          'w-[256px] cursor-pointer rounded-xl border bg-white p-2.5 shadow-sm transition-all duration-200',
          'hover:-translate-y-0.5 hover:border-violet-300 hover:shadow-md',
          selected && 'border-violet-400 shadow-md ring-2 ring-violet-200'
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
