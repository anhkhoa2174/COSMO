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
            <div className="relative w-full max-w-md mb-6">
                {imageSrc && imageSrc !== '' && isShowImage ? (
                    <img src={imageSrc} alt={"tour"} className="w-full min-w-[250px] h-[200px] rounded-lg" />
                ) : (
                    isShowImage ? (
                        <div className="w-full h-[200px] rounded-lg bg-gray-200 flex flex-col items-center justify-center">
                            <span className="text-gray-500">No Image</span>
                            <span className="text-gray-500 text-xs">(Image not available)</span>
                        </div>
                    ) : null
                )}
            </div>
            <h2 className="text-2xl font-bold mb-2 text-center">{title}</h2>
            <p className="text-center text-gray-600 mb-8">{description}</p>
            <div className="flex justify-between w-full max-w-md">
                <div className="flex gap-2">
                    {steps.length === 1 ? null : (
                        <div className='flex gap-2'>
                            <div className="flex items-center space-x-1">
                                {Array.from({ length: steps.length }).map((_, index) => (
                                    <span
                                        onClick={() => onNavigate?.(index)}
                                        key={index}
                                        className={cn('cursor-pointer block w-2 h-2 rounded-full', index === currentStep ? 'bg-blue-400' : 'bg-gray-400')}
                                    />
                                ))}
                            </div>
                            <button
                                onClick={onSkipAll}
                                className="text-gray-500 hover:text-gray-700 font-medium pb-1 pl-2"
                            >
                                Skip All
                            </button>
                        </div>
                    )}
                </div>

                <div className='flex gap-2'>
                    {currentStep > 0 && (
                        <button
                            onClick={onPrev}
                            className="text-gray-500 hover:text-gray-700 font-medium pb-1 pr-2"
                        >
                            Back
                        </button>
                    )}

                    {currentStep < steps.length - 1 ? (
                        <Button
                            onClick={onNext}
                            variant="default"
                        >
                            Next
                        </Button>
                    ) : (
                        <>
                            {isNextSubmit ? (
                                <Button
                                    onClick={onFinish}
                                    variant="default"
                                >
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
