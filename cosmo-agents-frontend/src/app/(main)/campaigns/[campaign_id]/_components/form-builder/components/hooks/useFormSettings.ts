import { FormSettings, SubmitButtonAppearance } from "../types";
import { useState } from "react";

export const useFormSettings = (initialSettings: FormSettings) => {
  const [currentSettings, setCurrentSettings] = useState<FormSettings>(initialSettings);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const handleSettingChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
    checkedValue?: boolean
  ) => {
    const { name, value } = e.target;

    if (name === 'gridColumns') {
      setCurrentSettings(prev => ({
        ...prev,
        [name]: Number(value) > 4 ? '4' : String(value),
      }));
    } else if (name === 'title' || name === 'description' || name === 'submitButtonText') {
      setCurrentSettings(prev => ({ ...prev, [name]: value }));
    } else if (name.startsWith('sba_')) {
      // Handle submit button appearance changes
      const propName = name.split('_')[1].replace('1', '');
      setCurrentSettings(prev => ({
        ...prev,
        submitButtonAppearance: {
          ...prev.submitButtonAppearance,
          [propName]: propName === 'fullWidth' ? checkedValue ?? (e.target as HTMLInputElement).checked : value,
        },
      }));
    } else {
      // Handle style changes
      const cleanName = name.replace('1', '');
      setCurrentSettings(prev => ({
        ...prev,
        styles: { ...prev.styles, [cleanName]: value },
      }));
    }
  };

  const handleAlignmentChange = (value: string) => {
    setCurrentSettings(prev => ({
      ...prev,
      submitButtonAppearance: {
        ...prev.submitButtonAppearance,
        alignment: value as SubmitButtonAppearance['alignment'],
      },
    }));
  };

  return {
    currentSettings,
    isDialogOpen,
    setCurrentSettings,
    setIsDialogOpen,
    handleSettingChange,
    handleAlignmentChange
  };
};