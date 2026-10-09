import { TourStep } from '@/hooks/use-tour';
import { cn } from '@/lib/utils';
import { CheckIcon } from 'lucide-react';
import React from 'react';
import { Button } from '../ui/button';

interface TourWithImageProps {
  isNextSubmit?: boolean;
  title: string | React.ReactNode;
  description: string | React.ReactNode;
  imageSrc?: string;
  onNext: () => void;
  onPrev: () => void;
  onFinish: () => void;
  onNavigate?: (step: number) => void;
  onSkipAll: () => void;
  steps: TourStep[];
  currentStep: number;
  isShowImage?: boolean;
}

const TourWithImage: React.FC<TourWithImageProps> = ({
  isNextSubmit,
  title,
  description,
  imageSrc,
  onNext,
  onPrev,
  onFinish,
  onNavigate,
  onSkipAll,
  steps,
  currentStep,
  isShowImage = false,
}) => {
  return (
    <div className="flex flex-col items-center justify-center">
      <div className="relative mb-6 w-full max-w-md">
        {imageSrc && imageSrc !== '' && isShowImage ? (
          <img
            src={imageSrc}
            alt={'tour'}
            className="h-[200px] w-full min-w-[250px] rounded-lg"
          />
        ) : isShowImage ? (
          <div className="flex h-[200px] w-full flex-col items-center justify-center rounded-lg bg-gray-200">
            <span className="text-gray-500">No Image</span>
            <span className="text-xs text-gray-500">(Image not available)</span>
          </div>
        ) : null}
      </div>
      <h2 className="mb-2 text-center text-2xl font-bold">{title}</h2>
      <p className="mb-8 text-center text-gray-600">{description}</p>
      <div className="flex w-full max-w-md justify-between">
        <div className="flex gap-2">
          {steps.length === 1 ? null : (
            <div className="flex gap-2">
              <div className="flex items-center space-x-1">
                {Array.from({ length: steps.length }).map((_, index) => (
                  <span
                    onClick={() => onNavigate?.(index)}
                    key={index}
                    className={cn(
                      'block h-2 w-2 cursor-pointer rounded-full',
                      index === currentStep ? 'bg-blue-400' : 'bg-gray-400'
                    )}
                  />
                ))}
              </div>
              <button
                onClick={onSkipAll}
                className="pb-1 pl-2 font-medium text-gray-500 hover:text-gray-700"
              >
                Skip All
              </button>
            </div>
          )}
        </div>

        <div className="flex gap-2">
          {currentStep > 0 && (
            <button
              onClick={onPrev}
              className="pb-1 pr-2 font-medium text-gray-500 hover:text-gray-700"
            >
              Back
            </button>
          )}

          {currentStep < steps.length - 1 ? (
            <Button onClick={onNext} variant="default">
              Next
            </Button>
          ) : (
            <>
              {isNextSubmit ? (
                <Button onClick={onFinish} variant="default">
                  Next
                </Button>
              ) : (
                <Button
                  onClick={onFinish}
                  variant="default"
                  className="flex items-center gap-2 bg-green-600 text-white hover:bg-green-700"
                >
                  <CheckIcon /> Finish
                </Button>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default TourWithImage;
