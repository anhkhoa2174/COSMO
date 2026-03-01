import React, { useState, useRef, useEffect } from 'react';
import { Pencil, Check, X } from 'lucide-react';
import { Input } from './ui/input';
import { MainButton } from './buttons/main-button';
import { cn } from '@/lib/utils';

// Editable Component
interface EditableProps {
  value: string | number;
  onSubmit: (newValue: string | number) => void;
  type?: 'text' | 'number';
  placeholder?: string;
  className?: string;
  href?: string;
}

const Editable: React.FC<EditableProps> = ({
  value,
  onSubmit,
  type = 'text',
  placeholder = 'Enter value',
  className = '',
  href,
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(String(value));
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    // Reset edit value when value prop changes
    setEditValue(String(value));
  }, [value]);

  const handleEdit = () => {
    setIsEditing(true);
    // Delay focus to ensure input is rendered
    setTimeout(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    }, 0);
  };

  const handleSubmit = () => {
    // Trim and validate input
    const trimmedValue = editValue.trim();
    if (trimmedValue !== '') {
      onSubmit(type === 'number' ? Number(trimmedValue) : trimmedValue);
    }
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditValue(String(value));
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSubmit();
    } else if (e.key === 'Escape') {
      handleCancel();
    }
  };

  if (isEditing) {
    return (
      <div className={`flex items-center space-x-2 ${className}`}>
        <Input
          ref={inputRef}
          type={type}
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
        />
        <MainButton
          icon={Check}
          onClick={handleSubmit}
          size="icon"
          variant="success-secondary"
        />
        <MainButton
          icon={X}
          size="icon"
          onClick={handleCancel}
          variant="destructive-secondary"
        />
      </div>
    );
  }

  return (
    <div className={`flex items-center`}>
      {href ? (
        <a className={cn('mr-2', className)} href={href}>
          {value}
        </a>
      ) : (
        <span className={cn('mr-2', className)}>{value}</span>
      )}
      <MainButton
        icon={Pencil}
        size="icon"
        onClick={handleEdit}
        variant="ghost"
      />
    </div>
  );
};

export { Editable };
