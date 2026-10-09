import { generateId } from '@/helpers';

const origin =
  process.env.NODE_ENV === 'development'
    ? 'http://localhost:3000'
    : 'https://cosmoagents.ai';

const getPreviewUrl = (formState: any) => {
  return `${origin}/${formState?.settings?.publicUrl?.path || ''}/${formState?.settings?.publicUrl?.name}`;
};

const settings = {
  settings: {
    title: 'Request Demo',
    description: 'Please fill out this form to get in touch with us',
    submitButtonText: 'Submit',
    gridColumns: '4',
    publicUrl: {
      origin: origin,
      path: 'public/forms',
      name: `new-customer-form-${generateId()}`,
    },
    submitButtonAppearance: {
      backgroundColor: '#071F1D',
      textColor: '#ffffff',
      fullWidth: true,
      alignment: 'right',
    },
    styles: {
      margin: '0px',
      padding: '16px',
      backgroundColor: '#ffffff',
      gap: '16px',
      borderRadius: '8px',
      border: '1px solid #e0e0e0',
      boxShadow: 'rgba(149, 157, 165, 0.2) 0px 8px 24px',
    },
  },
};

export { getPreviewUrl, origin, settings };
