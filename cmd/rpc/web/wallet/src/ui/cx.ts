import { twMerge } from 'tailwind-merge'
import clsx from 'clsx'
export const cx = (...args: any[]) => twMerge(clsx(args))
