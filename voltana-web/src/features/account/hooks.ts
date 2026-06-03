import { useMutation } from "@tanstack/react-query";
import { requestBotLink } from "./api";

export function useBotLink() {
  return useMutation({ mutationFn: requestBotLink });
}
