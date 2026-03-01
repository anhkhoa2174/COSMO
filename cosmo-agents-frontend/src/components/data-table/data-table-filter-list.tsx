'use client';

import { Input } from '@/components/ui/input';
import { useClickOutside } from '@/hooks/use-click-outside';
import { currentPageAtom } from '@/stores/atom';
import { useAtom } from 'jotai';
import _ from 'lodash';
import { Check, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { AddButton } from '../buttons/add-button';
import { Combobox } from '../combobox';
import { ColumnDef } from './data-table';

const columnOperators = [
  { value: '$ilike', label: 'Contains' },
  { value: '$nilike', label: 'Does not contain' },
  { value: '$eq', label: 'Is' },
  { value: '$ne', label: 'Is not' },
  // { value: '$empty', label: 'Is empty' },
  // { value: '$nempty', label: 'Is not empty' },
];

const filterOperators = [
  { value: '$and', label: 'and' },
  { value: '$or', label: 'or' },
];

interface FilterValue {
  column: ColumnDef['accessorKey'];
  operator: string;
  value: string;
}

export const DataTableFilterList = ({
  columns,
  onFilterChange,
  filterSelections,
  filterIdButton,
}: {
  columns: ColumnDef[];
  onFilterChange: any;
  filterSelections?: Record<string, any[]>;
  filterIdButton?: string;
}) => {
  const [tokens, setTokens] = useState<any[]>([]);
  const [trigger, setTrigger] = useState(0);

  const handleAddToken = () => {
    const _tokens = [...tokens];
    if (tokens.length > 0) {
      _tokens.push('$and');
    }
    _tokens.push({
      field: columns[0].accessorKey,
      operator: '$ilike',
      value: '',
    });
    setTokens(_tokens);
    setFocusedToken(_tokens.length - 1);
  };

  const handleRemoveToken = (index: number) => {
    if (index > 0) {
      setTokens((prev) =>
        prev.slice(0, index - 1).concat(prev.slice(index + 1))
      );
    } else {
      setTokens((prev) => prev.slice(2));
    }
    setTrigger((prev) => prev + 1);
  };

  const handleChangeToken = (index: number, value: any, key?: string) => {
    if (typeof tokens[index] === 'string') {
      setTokens((prev) =>
        prev.map((token, i) => (i === index ? value : token))
      );
      setTrigger((prev) => prev + 1);
    } else {
      if (!key) return;
      setTokens((prev) =>
        prev.map((token, i) => {
          if (i === index) {
            return {
              ...token,
              [key]: value,
              ...(key === 'field' ? { value: '' } : {}),
            };
          }
          return token;
        })
      );
    }
  };

  const [focusedToken, setFocusedToken] = useState<number | null>(null);

  const [, setCurrentPage] = useAtom(currentPageAtom);

  const handleApply = () => {
    const temp = tokens.reduce((acc, filter) => {
      if (typeof filter === 'string') {
        acc.push(filter);
      } else {
        acc.push({
          [filter.field]: {
            [filter.operator]: ['$ilike', '$nilike'].includes(filter.operator)
              ? `%${filter.value}%`
              : filter.value,
          },
        });
      }
      return acc;
    }, [] as any[]);
    // const filter = temp.length > 1 ? { [operator]: temp } : temp[0];
    setCurrentPage({ contactsTable: 1 });
    const filter = parseFilter(temp);
    onFilterChange?.(filter);
  };

  const columnLabel = (column: ColumnDef['accessorKey']) =>
    columns.find((col) => col.accessorKey === column)?.header;

  const operatorLabel = (operator: string) =>
    columnOperators.find((col) => col.value === operator)?.label;

  const handleCheck = () => {
    setFocusedToken(null);
    setTrigger((prev) => prev + 1);
  };

  useEffect(() => {
    if (trigger > 0) {
      _.debounce(handleApply, 300)();
    }
  }, [trigger]);

  const ref = useClickOutside(() => {
    setFocusedToken(null);
    handleApply();
  });

  return (
    <div className="space-y-4">
      {tokens.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {tokens.map((token, index) => {
            if (typeof token === 'string') {
              return (
                <Combobox
                  key={`token-${index}`}
                  value={token}
                  onChange={(value) => handleChangeToken(index, value)}
                  data={filterOperators}
                  placeholder="Select operator"
                  className="w-[64px] border-none bg-blue-50 text-blue-500 shadow-none focus:ring-0"
                  withInput={false}
                />
              );
            }

            return (
              <div
                key={`token-${index}`}
                className="flex items-center gap-2 text-nowrap rounded-md bg-zinc-100 p-2"
                ref={ref}
                // onBlur={() => {
                //   if (token.value === '') {
                //     setFocusedToken(null);
                //     handleRemoveToken(index);
                //   }
                // }}
              >
                <div tabIndex={0} onFocus={() => setFocusedToken(index)}>
                  {focusedToken === index ? (
                    <div className="flex items-center gap-2">
                      <Combobox
                        value={token.field}
                        onChange={(value) =>
                          handleChangeToken(index, value, 'field')
                        }
                        data={columns.map(({ accessorKey, header }) => ({
                          value: String(accessorKey),
                          label: header as string,
                        }))}
                        placeholder="Select column"
                      />
                      <Combobox
                        value={token.operator}
                        onChange={(value) =>
                          handleChangeToken(index, value, 'operator')
                        }
                        data={columnOperators}
                        placeholder="Select operator"
                        withInput={false}
                      />
                      {filterSelections?.[token.field] ? (
                        <Combobox
                          value={token.value}
                          onChange={(value) =>
                            handleChangeToken(index, value, 'value')
                          }
                          data={filterSelections[token.field].map((value) => ({
                            value: String(value),
                            label: String(value),
                          }))}
                          placeholder="Select value"
                          className="w-[168px]"
                        />
                      ) : (
                        <Input
                          autoFocus
                          value={token.value}
                          onChange={(e) =>
                            handleChangeToken(index, e.target.value, 'value')
                          }
                          className="w-[168px]"
                          placeholder="Enter value"
                        />
                      )}
                    </div>
                  ) : (
                    <span>
                      {columnLabel(token.field)}{' '}
                      <span className="font-medium lowercase text-blue-500">
                        {operatorLabel(token.operator)}
                      </span>{' '}
                      {token.value}
                    </span>
                  )}
                </div>
                {focusedToken === index ? (
                  <button
                    onClick={handleCheck}
                    type="button"
                    className="text-muted-foreground"
                  >
                    <Check className="h-4 w-4" />
                  </button>
                ) : (
                  <button
                    onClick={() => handleRemoveToken(index)}
                    type="button"
                    className="text-muted-foreground"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            );
          })}
        </div>
      )}
      <AddButton
        className="text-blue-500"
        text="Add filter"
        onClick={handleAddToken}
        variant="outline"
        id={filterIdButton}
        disabled={focusedToken !== null}
      />
    </div>
  );
};

const precedence = {
  $or: 1,
  $and: 2,
};

function parseFilter(tokens: any[]) {
  const outputStack: any[] = [];
  const operatorStack: any[] = [];

  const applyOperator = () => {
    const op = operatorStack.pop();
    const right = outputStack.pop();
    const left = outputStack.pop();

    // Merge operands under the operator
    outputStack.push({ [op]: [left, right] });
  };

  for (const token of tokens) {
    if (typeof token === 'string' && (token === '$and' || token === '$or')) {
      while (
        operatorStack.length &&
        precedence[operatorStack[operatorStack.length - 1]] >= precedence[token]
      ) {
        applyOperator();
      }
      operatorStack.push(token);
    } else {
      outputStack.push(token);
    }
  }

  while (operatorStack.length) {
    applyOperator();
  }

  return outputStack[0];
}
