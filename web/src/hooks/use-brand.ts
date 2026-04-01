import { createContext, useContext } from "react";
import type { BrandInfo } from "@/api";

export const BrandContext = createContext<BrandInfo>(null as unknown as BrandInfo);

export function useBrand(): BrandInfo {
  return useContext(BrandContext);
}
