// hooks/useTour.ts
import '@/assets/styles/useTour.css';
import TourWithImage from '@/components/tour/TourWithImage';
import { kyClient } from '@/lib/ky';
import { OnboardingPayload, User } from '@/models/auth';
import { ApiResponse } from '@/models/response';
import { driver } from 'driver.js'; // Use named export 'driver'
import 'driver.js/dist/driver.css';
import { useCallback, useRef } from 'react';
import { createRoot } from 'react-dom/client';

export interface TourStep {
  element: string;
  popover: {
    title: string;
    description: string;
    side?:
      | 'top'
      | 'left'
      | 'right'
      | 'bottom'
      | 'top-center'
      | 'bottom-center'
      | 'left-center'
      | 'right-center';
    customContent?: ({
      handleNext,
      handlePrev,
      handleFinish,
      handleSkipAll,
      handleNavigate,
      activeIndex,
      tourSteps,
    }: {
      handleNext: any;
      handlePrev: any;
      handleFinish: any;
      handleSkipAll: any;
      handleNavigate: any;
      activeIndex: any;
      tourSteps: TourStep[];
    }) => React.ReactNode;
  };
}

export interface UseTourReturn {
  start: () => void;
  reset: () => void;
  moveNext: () => void;
  moveTo: (step: number) => void;
  movePrevious: () => void;
  driver: ReturnType<typeof driver> | null; // Type the driver instance correctly
}

export const UpdateOnboarding = async (
  payload: OnboardingPayload,
  user: User | null
) => {
  const ui_metadata: OnboardingPayload = {
    ...user?.ui_metadata,
    ...payload,
  };
  try {
    await kyClient
      .patch('v1/users/me', {
        json: {
          ui_metadata,
        },
      })
      .json<ApiResponse<User>>();
  } catch (error) {}
};

interface CreateTourCustomContentProps {
  title: string | React.ReactNode;
  isNextSubmit?: boolean;
  description: string | React.ReactNode;
  imageSrc: string;
  user: User | null;
  onFinish?: (step: number) => void;
  onSkipAll?: (step: number) => void;
  onPrev?: (step: number) => void;
  onNext?: (step: number) => void;
  onNavigate?: (step: number) => void;
  update?: () => Promise<void>;
}

export const createTourCustomContent = ({
  title,
  isNextSubmit,
  description,
  imageSrc,
  user,
  onFinish,
  onSkipAll,
  onPrev,
  onNext,
  onNavigate,
  update,
}: CreateTourCustomContentProps) => {
  return ({
    handleNext,
    handlePrev,
    handleSkipAll,
    handleFinish,
    handleNavigate,
    activeIndex,
    tourSteps,
  }: any) => {
    const handleOnFinish = async () => {
      if (onFinish) {
        onFinish(activeIndex);
      } else {
        await UpdateOnboarding(
          {
            onboarding: false,
            status: 'completed',
            step: activeIndex,
          },
          user
        );
      }
      handleFinish();
      if (!isNextSubmit) {
        await update?.();
      }
    };
    const handleOnSkipAll = async () => {
      if (onSkipAll) {
        onSkipAll(activeIndex);
      } else {
        await UpdateOnboarding(
          {
            onboarding: false,
            step: activeIndex,
            status: 'skipped',
          },
          user
        );
      }
      handleSkipAll();
      if (!isNextSubmit) {
        await update?.();
      }
    };
    const handleOnPrev = () => {
      const prevStep = activeIndex - 1;
      if (prevStep < 0) {
        return;
      }
      if (onPrev) {
        onPrev(prevStep);
      } else {
        UpdateOnboarding(
          {
            onboarding: true,
            step: prevStep,
            status: 'in_progress',
          },
          user
        );
      }
      handlePrev();
    };
    const handleOnNext = () => {
      const nextStep = activeIndex + 1;
      if (nextStep >= tourSteps.length) {
        return;
      }
      if (onNext) {
        onNext(nextStep);
      } else {
        UpdateOnboarding(
          {
            onboarding: true,
            step: nextStep,
            status: 'in_progress',
          },
          user
        );
      }
      handleNext();
    };
    const handleOnNavigate = (step: number) => {
      if (step < 0 || step >= tourSteps.length) {
        return;
      }
      if (onNavigate) {
        onNavigate(step);
      } else {
        UpdateOnboarding(
          {
            onboarding: true,
            step: step,
            status: 'in_progress',
          },
          user
        );
      }
      handleNavigate(step);
    };
    return (
      <TourWithImage
        isNextSubmit={isNextSubmit}
        title={title}
        imageSrc={imageSrc}
        description={description}
        onNext={handleOnNext}
        onPrev={handleOnPrev}
        onNavigate={handleOnNavigate}
        onSkipAll={handleOnSkipAll}
        onFinish={handleOnFinish}
        currentStep={activeIndex}
        steps={tourSteps}
      />
    );
  };
};

