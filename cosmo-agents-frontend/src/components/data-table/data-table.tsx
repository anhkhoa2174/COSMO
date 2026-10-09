'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn } from '@/lib/utils';
import { currentPageAtom, selectedIdAtom } from '@/stores/atom';
import { Checkbox, Pagination } from '@mantine/core';
import { useAtom } from 'jotai';
import { ChevronDown, ChevronUp, SearchIcon } from 'lucide-react';
import React, { useEffect, useRef, useState } from 'react';
import { DataTableFilterList } from './data-table-filter-list';

interface ColumnDef<T = any> {
  accessorKey: keyof T | string;
  sortable?: boolean;
  filterable?: boolean;
  header?: string | JSX.Element;
  cell?: ({ row }: { row: T }) => string | JSX.Element;
}

interface CurrentPage {
  [key: string]: number;
}

interface ColumnSortState {
  id: ColumnDef['accessorKey'];
  direction: 'asc' | 'desc';
}

interface DataTableProps<T = any> {
  data: T[];
  columns: ColumnDef[];
  pageSize?: number;
  pageCount: number;
  rowSelectorExtraContent?: React.ReactNode;
  pageKey: string;
  loading?: boolean;
  showSelection?: boolean;
  onPageChange?: (page: number) => void;
  onSearchChange?: (term: string) => void;
  onFilterChange?: (filter: any) => void;
  filterIdButton?: string;
  headerExtraContent?: React.ReactNode;
  filterFieldSelections?: Record<ColumnDef['accessorKey'], string[]>;
  className?: string;
  isScrollViewport?: boolean;
  renderExpandedRow?: (row: T) => React.ReactNode;
  getExpandableIndicator?: (row: T) => boolean;
}

