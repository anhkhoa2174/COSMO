import React from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { type BadgeNode } from './index';

export function BadgeNode({ data }: NodeProps<BadgeNode>) {
  return (
    <>
      <Handle id="node2-left" type="target" position={Position.Left} />
      <div className="bg-primary flex justify-center items-center gap-2 w-[254px] text-primary-foreground p-2 rounded-lg shadow">
        {data.icon}
        {data.label}
      </div>
      <Handle id="node2-bottom" type="source" position={Position.Bottom} style={{ left: '25%' }} />
    </>
  );
}
