'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { User } from '@/models/auth';
import type { ApiResponse } from '@/models/response';
import { authService } from '@/services/authService';

// Query keys
export const userKeys = {
  all: ['user'] as const,
  me: () => [...userKeys.all, 'me'] as const,
};

/**
 * Hook to fetch current user data
 */
export function useGetMe(enabled: boolean = true) {
  return useQuery<ApiResponse<User>, Error>({
    queryKey: userKeys.me(),
    queryFn: () => authService.getMe(),
    enabled,
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (previously cacheTime)
    retry: (failureCount, error) => {
      // Don't retry on 401 or 403 errors
      if (error.message.includes('401') || error.message.includes('403')) {
        return false;
      }
      return failureCount < 2;
    },
  });
}

/**
 * Hook to get user data from cache or fetch if needed
 */
export function useUser() {
  const queryClient = useQueryClient();
  const { data, isLoading, error, refetch } = useGetMe();

  return {
    user: data?.data ?? null,
    isLoading,
    error,
    refetch,
    invalidate: () =>
      queryClient.invalidateQueries({ queryKey: userKeys.me() }),
  };
}

/**
 * Hook to update user data in cache
 */
export function useUpdateUserCache() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (user: User) => {
      return user;
    },
    onSuccess: (user) => {
      // Update the cache with new user data
      queryClient.setQueryData<ApiResponse<User>>(userKeys.me(), (old) => ({
        ...old,
        data: user,
        status: 'success',
      }));
    },
  });
}
