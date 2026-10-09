'use client';

import React, { useCallback } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';

import { Plate } from '@udecode/plate/react';
import { deserializeMd } from '@udecode/plate-markdown';

import { useCreateEditor } from '@/components/editor/use-create-editor';
import { Editor, EditorContainer } from '@/components/plate-ui/editor';

interface PlateEditorProps {
  readOnly?: boolean;
  value?: string;
  onChange?: (value: string) => void;
  onFocus?: () => void;
  onBlur?: () => void;
  deps?: any[];
}

export function PlateEditor({
  value = '',
  onChange,
  onFocus,
  onBlur,
  deps,
  readOnly = false,
}: PlateEditorProps) {
  const editor = useCreateEditor(
    { value: (editor) => deserializeMd(editor, value), readOnly },
    deps
  );

  const handleOnChange = useCallback(
    ({ editor }: { editor: any }) => {
      const _value = editor.api.markdown.serialize();
      onChange?.(_value);
    },
    [onChange]
  );

  return (
    <DndProvider backend={HTML5Backend}>
      <Plate editor={editor} onChange={handleOnChange} readOnly={readOnly}>
        <EditorContainer className="relative" data-registry="plate">
          <Editor variant="demo" onFocus={onFocus} onBlur={onBlur} />
        </EditorContainer>
      </Plate>
    </DndProvider>
  );
}
