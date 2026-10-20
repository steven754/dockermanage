import { forwardRef } from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  'inline-flex items-center gap-1.5 rounded-md border text-[13px] font-medium transition-colors select-none disabled:pointer-events-none disabled:opacity-50 whitespace-nowrap',
  {
    variants: {
      variant: {
        default:
          'border-border bg-bg-card text-fg-primary hover:bg-bg-cardHover hover:border-border-strong',
        primary: 'border-brand bg-brand text-white hover:bg-brand-hover hover:border-brand-hover',
        outline: 'border-border bg-transparent text-fg-secondary hover:bg-bg-card hover:text-fg-primary',
        ghost: 'border-transparent bg-transparent text-fg-muted hover:bg-bg-cardHover hover:text-fg-primary',
        danger: 'border-danger/30 bg-danger-soft text-danger hover:bg-danger/20',
        success: 'border-success/30 bg-success-soft text-success hover:bg-success/20',
      },
      size: {
        sm: 'h-7 px-2.5 text-xs',
        default: 'h-8 px-3',
        lg: 'h-9 px-4 text-sm',
        icon: 'h-8 w-8 justify-center',
        'icon-sm': 'h-7 w-7 justify-center',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, type, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        ref={ref}
        type={asChild ? undefined : (type ?? 'button')}
        className={cn(buttonVariants({ variant, size }), className)}
        {...props}
      />
    )
  },
)
Button.displayName = 'Button'

export { buttonVariants }
