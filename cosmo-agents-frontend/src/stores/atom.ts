import { atom } from 'jotai';
import type { Contact } from '@/models/contact';

export const isChangeContact = atom<boolean>(false);
export const currentPageAtom = atom<{ [key: string]: number }>({});
export const selectedIdAtom = atom<{ [key: string]: string[] }>({});
export const selectedRowAtom = atom<Contact | null>(null);
