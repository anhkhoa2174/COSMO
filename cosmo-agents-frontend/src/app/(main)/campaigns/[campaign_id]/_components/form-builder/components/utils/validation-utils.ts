'use client';

import { z } from 'zod';
import { FormElementType } from '../types';

type FieldDefinition = {
  type: FormElementType;
  required: boolean;
  name: string;
  // Add other field properties as needed
};

export function convertApiSchemaToZod(fields: FieldDefinition[]) {
  const shape: Record<string, z.ZodTypeAny> = {};

  if (!fields?.length) {
    return z.object({});
  }

  fields.forEach((field) => {
    let zodType: z.ZodTypeAny;

    switch (field.type) {
      case 'email':
        zodType = z.string().email('Invalid email format');
        break;
      case 'checkbox':
        zodType = z.boolean();
        if (field.required) {
          zodType = zodType.refine((val) => val === true, {
            message: 'Required',
          });
        }
        break;
      case 'url':
        zodType = z.string().url('Invalid URL format');
        break;
      case 'text':
      case 'number':
      case 'textarea':
      case 'password':
      case 'select':
        zodType = z.string();
        if (field.required) {
          zodType = (zodType as z.ZodString).min(1, 'Required');
        }
        break;
      case 'date':
        zodType = z.date();
        if (!field.required) {
          zodType = zodType.optional();
        }
        break;
      default:
        zodType = z.any();
    }

    if (!field.required && field.type !== 'date') {
      zodType = zodType.optional();
    }

    shape[field.name] = zodType;
  });

  return z.object(shape);
}

export const validateFormData = <T extends Record<string, unknown>>(
  data: T,
  schema: z.ZodSchema<T>
) => {
  return schema.safeParse(data);
};
