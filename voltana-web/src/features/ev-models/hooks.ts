import { useQuery } from "@tanstack/react-query";
import { listEVModels } from "./api";

export function useEVModels() {
  return useQuery({ queryKey: ["ev-models"], queryFn: listEVModels });
}
