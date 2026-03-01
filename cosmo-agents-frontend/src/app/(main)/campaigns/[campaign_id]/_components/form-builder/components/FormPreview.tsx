'use client';

import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { zodResolver } from '@hookform/resolvers/zod';
import { Loader } from 'lucide-react';
import React from 'react';
import { FieldValues, SubmitHandler, useForm } from 'react-hook-form';
import { FormElementRenderer } from './FormElements';
import { FormElement, FormState, SubmitButtonAppearance } from './types';

interface FormPreviewProps {
  formState?: FormState;
  validationSchema?: any;
  onFormSubmit: SubmitHandler<FieldValues>;
  selectedElement?: FormElement | null;
  loading?: boolean;
}

export default function FormPreview({
  formState,
  onFormSubmit,
  validationSchema,
  selectedElement,
  loading,
}: FormPreviewProps) {

  const form = useForm({
    resolver: validationSchema ? zodResolver(validationSchema) : undefined,
  });

  // Destructure settings for easier use, providing fallbacks
  const {
    title = 'Form Preview', // Default title
    description, // Optional, no default needed here if handled by conditional rendering
    submitButtonText = 'Submit', // Default submit text
    submitButtonAppearance, // Destructure this
    styles,
  } = formState?.settings || {
    styles: {},
    submitButtonAppearance: {} as SubmitButtonAppearance,
  };

  // Destructure submit button appearance with defaults
  const {
    backgroundColor: btnBgColor = '#071F1D', // Default from FormBuilder state
    textColor: btnTextColor = '#ffffff', // Default from FormBuilder state
    fullWidth: btnFullWidth = false,
    alignment: btnAlignment = 'right',
  } = submitButtonAppearance || ({} as SubmitButtonAppearance);

  // Destructure styles for easier use, providing fallbacks
  const {
    margin = '0px',
    padding = '16px', // For CardContent
    backgroundColor: formBgColor = '#ffffff', // For CardContent
    gap = '16px', // For elements container
    borderRadius = '8px', // Default from FormBuilder
    border = '1px solid #e0e0e0', // Default from FormBuilder
    boxShadow = 'rgba(149, 157, 165, 0.2) 0px 8px 24px', // Corrected default based on your FormBuilder change
  } = styles || {}; // Ensure styles is not undefined

  // Style object for the main Card
  const cardStyle: React.CSSProperties = {
    margin: margin,
    borderRadius: borderRadius,
    border: border,
    boxShadow: boxShadow,
  };

  // Create a style object for the CardContent
  const cardContentStyle = {
    padding: padding,
    backgroundColor: formBgColor,
  };

  // Style for the elements container
  const elementsContainerStyle = {
    display: 'grid',
    gap: gap,
  };

  // Styles for the submit button itself
  const submitButtonStyle: React.CSSProperties = {
    backgroundColor: btnBgColor,
    color: btnTextColor,
    width: btnFullWidth ? '100%' : 'auto',
  };

  // Styles for the div containing the submit button to handle alignment
  const submitButtonContainerStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent:
      btnAlignment === 'left'
        ? 'flex-start'
        : btnAlignment === 'center'
          ? 'center'
          : 'flex-end',
    paddingTop: '1rem', // Keep existing padding
  };
  const settings = formState?.settings;
  const elements = formState?.elements;
  return (
    <Card className="overflow-hidden" style={cardStyle}>
      <CardContent style={cardContentStyle}>
        <CardHeader className="mb-2 p-0 lg:mb-0">
          <CardTitle className="text-xl">{title}</CardTitle>
          {description && <CardDescription>{description}</CardDescription>}
        </CardHeader>
        <form onSubmit={form.handleSubmit(onFormSubmit)}>
          {elements?.length === 0 ? (
            <div className="py-6 text-center text-muted-foreground">
              Add form elements to see a preview
            </div>
          ) : (
            <div className="relative flex h-[500px] w-full flex-1 flex-col overflow-y-auto lg:h-full">
              <div
                className={`grid grid-cols-${settings?.gridColumns || 4} gap-${settings?.styles?.gap || 3}`}
                style={{ ...elementsContainerStyle } as React.CSSProperties}
              >
                {elements?.map((element) => {
                  const colSpan = element.colSpan || 1;
                  const error =
                    form.formState.errors?.[element.name]?.message;
                  const isSelected = selectedElement?.id === element.id;
                  return (
                    <div
                      key={element.id}
                      className={`col-span-${colSpan}`}
                      style={{ gridColumn: `span ${colSpan}` }}
                    >
                      <FormElementRenderer
                        element={element}
                        form={form}
                        error={error}
                        isSelected={isSelected}
                      />
                    </div>
                  );
                })}
              </div>
              <div
                className="sticky bottom-0 z-10"
                style={submitButtonContainerStyle}
              >
                <Button type="submit" style={submitButtonStyle} disabled={loading}>
                  {loading && (
                    <Loader className="mr-2 h-4 w-4 animate-spin text-white" />
                  )}
                  {submitButtonText}
                </Button>
              </div>
            </div>
          )}
        </form>
      </CardContent>
    </Card>
  );
}
