'use client';

import { toCamelCase } from '@/helpers';
import { v4 as uuidv4 } from 'uuid';
import { systemFields } from '../FormElementsCustom';
import { FormElement, FormElementType, FormState } from '../types';
import { origin, settings } from './constants';

// Helper function to create a new form element
export const createFormElement = (
  type: FormElementType,
  customProps?: Partial<FormElement>
): FormElement => {
  const id = customProps?.id || uuidv4();
  const label =
    customProps?.label || `New ${type.charAt(0).toUpperCase() + type.slice(1)}`;
  const name = customProps?.name || toCamelCase(label);
  const baseElement = {
    id,
    type,
    label,
    name,
    required: customProps?.required || false,
    placeholder: customProps?.placeholder || `Enter ${label}...`,
    colSpan: customProps?.colSpan || 4, // Default column span is 4
    ...customProps,
  };

  switch (type) {
    case 'text':
    case 'email':
    case 'password':
    case 'number':
      return {
        ...baseElement,
        type,
        defaultValue: '',
      };
    case 'url':
      return {
        ...baseElement,
        type,
        defaultValue: origin,
      };
    case 'textarea':
      return {
        ...baseElement,
        type: 'textarea',
        defaultValue: '',
        rows: 3,
      };
    case 'select':
      return {
        ...baseElement,
        type: 'select',
        // @ts-ignore
        options: customProps?.options || [
          { label: '-- Select Option --', value: null },
          { label: 'Option A', value: 'optionA' },
          { label: 'Option B', value: 'optionB' },
          { label: 'Option C', value: 'optionC' },
        ],
        // @ts-ignore
        defaultValue: customProps?.defaultValue || null,
      };
    case 'checkbox':
      return {
        ...baseElement,
        type: 'checkbox',
        defaultChecked: false,
      };
    case 'radio':
      return {
        ...baseElement,
        type: 'radio',
        // @ts-ignore
        options: customProps?.options || [
          { label: 'Option 1', value: 'option1' },
          { label: 'Option 2', value: 'option2' },
          { label: 'Option 3', value: 'option3' },
        ],
        defaultValue: customProps?.defaultValue || 'option1',
      };
    case 'date':
      return {
        ...baseElement,
        type,
        label: customProps?.label || 'Date',
        defaultValue: customProps?.defaultValue || new Date().toISOString(),
      };
    case 'spacer':
      return {
        ...baseElement,
        type,
        label: 'Spacer',
        styles: {
          height: '16px',
          backgroundColor: 'transparent',
        },
      };
    case 'divider':
      return {
        ...baseElement,
        type,
        label: 'Divider',
        styles: {
          color: '#e0e0e0',
          thickness: '1px',
          width: '100%',
          marginTop: '8px',
          marginBottom: '8px',
        },
      };
    default:
      throw new Error(`Unknown form element type: ${type}`);
  }
};

