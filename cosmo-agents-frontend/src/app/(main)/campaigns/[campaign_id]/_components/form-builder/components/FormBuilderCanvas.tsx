'use client';

import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import {
  DndContext,
  DragOverlay,
  MouseSensor,
  TouchSensor,
  closestCenter,
  useDroppable,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import {
  AlignLeft,
  ArrowDownUp,
  Calendar,
  CheckSquare,
  Copy,
  Hash,
  Link,
  ListFilter,
  Lock,
  Mail,
  Minus,
  Trash2,
  Type,
} from 'lucide-react';
import { v4 as uuidv4 } from 'uuid';
import SpotItem from './SpotItem';
import { FormElement, FormState } from './types';
import { findRowBoundary, optimizeRowLayouts } from './utils/layout-utils';

// Function to get the icon based on element type
export const getElementIcon = (element: FormElement) => {
  switch (element.type) {
    case 'text':
      return <Type size={18} />;
    case 'email':
      return <Mail size={18} />;
    case 'password':
      return <Lock size={18} />;
    case 'number':
      return <Hash size={18} />;
    case 'textarea':
      return <AlignLeft size={18} />;
    case 'select':
    case 'radio':
      return <ListFilter size={18} />;
    case 'checkbox':
      return <CheckSquare size={18} />;
    case 'date':
      return <Calendar size={18} />;
    case 'spacer':
      return <ArrowDownUp size={18} />;
    case 'divider':
      return <Minus size={18} />;
    case 'url':
      return <Link size={18} />;
    default:
      return <Type size={18} />;
  }
};

interface SortableElementProps {
  element: FormElement;
  isSelected: boolean;
  onSelect: () => void;
  onRemove: () => void;
  onDuplicate: () => void;
  gridColumns: number | string;
}

function SortableElement({
  gridColumns,
  element,
  isSelected,
  onSelect,
  onRemove,
  onDuplicate,
}: SortableElementProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: element.id,
    data: {
      type: 'form-element',
      element: element,
    },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0 : 1,
    position: 'relative' as const,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`relative flex items-center gap-2 ${isSelected ? 'ring-offset-2' : ''}`}
      onClick={() => onSelect()}
    >
      <Card
        {...attributes}
        {...listeners}
        className={`${isDragging ? 'cursor-grabbing' : 'cursor-grab'} group flex-1 p-3 ${isSelected ? 'ring-2 ring-primary' : ''}`}
      >
        <div className="flex items-center justify-between">
          <div className="z-30 flex items-center space-x-2">
            <div className="h-5 w-5 text-primary">
              {getElementIcon(element)}
            </div>
            <span className={cn('font-medium capitalize', element.colSpan === 1 && gridColumns === 4 ? 'text-[8px]' : '')}>{
              element.label
            }
            </span>
          </div>
          <div className="z-30 flex items-center space-x-1 opacity-0 transition-opacity duration-200 group-hover:opacity-100">
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={(e) => {
                e.stopPropagation();
                onDuplicate();
              }}
            >
              <Copy size={14} />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 text-destructive"
              onClick={(e) => {
                e.stopPropagation();
                onRemove();
              }}
            >
              <Trash2 size={14} />
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}

interface FormBuilderCanvasProps {
  formState: FormState;
  onChange: (
    formStateOrUpdater: FormState | ((prev: FormState) => FormState)
  ) => void;
  onSelectElement: (element: FormElement | null) => void;
  selectedElementId: string | null;
  useExternalDndContext?: boolean;
  draggedOverlayElement: FormElement | null;
  heightToolbox?: string;
}

