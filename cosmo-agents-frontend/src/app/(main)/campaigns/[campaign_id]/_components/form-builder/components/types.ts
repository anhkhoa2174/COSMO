export type FormElementType =
  | 'text'
  | 'number'
  | 'email'
  | 'password'
  | 'textarea'
  | 'select'
  | 'checkbox'
  | 'radio'
  | 'date'
  | 'spacer'
  | 'divider'
  | 'url';

export interface FormElementBase {
  id: string;
  type: FormElementType;
  label: string;
  name: string;
  required: boolean;
  placeholder?: string;
  className?: string;
  description?: string;
  minLength?: number;
  maxLength?: number;
  defaultChecked?: boolean;
  rows?: number;
  defaultValue?: string;
  colSpan?: number;
  styles?: {
    [key: string]: string;
  };
}

export interface TextFieldElement extends FormElementBase {
  type: 'text' | 'email' | 'password' | 'number' | 'url';
  defaultValue?: string;
  minLength?: number;
  maxLength?: number;
}

export interface TextareaElement extends FormElementBase {
  type: 'textarea';
  defaultValue?: string;
  rows?: number;
}

export interface SelectElement extends FormElementBase {
  type: 'select';
  options: { label: string; value: string }[];
  defaultValue?: string;
  multiple?: boolean;
}

export interface CheckboxElement extends FormElementBase {
  type: 'checkbox';
  defaultChecked?: boolean;
}

export interface RadioElement extends FormElementBase {
  type: 'radio';
  options: { label: string; value: string }[];
  defaultValue?: string;
}

export interface DateElement extends FormElementBase {
  type: 'date';
  defaultValue?: string;
}

export interface SpacerElement extends FormElementBase {
  type: 'spacer';
  styles: {
    height: string; // e.g., '16px', '2rem'
    backgroundColor?: string; // Optional background color
  };
}

export interface DividerElement extends FormElementBase {
  type: 'divider';
  styles: {
    color?: string; // e.g., '#000000', 'gray-300'
    thickness?: string; // e.g., '1px', '0.1rem'
    width?: string; // e.g., '100%', '50%', '200px'
    marginTop?: string; // e.g., '8px', '0.5rem'
    marginBottom?: string; // e.g., '8px', '0.5rem'
  };
}

export interface UrlElement extends FormElementBase {
  type: 'url';
  defaultValue?: string;
}

export type FormElement =
  | TextFieldElement
  | TextareaElement
  | SelectElement
  | CheckboxElement
  | RadioElement
  | DateElement
  | SpacerElement
  | DividerElement
  | UrlElement;

export interface SubmitButtonAppearance {
  backgroundColor: string;
  textColor: string;
  fullWidth: boolean;
  alignment: 'left' | 'center' | 'right';
}

export interface FormSettings {
  title: string;
  description?: string;
  publicUrl?: { origin: string; path: string; name: string };
  submitButtonText: string;
  submitButtonAppearance: SubmitButtonAppearance;
  gridColumns?: string;
  styles: {
    margin: string;
    padding: string;
    backgroundColor: string;
    gap: string;
    borderRadius?: string;
    border?: string;
    boxShadow?: string;
  };
}

export interface FormState {
  elements: FormElement[];
  settings: FormSettings;
}
