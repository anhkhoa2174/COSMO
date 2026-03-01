import { useQuery } from '@tanstack/react-query';
import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export type WatchGmailData = {
  history_id: string;
  expiration: number;
};
type WatchGmailResponse = ApiResponse<WatchGmailData>;

export const watchGmail = () => {
  return kyClient.post<WatchGmailResponse>('v2/google/gmail/watch');
};

export const useWatchGmailQuery = () => {
  return useQuery({
    queryKey: ['google', 'gmail', 'watch'],
    queryFn: watchGmail,
  });
};