export default function FormBuilderCanvas({
  formState,
  onChange,
  onSelectElement,
  selectedElementId,
  useExternalDndContext = false,
  draggedOverlayElement,
  heightToolbox = '400px',
}: FormBuilderCanvasProps) {
  const sensors = useSensors(
    useSensor(MouseSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(TouchSensor, {
      activationConstraint: {
        delay: 200,
        tolerance: 5,
      },
    })
  );

  const handleSelectElement = (element: FormElement) => {
    onSelectElement(element);
  };

  const handleRemoveElement = (elementId: string) => {
    const newElements = formState.elements.filter((el) => el.id !== elementId);
    onChange((prev) => ({
      ...prev,
      elements: newElements,
    }));

    // If the removed element was selected, deselect it
    if (selectedElementId === elementId) {
      onSelectElement(null);
    }
  };

  const handleDuplicateElement = (elementId: string) => {
    const elementToDuplicate = formState.elements.find(
      (el) => el.id === elementId
    );
    if (!elementToDuplicate) return;

    const duplicatedElement = {
      ...elementToDuplicate,
      id: uuidv4(),
      label: `${elementToDuplicate.label} (Copy)`,
    };

    const elementIndex = formState.elements.findIndex(
      (el) => el.id === elementId
    );
    const newElements = [...formState.elements];
    newElements.splice(elementIndex + 1, 0, duplicatedElement);

    onChange((prev) => ({
      ...prev,
      elements: newElements,
    }));

    // Select the new duplicated element
    onSelectElement(duplicatedElement);
  };

  // Set up the droppable area for the canvas
  const { setNodeRef: setCanvasRef, isOver: isOverCanvas } = useDroppable({
    id: 'canvas',
    data: {
      type: 'canvas',
      accepts: ['toolbox-item', 'form-element'],
    },
  });
  // Handle drop on any side of an element (left, right, top, bottom)
  const handleDropOnSpot = (
    elementId: string,
    position: 'left' | 'right' | 'top' | 'bottom',
    newElement?: FormElement
  ) => {
    // Find the index of the element we're dropping next to
    const elementIndex = formState.elements.findIndex(
      (el) => el.id === elementId
    );
    if (elementIndex === -1 || !newElement) return;

    // Get grid columns setting
    const gridColumns = Number(formState.settings.gridColumns) || 4;

    // Create a copy of the elements array
    const newElements = [...formState.elements];
    let newIndex: number;

    // Handle different drop positions
    switch (position) {
      case 'left':
      case 'right':
        // For left/right positions, adjust column spans and insert at appropriate index
        newIndex = position === 'left' ? elementIndex : elementIndex + 1;

        // Handle column span adjustments for horizontal placement
        handleHorizontalPlacement(
          newElements,
          elementIndex,
          newElement,
          gridColumns
        );
        break;

      case 'top':
      case 'bottom':
        // For top/bottom positions, find row boundaries and insert at row start/end
        newIndex = findRowBoundary(
          formState.elements,
          elementIndex,
          position,
          gridColumns
        );

        // Set full width for vertical placements
        newElement.colSpan = Math.min(4, gridColumns) as 1 | 2 | 3 | 4;
        break;
    }

    // Insert the new element at the calculated position
    newElements.splice(newIndex, 0, newElement);

    // Optimize row layouts after insertion
    optimizeRowLayouts(newElements, gridColumns);

    // Update the form state with the new elements
    onChange({
      ...formState,
      elements: newElements,
    });
  };

  // Helper function to handle horizontal placement (left/right)
  const handleHorizontalPlacement = (
    elements: FormElement[],
    targetIndex: number,
    newElement: FormElement,
    gridColumns: number
  ) => {
    const targetElement = elements[targetIndex];
    const targetColSpan = targetElement.colSpan || 1;
    let newTargetColSpan: 1 | 2 | 3 | 4 = 1;
    let newElementColSpan: 1 | 2 | 3 | 4 = 1;

    // Calculate optimal column spans based on target element's current span
    if (targetColSpan === gridColumns) {
      // If target takes full width, split evenly or proportionally
      if (gridColumns % 2 === 0) {
        // Even split for even grid columns
        const halfGrid = gridColumns / 2;
        newTargetColSpan = Math.min(4, halfGrid) as 1 | 2 | 3 | 4;
        newElementColSpan = Math.min(4, halfGrid) as 1 | 2 | 3 | 4;
      } else {
        // For odd grid columns, give more to the original element
        newTargetColSpan = Math.min(4, Math.ceil(gridColumns / 2)) as
          | 1
          | 2
          | 3
          | 4;
        newElementColSpan = Math.min(4, Math.floor(gridColumns / 2)) as
          | 1
          | 2
          | 3
          | 4;
      }
    } else if (targetColSpan > 1) {
      // If target has multiple columns but not full width, split proportionally
      newTargetColSpan = Math.max(1, Math.floor(targetColSpan / 2)) as
        | 1
        | 2
        | 3
        | 4;
      newElementColSpan = Math.max(
        1,
        Math.min(4, targetColSpan - newTargetColSpan)
      ) as 1 | 2 | 3 | 4;
    }

    // Update the target element's column span
    elements[targetIndex] = {
      ...targetElement,
      colSpan: newTargetColSpan,
    };

    // Update the new element's column span
    newElement.colSpan = newElementColSpan;
  };

  // Render the canvas content
  const renderCanvasContent = () => (
    <div
      ref={setCanvasRef}
      id="canvas"
      className={`rounded-lg bg-muted/30 p-4 transition-colors ${isOverCanvas ? 'border-2 border-dashed border-primary bg-primary/10' : ''}`}
      style={{ minHeight: heightToolbox }}
    >
      {formState.elements.length === 0 ? (
        <div className="flex h-[300px] flex-col items-center justify-center text-muted-foreground">
          <div className="mb-4 opacity-50">
            <svg
              width="48"
              height="48"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
              <line x1="12" y1="3" x2="12" y2="21"></line>
              <path d="M3 12h18"></path>
            </svg>
          </div>
          <p>Drag and drop form elements here</p>
          <p className="mt-2 text-sm">
            Components will appear in the form layout
          </p>
        </div>
      ) : (
        <SortableContext
          items={formState.elements.map((el) => el.id)}
          strategy={verticalListSortingStrategy}
        >
          <div
            className={`grid grid-cols-${formState.settings.gridColumns || 4} gap-3`}
          >
            {formState.elements.map((element) => (
              <div
                key={element.id}
                className={`col-span-${element.colSpan || 1}`}
                style={{ gridColumn: `span ${element.colSpan || 1}` }}
              >
                <SpotItem
                  id={element.id}
                  onDropLeft={() =>
                    handleDropOnSpot(
                      element.id,
                      'left',
                      draggedOverlayElement || undefined
                    )
                  }
                  onDropRight={() =>
                    handleDropOnSpot(
                      element.id,
                      'right',
                      draggedOverlayElement || undefined
                    )
                  }
                  onDropTop={() =>
                    handleDropOnSpot(
                      element.id,
                      'top',
                      draggedOverlayElement || undefined
                    )
                  }
                  onDropBottom={() =>
                    handleDropOnSpot(
                      element.id,
                      'bottom',
                      draggedOverlayElement || undefined
                    )
                  }
                >
                  <SortableElement
                    gridColumns={formState?.settings?.gridColumns || 4}
                    element={element}
                    isSelected={selectedElementId === element.id}
                    onSelect={() => handleSelectElement(element)}
                    onRemove={() => handleRemoveElement(element.id)}
                    onDuplicate={() => handleDuplicateElement(element.id)}
                  />
                </SpotItem>
              </div>
            ))}
          </div>
        </SortableContext>
      )}
    </div>
  );

  // Render with or without DndContext based on the prop
  return useExternalDndContext ? (
    <>
      {renderCanvasContent()}
      <DragOverlay>
        {draggedOverlayElement && ( // Use prop for overlay
          <div className="w-full opacity-80">
            {/* Custom compact rendering for DragOverlay */}
            <Card className="flex items-center bg-background p-3 shadow-lg">
              <div className="mr-2 flex h-5 w-5 items-center justify-center">
                {getElementIcon(draggedOverlayElement)}
              </div>
              <span>
                {draggedOverlayElement.label || `New ${draggedOverlayElement}`}
              </span>
            </Card>
          </div>
        )}
      </DragOverlay>
    </>
  ) : (
    <DndContext sensors={sensors} collisionDetection={closestCenter}>
      {renderCanvasContent()}

      <DragOverlay>
        {draggedOverlayElement && ( // Use prop for overlay, though this path is not taken in current setup
          <div className="w-full opacity-80">
            {/* Custom compact rendering for DragOverlay */}
            <Card className="flex items-center bg-background p-3 shadow-lg">
              <div className="mr-2 flex h-5 w-5 items-center justify-center">
                {getElementIcon(draggedOverlayElement)}
              </div>
              <span>
                {draggedOverlayElement.label || `New ${draggedOverlayElement}`}
              </span>
            </Card>
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
}
