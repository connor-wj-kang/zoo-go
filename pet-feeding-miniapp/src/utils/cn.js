import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * className 合并工具
 * 用于合并 Tailwind 类名，避免冲突
 */
export function cn(...inputs) {
  return twMerge(clsx(inputs))
}
