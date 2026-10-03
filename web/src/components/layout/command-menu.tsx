import { useNavigate } from '@tanstack/react-router'
import { Command } from 'cmdk'
import { Plus, Search, UserPlus } from 'lucide-react'
import { type ReactNode, useEffect } from 'react'
import { CATEGORY_ICON } from '@/components/ui/product-thumb'
import { NAV_ITEMS } from '@/lib/nav'
import { useProducts } from '@/lib/queries'

const itemClass =
  'flex cursor-pointer items-center gap-2.5 rounded-[9px] px-2.5 py-2.5 text-muted data-[selected=true]:bg-panel-3 data-[selected=true]:text-fg [&_svg]:size-4 data-[selected=true]:[&_svg]:text-accent'
const groupClass =
  '[&_[cmdk-group-heading]]:num [&_[cmdk-group-heading]]:px-2.5 [&_[cmdk-group-heading]]:pt-2.5 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:text-[10.5px] [&_[cmdk-group-heading]]:tracking-widest [&_[cmdk-group-heading]]:text-dim [&_[cmdk-group-heading]]:uppercase'

export function CommandMenu({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const navigate = useNavigate()
  const { data } = useProducts({})

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        onOpenChange(!open)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onOpenChange])

  const run = (fn: () => void) => () => {
    onOpenChange(false)
    fn()
  }

  return (
    <Command.Dialog
      open={open}
      onOpenChange={onOpenChange}
      label="Command menu"
      overlayClassName="fixed inset-0 z-60 animate-fade-in bg-[rgb(4_5_6/0.6)] backdrop-blur-[3px]"
      contentClassName="fixed top-[14vh] left-1/2 z-70 w-[min(620px,calc(100vw-24px))] -translate-x-1/2 animate-pop-in overflow-hidden rounded-2xl border border-line-2 bg-panel shadow-float"
    >
      <div className="flex items-center gap-2.5 border-b border-line px-4">
        <Search className="size-4 text-dim" />
        <Command.Input placeholder="Type a command or search…" className="h-13 flex-1 bg-transparent text-[15px] outline-none placeholder:text-dim" />
        <kbd className="num rounded-md border border-line-2 bg-panel-2 px-1.5 text-[11px] text-muted">Esc</kbd>
      </div>
      <Command.List className="max-h-90 overflow-y-auto p-1.5">
        <Command.Empty className="py-7 text-center text-muted">No results</Command.Empty>
        <Command.Group heading="Pages" className={groupClass}>
          {NAV_ITEMS.map((item) => (
            <Item key={item.to} onSelect={run(() => navigate({ to: item.to }))} icon={<item.icon />}>
              {item.label}
            </Item>
          ))}
        </Command.Group>
        <Command.Group heading="Actions" className={groupClass}>
          <Item onSelect={run(() => navigate({ to: '/products', search: { edit: 'new' } }))} icon={<Plus />}>
            Add product
          </Item>
          <Item onSelect={run(() => navigate({ to: '/team', search: { edit: 'new' } }))} icon={<UserPlus />}>
            Invite team member
          </Item>
        </Command.Group>
        <Command.Group heading="Products" className={groupClass}>
          {data?.items.map((p) => {
            const Icon = CATEGORY_ICON[p.category]
            return (
              <Item
                key={p.id}
                value={`${p.name} ${p.sku}`}
                onSelect={run(() => navigate({ to: '/products', search: { edit: p.id } }))}
                icon={<Icon />}
                hint={p.sku}
              >
                {p.name}
              </Item>
            )
          })}
        </Command.Group>
      </Command.List>
      <div className="flex gap-3.5 border-t border-line px-3.5 py-2.5 text-[11.5px] text-dim">
        <span>↑↓ navigate</span>
        <span>↵ open</span>
        <span className="num ml-auto">GoAdmin</span>
      </div>
    </Command.Dialog>
  )
}

function Item({ children, icon, hint, value, onSelect }: { children: ReactNode; icon: ReactNode; hint?: string; value?: string; onSelect: () => void }) {
  return (
    <Command.Item value={value} onSelect={onSelect} className={itemClass}>
      {icon}
      {children}
      {hint && <small className="num ml-auto text-[11px] text-dim">{hint}</small>}
    </Command.Item>
  )
}