// Create a predefined form layout based on the provided image
export const createPredefinedForm = (): FormState => {
  return {
    elements: [
      createFormElement('spacer', {
        id: uuidv4(),
        label: 'Spacer',
        required: false,
        placeholder: '',
        colSpan: 4,
        styles: {
          height: '8px',
          backgroundColor: 'transparent',
        },
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: 'First Name',
        required: true,
        placeholder: 'Enter your first name...',
        colSpan: 2,
        defaultValue: '',
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: 'Last Name',
        required: true,
        placeholder: 'Enter your last name...',
        colSpan: 2,
        defaultValue: '',
      }),
      createFormElement('email', {
        id: uuidv4(),
        label: 'Business Email',
        required: true,
        placeholder: 'Enter your business email...',
        colSpan: 4,
        defaultValue: '',
      }),
      createFormElement('date', {
        id: uuidv4(),
        label: 'Demo Day',
        required: true,
        placeholder: 'Enter your demo day...',
        colSpan: 4,
        defaultValue: new Date().toISOString(),
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: 'Company',
        required: true,
        placeholder: 'Enter your company...',
        colSpan: 4,
        defaultValue: '',
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: 'Job Title',
        required: true,
        placeholder: 'Enter your job title...',
        colSpan: 4,
        defaultValue: '',
      }),
      createFormElement('select', {
        id: uuidv4(),
        label: 'Industry',
        required: true,
        placeholder: 'Choose your industry...',
        colSpan: 2,
        options: [
          {
            label: '-- Select Industry --',
            // @ts-ignore
            value: null,
          },
          {
            label: 'Technology',
            value: 'technology',
          },
          {
            label: 'E-commerce',
            value: 'e-commerce',
          },
          {
            label: 'Education',
            value: 'education',
          },
          {
            label: 'Retail',
            value: 'retail',
          },
          {
            label: 'Other',
            value: 'other',
          },
        ],
        // @ts-ignore
        defaultValue: null,
      }),
      createFormElement('select', {
        id: uuidv4(),
        label: 'Company Size',
        required: true,
        placeholder: 'Choose your company size...',
        colSpan: 2,
        options: [
          {
            label: '-- Select Company Size --',
            // @ts-ignore
            value: null,
          },
          {
            label: '10+ employees',
            value: '10',
          },
          {
            label: '100+ employees',
            value: '100',
          },
          {
            label: '1000+ employees',
            value: '1000',
          },
        ],
        // @ts-ignore
        defaultValue: null,
      }),
      createFormElement('textarea', {
        id: uuidv4(),
        label: 'What are you interested in?',
        required: false,
        placeholder: 'Interested in threat hunting capabilities',
        colSpan: 4,
        defaultValue: '',
        rows: 3,
      }),
    ],
    settings: {
      ...settings.settings,
      submitButtonAppearance: {
        ...settings.settings.submitButtonAppearance,
        alignment: 'right' as const,
      },
    },
  };
};

export const createCustomForm = (): FormState => {
  return {
    elements: [
      createFormElement('spacer', {
        id: uuidv4(),
        label: 'Spacer',
        required: false,
        placeholder: '',
        colSpan: 4,
        styles: {
          height: '8px',
          backgroundColor: 'transparent',
        },
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: systemFields.find((el) => el.normalized_name === 'first_name')
          ?.name,
        name: systemFields.find((el) => el.normalized_name === 'first_name')
          ?.normalized_name,
        required: systemFields.find((el) => el.normalized_name === 'first_name')
          ?.is_required,
        placeholder: 'Enter your first name...',
        colSpan: 2,
        defaultValue: '',
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: systemFields.find((el) => el.normalized_name === 'last_name')
          ?.name,
        name: systemFields.find((el) => el.normalized_name === 'last_name')
          ?.normalized_name,
        required: systemFields.find((el) => el.normalized_name === 'last_name')
          ?.is_required,
        placeholder: 'Enter your last name...',
        colSpan: 2,
        defaultValue: '',
      }),
      createFormElement('email', {
        id: uuidv4(),
        label: systemFields.find((el) => el.normalized_name === 'email')?.name,
        name: systemFields.find((el) => el.normalized_name === 'email')
          ?.normalized_name,
        required: systemFields.find((el) => el.normalized_name === 'email')
          ?.is_required,
        placeholder: 'Enter your email...',
        colSpan: 4,
        defaultValue: '',
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: systemFields.find((el) => el.normalized_name === 'company')
          ?.name,
        name: systemFields.find((el) => el.normalized_name === 'company')
          ?.normalized_name,
        required: systemFields.find((el) => el.normalized_name === 'company')
          ?.is_required,
        placeholder: 'Enter your company...',
        colSpan: 4,
        defaultValue: '',
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: systemFields.find((el) => el.normalized_name === 'job_title')
          ?.name,
        name: systemFields.find((el) => el.normalized_name === 'job_title')
          ?.normalized_name,
        required: systemFields.find((el) => el.normalized_name === 'job_title')
          ?.is_required,
        placeholder: 'Enter your job title...',
        colSpan: 4,
        defaultValue: '',
      }),
      createFormElement('text', {
        id: uuidv4(),
        label: systemFields.find((el) => el.normalized_name === 'address')
          ?.name,
        name: systemFields.find((el) => el.normalized_name === 'address')
          ?.normalized_name,
        required: systemFields.find((el) => el.normalized_name === 'address')
          ?.is_required,
        placeholder: 'Enter Address...',
        colSpan: 4,
        defaultValue: '',
      }),
    ],
    settings: {
      ...settings.settings,
      submitButtonAppearance: {
        ...settings.settings.submitButtonAppearance,
        alignment: 'right',
      },
    },
  };
};
