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
  const { children, label, isCompleted = false, justEdited = false } = data;

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
        'group min-w-[256px] cursor-pointer overflow-hidden rounded-xl border bg-white shadow-sm transition-all duration-200',
        'hover:-translate-y-0.5 hover:border-violet-300 hover:shadow-lg',
        // A ring instead of a thicker top border: widening the border shifts
        // the node's contents and makes the whole canvas jump on select.
        selected && 'border-violet-400 shadow-lg ring-2 ring-violet-200',
        // Amber, not violet: it must read as "this one changed", distinct from
        // "this one is selected", because both can be true at once.
        justEdited &&
          !selected &&
          'border-amber-400 ring-2 ring-amber-200'
      )}
    >
      <div className="flex items-center justify-between border-b bg-gradient-to-r from-violet-50 to-indigo-50 p-2.5">
        <div className="flex items-center gap-2">
          <span className="grid size-7 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 text-white">
            <Mail className="h-4 w-4" />
          </span>
          <p className="font-medium">{label}</p>
          {justEdited && (
            <span className="rounded-full bg-amber-100 px-1.5 py-0.5 text-[0.6rem] font-semibold uppercase tracking-wide text-amber-700">
              Edited
            </span>
          )}
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
