import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listCars, createCar, updateCar, deleteCar, type CarInput } from "./api";

export function useCars() {
  return useQuery({ queryKey: ["cars"], queryFn: listCars });
}

export function useCreateCar() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CarInput) => createCar(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cars"] }),
  });
}

export function useUpdateCar() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: CarInput }) => updateCar(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cars"] }),
  });
}

export function useDeleteCar() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteCar(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cars"] }),
  });
}
