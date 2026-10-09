import { atom, useAtom } from 'jotai';
import type { Conversation } from '@/models/conversation';
import type { Member } from '@/models/organization';

type Config = {
  selected: Conversation | null;
  members: Member[];
};

const configAtom = atom<Config>({
  selected: null,
  members: [],
});

export function useConversation() {
  return useAtom(configAtom);
}
