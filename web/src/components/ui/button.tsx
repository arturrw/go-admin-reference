import { cva, type VariantProps } from 'class-variance-authority'
import type { ComponentProps } from 'react'
import { cn } from '@/lib/utils'

export const buttonVariants = cva(
  'inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-[10px] text-[13px] font-medium transition-colors disabled:pointer-events-none disabled:opacity-45 [&_svg]:size-4 [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default: 'border border-line-2 bg-panel-2 text-fg hover:border-white/16 hover:bg-panel-3',
        primary: 'bg-accent text-accent-ink shadow-glow hover:bg-accent/88',
        ghost: 'text-muted hover:bg-panel-2 hover:text-fg',
        danger: 'border border-line-2 bg-panel-2 text-danger hover:border-danger/30 hover:bg-danger/12',
      },
      size: {
        md: 'h-[34px] px-3.5',
        sm: 'h-7 rounded-lg px-2.5 text-[12.5px]',
        icon: 'size-[34px]',
        'icon-sm': 'size-7 rounded-lg',
      },
    },
    defaultVariants: { variant: 'default', size: 'md' },
  },
)

export type ButtonProps = ComponentProps<'button'> & VariantProps<typeof buttonVariants>

export function Button({ className, variant, size, type = 'button', ...props }: ButtonProps) {
  return <button type={type} className={cn(buttonVariants({ variant, size }), className)} {...props} />
}
