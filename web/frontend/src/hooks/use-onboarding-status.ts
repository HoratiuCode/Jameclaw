import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  completeOnboarding,
  getOnboardingStatus,
  resetOnboarding,
} from "@/api/onboarding"

export function useOnboardingStatus() {
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ["onboarding-status"],
    queryFn: getOnboardingStatus,
    refetchInterval: 5000,
  })

  const completeMutation = useMutation({
    mutationFn: completeOnboarding,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["onboarding-status"] })
    },
  })

  const resetMutation = useMutation({
    mutationFn: resetOnboarding,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["onboarding-status"] })
    },
  })

  return {
    ...query,
    completeOnboarding: completeMutation.mutateAsync,
    resetOnboarding: resetMutation.mutateAsync,
    completing: completeMutation.isPending,
    resetting: resetMutation.isPending,
  }
}
