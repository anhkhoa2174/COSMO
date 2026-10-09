import React from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { type BadgeNode } from './index';

export function BadgeNode({ data }: NodeProps<BadgeNode>) {
  return (
    <>
      <Handle id="node2-left" type="target" position={Position.Left} />
      <div className="flex w-[254px] items-center justify-center gap-2 rounded-lg bg-primary p-2 text-primary-foreground shadow">
        {data.icon}
        {data.label}
      </div>
      <Handle
        id="node2-bottom"
        type="source"
        position={Position.Bottom}
        style={{ left: '25%' }}
      />
    </>
  );
}
