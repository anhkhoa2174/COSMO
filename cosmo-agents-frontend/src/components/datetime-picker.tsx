'use client';

import React, { useEffect, useRef, useState } from 'react';
import { DayPickerSingleProps } from 'react-day-picker';
import { Calendar } from '@/components/ui/calendar';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

/**
 * Regular expression to check for valid hour format (01-23)
 */
function isValidHour(value: string) {
  return /^(0[0-9]|1[0-9]|2[0-3])$/.test(value);
}

/**
 * Regular expression to check for valid minute format (00-59)
 */
function isValidMinuteOrSecond(value: string) {
  return /^[0-5][0-9]$/.test(value);
}

type GetValidNumberConfig = { max: number; min?: number; loop?: boolean };

function getValidNumber(value: string, { max, min = 0, loop = false }: GetValidNumberConfig) {
  let numericValue = parseInt(value, 10);

  if (!Number.isNaN(numericValue)) {
    if (!loop) {
      if (numericValue > max) {
        numericValue = max;
      }
      if (numericValue < min) {
        numericValue = min;
      }
    } else {
      if (numericValue > max) {
        numericValue = min;
      }
      if (numericValue < min) {
        numericValue = max;
      }
    }
    return numericValue.toString().padStart(2, '0');
  }

  return '00';
}

function getValidHour(value: string) {
  if (isValidHour(value)) {
    return value;
  }
  return getValidNumber(value, { max: 23 });
}

function getValidMinuteOrSecond(value: string) {
  if (isValidMinuteOrSecond(value)) {
    return value;
  }
  return getValidNumber(value, { max: 59 });
}

type TimePickerType = 'hour' | 'minute' | 'second';

function setDateByType(value: string, type: TimePickerType) {
  switch (type) {
    case 'hour':
      return getValidHour(value);
    case 'minute':
    case 'second':
      return getValidMinuteOrSecond(value);
    default:
      return value;
  }
}

function getDateByType(value: string, type: TimePickerType) {
  switch (type) {
    case 'hour':
      return getValidHour(value);
    case 'minute':
    case 'second':
      return getValidMinuteOrSecond(value);
    default:
      return '00';
  }
}

interface TimeInputProps extends Omit<React.ComponentProps<'input'>, 'value'> {
  picker: 'hour' | 'minute' | 'second';
  onRightFocus?: () => void;
  onLeftFocus?: () => void;
  value?: string;
  onValueChange?: (val: string) => void;
}

const TimeInput = React.forwardRef<HTMLInputElement, TimeInputProps>(
  (
    {
      picker,
      value,
      onValueChange,
      onChange,
      onKeyDown,
      onRightFocus,
      onLeftFocus,
      className,
      ...props
    },
    ref
  ) => {
    const [flag, setFlag] = useState<boolean>(false);

    /**
     * Allow the user to enter the second digit within 2 seconds
     * otherwise start again with entering first digit
     */
    useEffect(() => {
      if (flag) {
        const timer = setTimeout(() => {
          setFlag(false);
        }, 2000);

        return () => clearTimeout(timer);
      }
    }, [flag]);

    const calculatedValue = React.useMemo(() => {
      return getDateByType(value || '0', picker);
    }, [value, picker]);

    const calculateNewValue = (key: string) => {
      return !flag ? `0${key}` : calculatedValue.slice(1, 2) + key;
    };

    const handleOnKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Tab') {
        return;
      }
      e.preventDefault();
      if (e.key === 'ArrowRight') {
        onRightFocus?.();
      }
      if (e.key === 'ArrowLeft') {
        onLeftFocus?.();
      }
      if (e.key >= '0' && e.key <= '9') {
        const newValue = calculateNewValue(e.key);
        if (flag) {
          onRightFocus?.();
        }
        setFlag((prev) => !prev);
        onValueChange?.(setDateByType(newValue, picker));
      }
    };

    return (
      <Input
        ref={ref}
        type="tel"
        className={cn(
          'w-12 [&::-webkit-inner-spin-button]:appearance-none tabular-nums caret-transparent',
          className
        )}
        inputMode="decimal"
        value={setDateByType(calculatedValue, picker)}
        onChange={(e) => {
          e.preventDefault();
          onChange?.(e);
        }}
        onKeyDown={(e) => {
          onKeyDown?.(e);
          handleOnKeyDown(e);
        }}
        {...props}
      />
    );
  }
);

interface DatetimePickerProps
  extends Omit<DayPickerSingleProps, 'mode' | 'selected' | 'onSelect' | 'disabled'> {
  value?: Date;
  onChange?: (date: Date) => void;
}

const DatetimePicker = ({ value, onChange, ...props }: DatetimePickerProps) => {
  const hourRef = useRef<HTMLInputElement>(null);
  const minuteRef = useRef<HTMLInputElement>(null);
  const secondRef = useRef<HTMLInputElement>(null);

  const [date, setDate] = useState(value);

  const handleSelect = (newDate: Date | undefined) => {
    if (newDate) {
      newDate.setHours(date?.getHours() ?? 0, date?.getMinutes() ?? 0, date?.getSeconds() ?? 0);
      setDate(newDate);
      onChange?.(newDate);
    }
  };

  const handleOnTimeInputChange = (value: string, picker: TimePickerType) => {
    const newDate = date ? new Date(date) : new Date(new Date().setHours(0, 0, 0));
    if (picker === 'hour') {
      newDate.setHours(parseInt(value, 10));
    } else if (picker === 'minute') {
      newDate.setMinutes(parseInt(value, 10));
    } else if (picker === 'second') {
      newDate.setSeconds(parseInt(value, 10));
    }
    setDate(newDate);
    onChange?.(newDate);
  };

  return (
    <div className="flex divide-x">
      <Calendar
        mode="single"
        selected={date}
        onSelect={handleSelect}
        disabled={{ before: new Date() }}
        {...props}
      />
      <div className="flex flex-col justify-center items-center p-3 gap-3">
        <p>Hour</p>
        <TimeInput
          picker="hour"
          ref={hourRef}
          value={date?.getHours().toString()}
          onValueChange={(val) => handleOnTimeInputChange(val, 'hour')}
          // onLeftFocus={() => secondRef.current?.focus()}
          onRightFocus={() => minuteRef.current?.focus()}
        />
        <p>Minute</p>
        <TimeInput
          picker="minute"
          ref={minuteRef}
          value={date?.getMinutes().toString()}
          onValueChange={(val) => handleOnTimeInputChange(val, 'minute')}
          onLeftFocus={() => hourRef.current?.focus()}
          onRightFocus={() => secondRef.current?.focus()}
        />
        <p>Second</p>
        <TimeInput
          picker="second"
          ref={secondRef}
          value={date?.getSeconds().toString()}
          onValueChange={(val) => handleOnTimeInputChange(val, 'second')}
          onLeftFocus={() => minuteRef.current?.focus()}
          // onRightFocus={() => hourRef.current?.focus()}
        />
      </div>
    </div>
  );
};

DatetimePicker.displayName = 'DatetimePicker';

export { DatetimePicker };
export type { DatetimePickerProps };
