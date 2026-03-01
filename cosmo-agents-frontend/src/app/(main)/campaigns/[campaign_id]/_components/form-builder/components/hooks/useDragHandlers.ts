import {
  Active,
  DragEndEvent,
  DragOverEvent,
  DragStartEvent,
  Over,
} from '@dnd-kit/core';
import { arrayMove } from '@dnd-kit/sortable';
import { useState } from 'react';
import { FormElement, FormElementType, FormState } from '../types';
import { createFormElement } from '../utils/form-element-utils';
import {
  calculateHorizontalSpans,
  findRowBoundary,
  optimizeRowLayouts,
} from '../utils/layout-utils';

interface DragElementData {
  type:
    | 'form-element'
    | 'toolbox-item'
    | `spot-${'left' | 'right' | 'top' | 'bottom'}`;
  elementType?: FormElementType;
  elementId?: string;
  id?: string;
  name?: string;
  label?: string;
  required?: boolean;
  options?: { label: string; value: string }[];
  defaultValue?: string;
}

type DragEventElement = Active | Over;

const hasValidDragData = (
  data: unknown
): data is { current?: DragElementData } => {
  return typeof data === 'object' && data !== null;
};

const getDragData = (
  element: DragEventElement
): DragElementData | undefined => {
  return hasValidDragData(element.data) ? element.data.current : undefined;
};

export const useDragHandlers = (
  formState: FormState,
  setFormState: React.Dispatch<React.SetStateAction<FormState>>,
  setSelectedElement: React.Dispatch<React.SetStateAction<FormElement | null>>
) => {
  const [activeDragElement, setActiveDragElement] =
    useState<FormElement | null>(null);

  const createNewElement = (type: FormElementType, item: DragElementData) => {
    const newItem = {
      id: item.id,
      type: item.elementType,
      name: item.name,
      label: item.label,
      required: item.required,
      options: item.options,
      defaultValue: item.defaultValue || null,
    };
    const newElement = createFormElement(type, newItem as Partial<FormElement>);
    return newElement || null;
  };

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event;

    // Handle dragging existing form elements
    const activeElement = formState.elements.find((el) => el.id === active.id);
    if (activeElement) {
      setActiveDragElement(activeElement);
      return;
    }

    // Handle dragging from toolbox
    const dragData = getDragData(active);
    if (dragData?.type === 'toolbox-item') {
      const elementType = dragData.elementType as FormElementType;
      const dummyElement = createNewElement(elementType, dragData);
      setActiveDragElement(dummyElement);
    }
  };

  const handleDragOver = (event: DragOverEvent) => {
    // Add any drag over logic here if needed
  };

  const handleDragEnd = (event: DragEndEvent) => {
    try {
      setActiveDragElement(null);
      const { active, over } = event;

      // Skip if no valid drop target
      if (!over) {
        return;
      }

      // Get drag data with proper type checking
      const dragData = getDragData(active);
      if (!dragData) {
        console.warn('No valid drag data found');
        return;
      }

      // Handle different drop scenarios
      if (dragData.type === 'toolbox-item') {
        handleToolboxItemDrop(active, over);
      } else if (dragData.type === 'form-element') {
        handleFormElementDrop(active, over);
      }
    } catch (error) {
      console.error('Drag operation failed:', error);
      setActiveDragElement(null);
    }
  };

  const handleToolboxItemDrop = (active: Active | Over, over: Over) => {
    const dragData = getDragData(active);
    if (!dragData) return;

    const type = dragData.elementType as FormElementType;
    const newElement = createNewElement(type, dragData);

    if (over.id === 'canvas') {
      // Drop onto canvas
      setFormState((prev) => ({
        ...prev,
        elements: [...prev.elements, newElement],
      }));
      // setSelectedElement(newElement);
    } else {
      const overData = getDragData(over);
      if (!overData) return;
      if (overData.type?.startsWith('spot-')) {
        // Drop onto spot zones
        handleSpotZoneDrop(over, newElement);
      } else if (overData.type === 'form-element') {
        // Drop onto existing element (replace)
        setFormState((prev) => {
          const targetIndex = prev.elements.findIndex(
            (el) => el.id === over.id
          );
          const newElements = [...prev.elements];
          newElements[targetIndex] = newElement;
          return {
            ...prev,
            elements: newElements,
          };
        });
        // setSelectedElement(newElement);
      }
    }
  };

  const handleFormElementDrop = (active: Active, over: Over) => {
    const activeData = getDragData(active);
    const overData = getDragData(over);
    if (
      activeData?.type === 'form-element' &&
      overData?.type === 'form-element' &&
      active.id !== over.id
    ) {
      setFormState((prev) => {
        const oldIndex = prev.elements.findIndex((el) => el.id === active.id);
        const newIndex = prev.elements.findIndex((el) => el.id === over.id);

        return {
          ...prev,
          elements: arrayMove(prev.elements, oldIndex, newIndex),
        };
      });
    }
  };

  const handleSpotZoneDrop = (over: Over, newElement: FormElement) => {
    const dragData = getDragData(over);
    if (!dragData?.elementId || !dragData.type?.startsWith('spot-')) return;

    const targetElementId = dragData.elementId;
    const spotType = dragData.type.replace('spot-', '') as
      | 'left'
      | 'right'
      | 'top'
      | 'bottom';

    setFormState((prev) => {
      const targetIndex = prev.elements.findIndex(
        (el) => el.id === targetElementId
      );
      if (targetIndex === -1) return prev;

      const targetElement = prev.elements[targetIndex];
      const gridColumns = Number(prev.settings.gridColumns) || 4;
      const newElements = [...prev.elements];
      let insertIndex: number;

      if (spotType === 'left' || spotType === 'right') {
        const { leftColSpan, rightColSpan } = calculateHorizontalSpans(
          targetElement.colSpan || 1,
          gridColumns
        );

        newElements[targetIndex] = {
          ...targetElement,
          colSpan: leftColSpan,
        };

        newElement.colSpan = rightColSpan;
        insertIndex = spotType === 'left' ? targetIndex : targetIndex + 1;
      } else {
        insertIndex = findRowBoundary(
          prev.elements,
          targetIndex,
          spotType,
          gridColumns
        );
        newElement.colSpan = Math.min(4, gridColumns) as 1 | 2 | 3 | 4;
      }

      newElements.splice(insertIndex, 0, newElement);
      optimizeRowLayouts(newElements, gridColumns);

      return {
        ...prev,
        elements: newElements,
      };
    });
    // setSelectedElement(newElement);
  };

  return {
    activeDragElement,
    handleDragStart,
    handleDragOver,
    handleDragEnd,
  };
};
