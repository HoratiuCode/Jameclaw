import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  type UpdateStatusResponse,
  dismissUpdate,
  getUpdateStatus,
  openUpdatePage,
} from "@/api/update"

export function useUpdateStatus() {
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ["update-status"],
    queryFn: getUpdateStatus,
    staleTime: 15 * 60_000,
    refetchInterval: 60 * 60_000,
    retry: 1,
  })

  const dismissMutation = useMutation({
    mutationFn: dismissUpdate,
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ["update-status"] })
      const previous = queryClient.getQueryData<UpdateStatusResponse>([
        "update-status",
      ])
      queryClient.setQueryData<UpdateStatusResponse>(
        ["update-status"],
        previous ? { ...previous, dismissed: true } : previous,
      )
      return { previous }
    },
    onError: (_error, _version, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["update-status"], context.previous)
      }
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["update-status"] })
    },
  })

  const openMutation = useMutation({
    mutationFn: openUpdatePage,
  })

  return {
    ...query,
    dismiss: dismissMutation.mutate,
    dismissing: dismissMutation.isPending,
    openUpdate: openMutation.mutate,
    opening: openMutation.isPending,
  }
}
