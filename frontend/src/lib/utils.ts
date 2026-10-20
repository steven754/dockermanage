import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

/** 合并 className（clsx + tailwind-merge，后者负责去冲突） */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}