export const useTour = (
  steps: TourStep[],
  options?: any // Use DriverOptions type
): UseTourReturn => {
  const driverRef = useRef<ReturnType<typeof driver> | null>(null);

  const start = useCallback(() => {
    if (!driverRef.current) {
      driverRef.current = driver({
        showProgress: true,
        allowClose: true,
        overlayClickNext: false,
        nextBtnText: 'Next',
        prevBtnText: 'Back',
        doneBtnText: 'Done',
        steps: steps,
        ...options,
        onPopoverRender: (popover, { config, state, driver: tourDriver }) => {
          // Renamed driver to tourDriver
          if (
            typeof state.activeIndex !== 'number' ||
            state.activeIndex < 0 ||
            !config.steps
          ) {
            return;
          }
          const currentStep = config.steps[state.activeIndex] as TourStep;
          let customContentContainer = popover.wrapper.querySelector(
            '.custom-popover-content'
          ) as HTMLElement | null;

          if (currentStep && currentStep.popover.customContent) {
            popover.wrapper.classList.add(
              `has-custom-content-${currentStep.popover.side}`
            );
            popover.title.innerHTML = '';
            popover.description.innerHTML = '';
            popover.footer.style.display = 'none'; // Hide footer for custom content

            if (!customContentContainer) {
              customContentContainer = document.createElement('div');
              customContentContainer.className = 'custom-popover-content';
              popover.wrapper.appendChild(customContentContainer);
            } else {
              // If container exists, unmount previous React content before rendering new
              const root = createRoot(customContentContainer);
              root.unmount(); // Unmounts any existing React content from this container
            }

            const reactNode = currentStep.popover.customContent({
              handleNext: tourDriver?.moveNext,
              handlePrev: tourDriver?.movePrevious,
              handleSkipAll: tourDriver?.destroy,
              handleFinish: tourDriver?.destroy,
              handleNavigate: tourDriver?.moveTo,
              activeIndex: state.activeIndex,
              tourSteps: steps,
            });
            // Create a new root for the new content on the (potentially reused) container
            const root = createRoot(customContentContainer!); // customContentContainer is guaranteed to be non-null here
            root.render(reactNode!);
          } else if (currentStep) {
            popover.wrapper.classList.remove(
              `has-custom-content-${currentStep.popover.side}`
            );
            // This is a standard step (no custom content)
            if (customContentContainer) {
              const root = createRoot(customContentContainer);
              root.unmount(); // Unmount React content if it was a custom step
              customContentContainer.remove(); // Remove custom content container
            }
            popover.title.innerHTML = currentStep.popover.title;
            popover.description.innerHTML = currentStep.popover.description;
          }
        },
      });
    }

    driverRef.current.drive();
  }, [steps, options]);

  const reset = useCallback(() => {
    driverRef.current?.destroy();
    driverRef.current = null;
  }, []);

  const moveNext = useCallback(() => {
    driverRef.current?.moveNext();
  }, []);

  const movePrevious = useCallback(() => {
    driverRef.current?.movePrevious();
  }, []);

  const moveTo = useCallback((step: number) => {
    driverRef.current?.moveTo(step);
  }, []);

  return {
    start,
    reset,
    moveTo,
    moveNext,
    movePrevious,
    driver: driverRef.current,
  };
};
