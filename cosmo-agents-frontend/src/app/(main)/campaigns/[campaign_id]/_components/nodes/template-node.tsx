'use client';

import React, { useRef, useState } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { type TemplateNode } from './index';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ChevronDown, Mail, Trash } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';

export function TemplateNode({ id, data, selected }: NodeProps<TemplateNode>) {
  const { children, label, isCompleted = false } = data;

  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Handle node click without opening dropdown
  const handleNodeClick = (e: React.MouseEvent) => {
    // Check if click is on the dropdown menu or its trigger
    if (dropdownRef.current && dropdownRef.current.contains(e.target as Node)) {
      e.stopPropagation(); // Prevent node selection when clicking dropdown
    }
  };

  // Handle delete action
  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent drawer from opening
    // if (data.deleteNode) {
    //   data.deleteNode(id);
    // }
    setIsMenuOpen(false);
  };

  return (
    <div
      onClick={handleNodeClick}
      className={cn(
        'min-w-[256px] cursor-pointer overflow-hidden rounded-lg border bg-white shadow hover:border-accent-foreground',
        selected && 'border-t-4 border-accent-foreground'
      )}
    >
      <div className="flex items-center justify-between bg-zinc-100 p-2">
        <div className="flex items-center gap-2">
          <span className="text-blue-500">
            <Mail className="h-5 w-5" />
          </span>
          <p className="font-medium">{label}</p>
        </div>
        {/* {!['Entry Rules', 'Outreach Email'].includes(data.label) && (
          <div ref={dropdownRef} onClick={(e) => e.stopPropagation()}>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button className="rounded p-1 hover:bg-zinc-50">
                  <ChevronDown className="h-4 w-4" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem
                  className="text-destructive focus:bg-destructive/10 focus:text-destructive"
                  onClick={handleDelete}
                >
                  <Trash /> Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )} */}
      </div>
      <div className="flex items-center justify-between p-2">
        <div className="text-muted-foreground">{children}</div>
        {!isCompleted && (
          <Tooltip>
            <TooltipTrigger>
              <div className="flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full bg-yellow-100 p-1 font-bold text-yellow-600">
                !
              </div>
            </TooltipTrigger>
            <TooltipContent>Incomplete settings</TooltipContent>
          </Tooltip>
        )}
      </div>
      <Handle
        id="email-node-handle-left"
        type="target"
        position={Position.Left}
        style={{ top: 44 }}
      />
    </div>
  );
}
