import { Backpack, Headphones, House, LampDesk, Laptop, type LucideIcon, Watch } from 'lucide-react'
import type { CSSProperties, ReactNode } from 'react'
import type { Category } from '@/lib/api'
import { imageAt } from '@/lib/image'
import { cn } from '@/lib/utils'

export const CATEGORY_ICON: Record<Category, LucideIcon> = {
  Audio: Headphones,
  Wearables: Watch,
  Lighting: LampDesk,
  Home: House,
  Computing: Laptop,
  Accessories: Backpack,
}

/**
 * Product picture: the image when there is one, otherwise the category icon
 * over a hue-tinted gradient.
 */
export function ProductThumb({
  category,
  hue,
  src,
  alt = '',
  size = 36,
  srcSize,
  className,
  iconClassName,
  children,
}: {
  category: Category
  hue: number
  src?: string
  alt?: string
  /** Pixel size; pass null to size via className instead. */
  size?: number | null
  /** Drawn width in CSS px when size is null, to fetch a fitting image. */
  srcSize?: number
  className?: string
  iconClassName?: string
  children?: ReactNode
}) {
  const Icon = CATEGORY_ICON[category]
  const style: CSSProperties = {
    ...(size ? { width: size, height: size } : {}),
    color: `hsl(${hue} 85% 74%)`,
    borderColor: `hsl(${hue} 60% 60% / 0.18)`,
    background: `radial-gradient(circle at 30% 20%, hsl(${hue} 80% 60% / 0.35), transparent 60%), linear-gradient(135deg, hsl(${hue} 50% 22%), hsl(${hue} 40% 11%))`,
  }
  return (
    <div className={cn('relative grid shrink-0 place-items-center overflow-hidden rounded-[10px] border', className)} style={style}>
      {src ? (
        <img
          src={imageAt(src, size ?? srcSize ?? 400)}
          alt={alt}
          loading="lazy"
          decoding="async"
          draggable={false}
          className="absolute inset-0 size-full object-cover"
        />
      ) : (
        <Icon className={iconClassName} style={size ? { width: size * 0.46, height: size * 0.46 } : undefined} />
      )}
      {children}
    </div>
  )
}
