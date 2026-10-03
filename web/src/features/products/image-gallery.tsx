import { ImagePlus, Loader2, Star, Trash2 } from 'lucide-react'
import { type DragEvent, type ReactNode, useRef, useState } from 'react'
import { toast } from 'sonner'
import { ProductThumb } from '@/components/ui/product-thumb'
import type { Product } from '@/lib/api'
import { useDeleteImage, useSetPrimaryImage, useUploadImage } from '@/lib/queries'
import { cn } from '@/lib/utils'

const ACCEPT = 'image/jpeg,image/png,image/webp,image/gif'
const MAX_BYTES = 5 << 20
const MAX_IMAGES = 8

/** Product image manager: preview, upload (click or drop), delete, set cover. */
export function ImageGallery({ product, editable }: { product: Product; editable: boolean }) {
  const [selected, setSelected] = useState(0)
  const [dragging, setDragging] = useState(false)
  const input = useRef<HTMLInputElement>(null)
  const upload = useUploadImage()
  const remove = useDeleteImage()
  const primary = useSetPrimaryImage()
  const images = product.images
  const current = images[Math.min(selected, images.length - 1)]
  const busy = upload.isPending || remove.isPending || primary.isPending

  const addFiles = async (files: FileList | File[]) => {
    for (const f of Array.from(files)) {
      if (!ACCEPT.split(',').includes(f.type)) {
        toast.error(`${f.name}: use JPEG, PNG, WebP or GIF`)
        continue
      }
      if (f.size > MAX_BYTES) {
        toast.error(`${f.name}: larger than 5 MB`)
        continue
      }
      await upload.mutateAsync({ id: product.id, file: f }).catch(() => {})
    }
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragging(false)
    if (editable && e.dataTransfer.files.length) addFiles(e.dataTransfer.files)
  }

  return (
    <div className="flex flex-col gap-2.5" data-testid="image-gallery">
      <ProductThumb
        category={product.category}
        hue={product.hue}
        src={current?.url}
        alt={current?.alt}
        size={null}
        className="aspect-[4/3] w-full rounded-[14px]"
        iconClassName="size-13 stroke-[1.25]"
      >
        {current && (
          <span className="absolute bottom-2.5 left-2.5 rounded-md bg-black/45 px-2 py-0.5 text-[11px] text-fg/90 backdrop-blur-sm">
            {selected === 0 ? 'Cover · ' : ''}
            {current.alt}
          </span>
        )}
      </ProductThumb>

      <div className="grid grid-cols-5 gap-2 sm:grid-cols-6" data-testid="thumbs">
        {images.map((img, i) => (
          <div key={img.id} className="group relative">
            <button
              type="button"
              onClick={() => setSelected(i)}
              aria-label={`Show image ${i + 1}`}
              className={cn('block w-full overflow-hidden rounded-lg border-2 transition', i === selected ? 'border-accent' : 'border-transparent hover:border-line-2')}
            >
              <img src={img.url} alt={img.alt} loading="lazy" className="aspect-square w-full object-cover" />
            </button>
            {i === 0 && <Star className="absolute top-1 left-1 size-3.5 fill-accent text-accent drop-shadow" aria-label="Cover image" />}
            {editable && (
              <div className="absolute inset-x-1 bottom-1 hidden justify-between group-hover:flex group-focus-within:flex">
                {i > 0 ? (
                  <IconAction label="Make cover" disabled={busy} onClick={() => primary.mutate({ id: product.id, imageId: img.id }, { onSuccess: () => setSelected(0) })}>
                    <Star />
                  </IconAction>
                ) : (
                  <span />
                )}
                <IconAction
                  label="Delete image"
                  danger
                  disabled={busy}
                  onClick={() => remove.mutate({ id: product.id, imageId: img.id }, { onSuccess: () => setSelected((s) => Math.max(0, Math.min(s, images.length - 2))) })}
                >
                  <Trash2 />
                </IconAction>
              </div>
            )}
          </div>
        ))}

        {editable && images.length < MAX_IMAGES && (
          <button
            type="button"
            onClick={() => input.current?.click()}
            onDragOver={(e) => {
              e.preventDefault()
              setDragging(true)
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={onDrop}
            disabled={busy}
            aria-label="Upload images"
            className={cn(
              'grid aspect-square place-items-center rounded-lg border-2 border-dashed text-dim transition hover:border-accent/50 hover:text-accent',
              dragging ? 'border-accent bg-accent/8 text-accent' : 'border-line-2',
            )}
          >
            {upload.isPending ? <Loader2 className="size-5 animate-spin" /> : <ImagePlus className="size-5" />}
          </button>
        )}
      </div>
      {editable && (
        <p className="text-xs text-dim">
          Drop files or click + · JPEG, PNG, WebP, GIF up to 5 MB · {images.length}/{MAX_IMAGES} images · first image is the cover
        </p>
      )}
      <input
        ref={input}
        type="file"
        accept={ACCEPT}
        multiple
        hidden
        data-testid="image-input"
        onChange={(e) => {
          if (e.target.files) addFiles(e.target.files)
          e.target.value = ''
        }}
      />
    </div>
  )
}

function IconAction({ label, onClick, danger, disabled, children }: { label: string; onClick: () => void; danger?: boolean; disabled?: boolean; children: ReactNode }) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        'grid size-6 place-items-center rounded-md bg-black/60 backdrop-blur-sm transition [&_svg]:size-3.5',
        danger ? 'text-danger hover:bg-danger hover:text-white' : 'text-fg hover:bg-accent hover:text-accent-ink',
      )}
    >
      {children}
    </button>
  )
}
