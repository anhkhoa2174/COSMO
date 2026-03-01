'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useDraggable } from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';
import {
  AlignLeft,
  ArrowDownUp,
  Calendar,
  CheckSquare,
  Hash,
  Link,
  ListFilter,
  Lock,
  Mail,
  Minus,
  Type
} from 'lucide-react';
import React, { useEffect, useRef } from 'react';
import { FormElementType } from './types';

interface ElementButtonProps {
  type: FormElementType;
  label: string;
  icon: React.ReactNode;
}

const DraggableElementButton = ({ type, label, icon }: ElementButtonProps) => {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `toolbox-${type}`,
    data: {
      type: 'toolbox-item',
      elementType: type,
      isToolboxItem: true
    },
  });

  const style = transform ? {
    transform: CSS.Transform.toString(transform),
    zIndex: isDragging ? 1000 : 1,
    opacity: isDragging ? 0 : 1,
    boxShadow: isDragging ? '0 5px 15px rgba(0, 0, 0, 0.15)' : 'none',
  } : {
    opacity: isDragging ? 0 : 1,
    zIndex: isDragging ? 1000 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      className="cursor-grab"
      style={style}
    >
      <div className="flex items-center p-3 mb-2 border rounded-md bg-background hover:bg-accent hover:text-accent-foreground transition-colors">
        <div className="mr-2 h-5 w-5">{icon}</div>
        <span>{label}</span>
      </div>
    </div>
  );
};

export default function FormElementsToolbox({ setHeightToolbox }: { setHeightToolbox: (height: string) => void }) {
  const heightComponent = useRef<HTMLDivElement>(null);
  const elements: ElementButtonProps[] = [
    { type: 'text', label: 'Text Input', icon: <Type size={18} /> },
    { type: 'email', label: 'Email', icon: <Mail size={18} /> },
    { type: 'password', label: 'Password', icon: <Lock size={18} /> },
    { type: 'number', label: 'Number', icon: <Hash size={18} /> },
    { type: 'textarea', label: 'Text Area', icon: <AlignLeft size={18} /> },
    { type: 'select', label: 'Select', icon: <ListFilter size={18} /> },
    { type: 'checkbox', label: 'Checkbox', icon: <CheckSquare size={18} /> },
    { type: 'radio', label: 'Radio Group', icon: <ListFilter size={18} /> },
    { type: 'date', label: 'Date', icon: <Calendar size={18} /> },
    { type: 'spacer', label: 'Spacer', icon: <ArrowDownUp size={18} /> },
    { type: 'divider', label: 'Divider', icon: <Minus size={18} /> },
    { type: 'url', label: 'Url', icon: <Link size={18} /> },
  ];

  useEffect(() => {
    if (heightComponent.current) {
      const height = heightComponent.current.offsetHeight;
      setHeightToolbox?.(`${height}px`);
    }
  }, []);

  return (
    <Card ref={heightComponent}>
      <CardHeader className='p-0 px-4 pt-4 pb-3 border-b bg-[#F1F5F9] rounded-t-lg'>
        <CardTitle>Form Elements</CardTitle>
      </CardHeader>
      <CardContent className='p-2 '>
        <div className="space-y-1">
          {elements.map((element) => (
            <DraggableElementButton
              key={element.type}
              type={element.type}
              label={element.label}
              icon={element.icon}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
