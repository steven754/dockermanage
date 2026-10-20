import { forwardRef, type HTMLAttributes, type InputHTMLAttributes, type LabelHTMLAttributes, type TextareaHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

/** 输入框（对应设计稿 .input） */
export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        'w-full rounded-md border border-border bg-bg-base px-2.5 py-1.5 text-[13px] text-fg-primary outline-none transition-[border-color,box-shadow] placeholder:text-fg-subtle focus:border-brand focus:ring-[3px] focus:ring-brand/10 disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    />
  ),
)
Input.displayName = 'Input'

/** 多行输入 */
export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        'w-full rounded-md border border-border bg-bg-base px-2.5 py-1.5 text-[13px] text-fg-primary outline-none transition-[border-color,box-shadow] placeholder:text-fg-subtle focus:border-brand focus:ring-[3px] focus:ring-brand/10 disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    />
  ),
)
Textarea.displayName = 'Textarea'

/** 表单标签 */
export const Label = forwardRef<HTMLLabelElement, LabelHTMLAttributes<HTMLLabelElement>>(
  ({ className, ...props }, ref) => (
    <label
      ref={ref}
      className={cn('text-[13px] font-medium text-fg-secondary', className)}
      {...props}
    />
  ),
)
Label.displayName = 'Label'

/** 骨架屏 */
export const Skeleton = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('skeleton', className)} {...props} />
  ),
)
Skeleton.displayName = 'Skeleton'
