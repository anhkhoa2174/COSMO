import { atom } from 'jotai';
import type { CategoryId } from '@/types/daily-actions';
import type { Language } from '@/network/client/outreach';

/** Which contact detail panel is open (null = closed) */
export const activePanelAtom = atom<{ contactId: string } | null>(null);

/** Which category sections are currently expanded */
export const expandedCategoriesAtom = atom<Set<CategoryId>>(
  new Set<CategoryId>()
);

/** Preserved chat input draft text */
export const chatDraftAtom = atom<string>('');


/** Language preference for generated content */
export const languageAtom = atom<Language>('vi');
