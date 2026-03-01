import { FormElement } from '../types';

// Helper function to calculate column spans for horizontal placement
export const calculateHorizontalSpans = (
  targetColSpan: number,
  gridColumns: number
) => {
  const leftColSpan = Math.floor((gridColumns - targetColSpan) / 2);
  const rightColSpan = gridColumns - targetColSpan - leftColSpan;
  return { leftColSpan, rightColSpan };
};

// Helper function to find row boundaries for top/bottom placement
export const findRowBoundary = (
  elements: FormElement[],
  targetIndex: number,
  position: 'top' | 'bottom',
  gridColumns: number
): number => {
  if (elements.length === 0) return 0;
  if (targetIndex < 0) return 0;
  if (targetIndex >= elements.length) return elements.length;

  let currentRowSpan = 0;
  let currentRow = 0;
  const rowElements: { [key: number]: number[] } = {};

  // First pass: determine which row each element belongs to
  elements.forEach((el, index) => {
    const colSpan = el.colSpan || 1;

    // If adding this element would exceed grid width, move to next row
    if (currentRowSpan + colSpan > gridColumns) {
      currentRow++;
      currentRowSpan = colSpan;
    } else {
      currentRowSpan += colSpan;
    }

    // Add element index to the current row
    if (!rowElements[currentRow]) {
      rowElements[currentRow] = [];
    }
    rowElements[currentRow].push(index);

    // If we reach exactly the grid width, reset for next row
    if (currentRowSpan === gridColumns) {
      currentRow++;
      currentRowSpan = 0;
    }
  });

  // Find which row the target element is in
  let targetRow = -1;
  for (const [row, elementsInRow] of Object.entries(rowElements)) {
    if (elementsInRow.includes(targetIndex)) {
      targetRow = parseInt(row);
      break;
    }
  }

  if (targetRow === -1) {
    // Element not found in any row (shouldn't happen)
    return position === 'top' ? targetIndex : targetIndex + 1;
  }

  if (position === 'top') {
    // Find the first element in the target row
    return rowElements[targetRow][0];
  } else {
    // Find the last element in the target row
    const lastInRow = rowElements[targetRow][rowElements[targetRow].length - 1];
    return lastInRow + 1;
  }
};

// Helper function to optimize row layouts after insertion
export const optimizeRowLayouts = (
  elements: FormElement[],
  gridColumns: number
) => {
  let currentRowSpan = 0;
  let currentRow = 0;
  const rowElements: { [key: number]: number[] } = {};

  // First pass: determine which row each element belongs to
  elements.forEach((el, index) => {
    const colSpan = el.colSpan || 1;

    // If adding this element would exceed grid width, move to next row
    if (currentRowSpan + colSpan > gridColumns) {
      currentRow++;
      currentRowSpan = colSpan;
    } else {
      currentRowSpan += colSpan;
    }

    // Add element index to the current row
    if (!rowElements[currentRow]) {
      rowElements[currentRow] = [];
    }
    rowElements[currentRow].push(index);

    // If we reach exactly the grid width, reset for next row
    if (currentRowSpan === gridColumns) {
      currentRow++;
      currentRowSpan = 0;
    }
  });

  // Second pass: adjust column spans for elements in rows that don't fill the grid
  Object.keys(rowElements).forEach((rowKey) => {
    const row = Number(rowKey);
    const elementsInRow = rowElements[row];
    let rowTotalSpan = 0;

    // Calculate current total span for the row
    elementsInRow.forEach((elementIndex) => {
      rowTotalSpan += elements[elementIndex].colSpan || 1;
    });

    // If row doesn't fill grid and has multiple elements, adjust spans
    if (rowTotalSpan < gridColumns && elementsInRow.length > 1) {
      const extraSpace = gridColumns - rowTotalSpan;
      const elementsToAdjust = Math.min(elementsInRow.length, extraSpace);

      // Distribute extra space among elements
      for (let i = 0; i < elementsToAdjust; i++) {
        const elementIndex = elementsInRow[i];
        const element = elements[elementIndex];
        const currentColSpan = element.colSpan || 1;

        // Only increase if we can stay within the valid range (1-4)
        if (currentColSpan < 4) {
          // Calculate new span and ensure it's one of the valid types
          const newColSpan = Math.min(4, currentColSpan + 1) as 1 | 2 | 3 | 4;

          elements[elementIndex] = {
            ...element,
            colSpan: newColSpan,
          };
        }
      }
    }
  });
};
