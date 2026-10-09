import React, { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { MainButton } from '@/components/buttons/main-button';
import { cn } from '@/lib/utils';
import { Card, CardContent } from './card';

interface TagsInputProps extends Omit<
  React.ComponentProps<'input'>,
  'value' | 'defaultValue' | 'onChange'
> {
  data?: string[];
  value?: string[];
  onValueChange?: (value: string[]) => void;
}

const TagsInput = React.forwardRef<HTMLInputElement, TagsInputProps>(
  ({ value, onValueChange, data = [], ...props }, ref) => {
    const [tags, setTags] = useState<string[]>(value || []);
    const [inputValue, setInputValue] = useState<string>('');
    const [showSuggestions, setShowSuggestions] = useState(false);
    const [suggestions, setSuggestions] = useState<string[]>(data);

    useEffect(() => {
      if (value) {
        setTags(value);
      }
    }, [value]);

    useEffect(() => {
      setSuggestions(
        data.filter((suggestion) =>
          suggestion.toLowerCase().includes(inputValue.toLowerCase())
        )
      );
    }, [inputValue, data]);

    const handleAddTag = (tag: string) => {
      if (!tags.includes(tag)) {
        const newTags = [...tags, tag];
        setTags(newTags);
        onValueChange?.(newTags);
      }
    };

    const handleRemoveTag = (tag: string) => {
      const newTags = tags.filter((t) => t !== tag);
      setTags(newTags);
      onValueChange?.(newTags);
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      setInputValue(e.currentTarget.value);
      setShowSuggestions(true);
    };

    const handleInputKeydown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      const _inputValue = inputValue.trim();
      if (e.key === 'Enter' && _inputValue) {
        e.preventDefault();
        handleAddTag(_inputValue);
        setInputValue('');
      }
    };

    const handleInputBlur = () => {
      const _inputValue = inputValue.trim();
      if (_inputValue) {
        handleAddTag(_inputValue);
        setInputValue('');
      }
      setShowSuggestions(false);
    };

    const handleInputFocus = () => {
      setShowSuggestions(true);
    };

    return (
      <div className="relative flex w-full flex-wrap gap-2 rounded-md border border-input bg-transparent px-2 py-1 text-base shadow-sm transition-colors focus-within:outline-none focus-within:ring-1 focus-within:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-sm">
        {tags.map((tag) => (
          <span
            key={tag}
            className="animate-fadeIn relative inline-flex h-7 flex-shrink-0 cursor-default items-center rounded-md border-solid bg-[#EEFAFF] pe-7 pl-2 ps-2 text-xs font-medium text-secondary-foreground transition-all hover:bg-[#EEFAFF] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {tag}
            <button
              className="absolute -end-px flex size-7 items-center justify-center whitespace-nowrap rounded-md rounded-e-lg p-0 text-sm font-medium text-muted-foreground/80 outline-0 transition-colors hover:bg-transparent hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring/70 focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
              type="button"
              onClick={() => handleRemoveTag(tag)}
            >
              <X className="h-4 w-4" />
            </button>
          </span>
        ))}
        <input
          ref={ref}
          {...props}
          className="flex-1 p-1 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none"
          type="text"
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleInputKeydown}
          onBlur={handleInputBlur}
          onFocus={handleInputFocus}
        />
        {showSuggestions && (
          <Card className="absolute left-0 top-full mt-2 max-h-40 w-full overflow-y-auto rounded-md">
            <CardContent className="p-1">
              {suggestions.length > 0 ? (
                suggestions.map((suggestion, index) => (
                  <MainButton
                    key={`suggestion-${index}`}
                    variant="ghost"
                    className={cn(
                      'w-full justify-start px-2 text-left transition-all duration-200'
                    )}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => {
                      handleAddTag(suggestion);
                      setInputValue('');
                      setShowSuggestions(false);
                    }}
                    text={suggestion}
                  />
                ))
              ) : (
                <p className="p-2 text-center text-sm text-muted-foreground">
                  No suggestions found
                </p>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    );
  }
);

TagsInput.displayName = 'TagsInput';

export { TagsInput };