const DataTable = ({
  data,
  columns,
  pageSize = 25,
  pageCount,
  rowSelectorExtraContent,
  showSelection = true,
  pageKey,
  loading,
  onSearchChange,
  onFilterChange,
  headerExtraContent,
  filterIdButton,
  filterFieldSelections,
  className,
  isScrollViewport,
  renderExpandedRow,
  getExpandableIndicator,
}: DataTableProps) => {
  const [sortState, setSortState] = useState<ColumnSortState>({
    id: '',
    direction: 'asc',
  });
  const [currentPage, setCurrentPage] = useAtom(currentPageAtom);
  const [selectedId, setSelectedId] = useAtom(selectedIdAtom);
  const [headerTableHeight, setHeaderTableHeight] = useState<number>(40);
  const headerTableRef = useRef<HTMLTableSectionElement | null>(null);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  // Helper to clean Go map format strings
  const cleanValue = (value: any): string => {
    if (typeof value === 'string' && value.includes('map[')) {
      return ''; // Hide Go map format strings
    }
    return String(value || '');
  };

  const sortedData = [...data].sort((a, b) => {
    if (sortState.id) {
      const aVal = String(a[sortState.id]);
      const bVal = String(b[sortState.id]);
      return sortState.direction === 'asc'
        ? aVal.localeCompare(bVal)
        : bVal.localeCompare(aVal);
    }
    return 0;
  });

  const totalPages = Math.ceil(pageCount / pageSize);

  const page = currentPage[pageKey] || 1;

  const handlePageChange = (newPage: number) => {
    setCurrentPage((prev: CurrentPage) => ({ ...prev, [pageKey]: newPage }));
  };

  const handleSearchChange = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const term = (e.target as HTMLInputElement).value;
      if (onSearchChange) {
        onSearchChange(term);
      }
    }
  };

  const toggleSelectItem = (id: string) => {
    setSelectedId((prevIds: { [key: string]: string[] }) => {
      const currentSelectedIds = prevIds[pageKey] || [];
      if (currentSelectedIds.includes(id)) {
        return {
          ...prevIds,
          [pageKey]: currentSelectedIds.filter((itemId) => itemId !== id),
        };
      }
      return { ...prevIds, [pageKey]: [...currentSelectedIds, id] };
    });
  };

  const toggleSelectAll = (checked: boolean, key: string) => {
    setSelectedId((prevIds: { [key: string]: string[] }) => {
      if (checked) {
        return {
          ...prevIds,
          [key]: [
            ...Array.from(
              new Set([
                ...(prevIds[key] || []),
                ...sortedData.map((item) => String(item.id)),
              ])
            ),
          ],
        };
      }
      return {
        ...prevIds,
        [key]: (prevIds[key] || []).filter(
          (itemId: string) =>
            !sortedData.some((item) => String(item.id) === itemId)
        ),
      };
    });
  };

  const handleSort = (column_id: ColumnDef['accessorKey']) => {
    if (!sortState) {
      return;
    }

    if (sortState.id === column_id) {
      if (sortState.direction === 'desc') {
        setSortState({ id: '', direction: 'asc' });
      } else {
        setSortState((prev) => ({
          ...prev,
          direction: prev.direction === 'asc' ? 'desc' : 'asc',
        }));
      }
    } else {
      setSortState({ id: column_id, direction: 'asc' });
    }
  };

  useEffect(() => {
    if (headerTableRef.current) {
      setHeaderTableHeight(headerTableRef.current.offsetHeight);
    }
  }, [headerTableRef.current?.offsetHeight]);

  const isReorder = (column: ColumnDef, type: string = 'asc') => {
    return sortState.id === column.accessorKey && sortState.direction === type;
  };

  const toggleRowExpansion = (rowId: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(rowId)) {
        next.delete(rowId);
      } else {
        next.add(rowId);
      }
      return next;
    });
  };

  return (
    <div>
      <div className="mb-4 flex items-end gap-2">
        {onFilterChange && (
          <DataTableFilterList
            columns={columns}
            onFilterChange={onFilterChange}
            filterSelections={filterFieldSelections}
            filterIdButton={filterIdButton}
          />
        )}
        {onSearchChange && (
          <div className="relative">
            <Input
              className="peer ps-9"
              placeholder="Search"
              type="text"
              onKeyDown={handleSearchChange}
            />
            <div className="pointer-events-none absolute inset-y-0 start-0 flex items-center justify-center ps-3 text-muted-foreground/80 peer-disabled:opacity-50">
              <SearchIcon size={16} aria-hidden="true" />
            </div>
          </div>
        )}
        {headerExtraContent && (
          <div className="ml-auto flex gap-2">{headerExtraContent}</div>
        )}
      </div>
      <div className="w-full">
        <div className="rounded-md border shadow-sm">
          {/* <div className="rounded-md border shadow-md"> */}
          <ScrollArea
            className={`relative overflow-auto ${className || 'h-[63vh]'}`}
            isScrollViewport={isScrollViewport}
          >
            <Table>
              <TableHeader
                ref={headerTableRef}
                className="sticky top-0 z-50 bg-gray-100"
              >
                <TableRow>
                  {renderExpandedRow && <TableHead className="w-12 py-4" />}
                  {showSelection && <TableHead className="py-4" />}
                  {columns.map((column) => (
                    <TableHead key={String(column.accessorKey)}>
                      {typeof column.header === 'string' ? (
                        <>
                          {column.sortable ? (
                            <Button
                              className="p-2 font-semibold"
                              variant="ghost"
                              onClick={() => handleSort(column.accessorKey)}
                            >
                              {column.header}
                              <span className="nowrap flex flex-col">
                                <ChevronUp
                                  className={cn(
                                    'mb-[-4px] h-[16px] w-[16px]',
                                    !isReorder(column, 'asc') &&
                                      'text-[#c5c5c5]'
                                  )}
                                />
                                <ChevronDown
                                  className={cn(
                                    'mt-[-4px] h-[16px] w-[16px]',
                                    !isReorder(column, 'desc') &&
                                      'text-[#c5c5c5]'
                                  )}
                                />
                              </span>
                            </Button>
                          ) : (
                            <span className="px-2">{column.header}</span>
                          )}
                        </>
                      ) : (
                        (column.header ?? '')
                      )}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  Array.from({ length: 5 }).map((_, index) => (
                    <TableRow key={`skeleton-${index}`}>
                      <TableCell colSpan={columns.length + 2} align="center">
                        <Skeleton className="h-10 rounded-md" />
                      </TableCell>
                    </TableRow>
                  ))
                ) : sortedData.length > 0 ? (
                  <>
                    {showSelection && selectedId[pageKey]?.length > 0 && (
                      <TableRow
                        className="sticky z-10 bg-white"
                        style={{
                          top: `calc(${headerTableHeight}px - 1px)`,
                          boxShadow: '0 2px 4px rgba(0, 0, 0, 0.1)',
                        }}
                      >
                        <TableCell>
                          <Checkbox
                            aria-label="Select all rows"
                            color="cyan"
                            checked={
                              sortedData.length > 0 &&
                              sortedData.every((item) =>
                                selectedId[pageKey]?.includes(String(item.id))
                              )
                            }
                            indeterminate={
                              sortedData.length === 1 ||
                              (sortedData.some((item) =>
                                selectedId[pageKey]?.includes(String(item.id))
                              ) &&
                                !sortedData.every((item) =>
                                  selectedId[pageKey]?.includes(String(item.id))
                                ))
                            }
                            onChange={(event) =>
                              toggleSelectAll(
                                event.currentTarget.checked,
                                pageKey
                              )
                            }
                          />
                        </TableCell>
                        <TableCell
                          colSpan={columns.length + 2}
                          className="px-4"
                        >
                          {selectedId[pageKey]?.length > 0 && (
                            <div className="border-b-1 m-0 flex h-[60px] w-full items-center justify-start">
                              <p className="font-semibold">
                                {`Selected (${selectedId[pageKey]?.length}) rows`}
                              </p>
                              {rowSelectorExtraContent && (
                                <div className="ml-auto flex gap-2">
                                  {rowSelectorExtraContent}
                                </div>
                              )}
                            </div>
                          )}
                        </TableCell>
                      </TableRow>
                    )}
                    {sortedData.map((pData: any, index) => {
                      const isExpanded = expandedRows.has(String(pData.id));
                      const isExpandable = getExpandableIndicator
                        ? getExpandableIndicator(pData)
                        : true;

                      return (
                        <React.Fragment key={index}>
                          <TableRow>
                            {renderExpandedRow && (
                              <TableCell className="w-12">
                                {isExpandable && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-8 w-8 p-0"
                                    onClick={() =>
                                      toggleRowExpansion(String(pData.id))
                                    }
                                  >
                                    {isExpanded ? (
                                      <ChevronUp className="h-4 w-4" />
                                    ) : (
                                      <ChevronDown className="h-4 w-4" />
                                    )}
                                  </Button>
                                )}
                              </TableCell>
                            )}
                            {showSelection && (
                              <TableCell>
                                <Checkbox
                                  color="cyan"
                                  checked={
                                    selectedId[pageKey]?.includes(
                                      String(pData.id)
                                    ) || false
                                  }
                                  onChange={() =>
                                    toggleSelectItem(String(pData.id))
                                  }
                                />
                              </TableCell>
                            )}
                            {columns.map((column) => (
                              <TableCell
                                key={String(column.accessorKey)}
                                className="px-4"
                              >
                                {column.cell
                                  ? column.cell({ row: pData })
                                  : cleanValue(pData[column.accessorKey]) ||
                                    '-'}
                              </TableCell>
                            ))}
                          </TableRow>
                          {renderExpandedRow && isExpanded && (
                            <TableRow>
                              <TableCell
                                colSpan={
                                  columns.length + (showSelection ? 2 : 1)
                                }
                                className="p-0"
                              >
                                {renderExpandedRow(pData)}
                              </TableCell>
                            </TableRow>
                          )}
                        </React.Fragment>
                      );
                    })}
                  </>
                ) : (
                  <TableRow>
                    <TableCell colSpan={columns.length + 2} align="center">
                      <p className="p-2 text-center text-muted-foreground">
                        No data
                      </p>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </ScrollArea>
        </div>
      </div>
      <div className="mt-4 flex items-center justify-between">
        {sortedData.length > 0 && (
          <p>
            Show <span className="font-semibold">{data.length}</span> of{' '}
            <span className="font-semibold">{pageCount}</span>
          </p>
        )}
        <Pagination
          withEdges
          value={page}
          onChange={handlePageChange}
          total={totalPages}
        />
      </div>
    </div>
  );
};

export { DataTable };
export type { ColumnDef };
