import { UserPlus } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input, Select } from '@/components/ui/input'
import { Switch } from '@/components/ui/misc'
import { Sheet } from '@/components/ui/sheet'
import { TagInput } from '@/components/ui/tag-input'
import { ApiError, type Customer } from '@/lib/api'
import { useCreateCustomer } from '@/lib/queries'

const COUNTRIES: [string, string][] = [
  ['US', 'United States'], ['DE', 'Germany'], ['GB', 'United Kingdom'], ['FR', 'France'], ['NL', 'Netherlands'], ['PL', 'Poland'],
  ['CA', 'Canada'], ['ES', 'Spain'], ['IT', 'Italy'], ['SE', 'Sweden'], ['JP', 'Japan'], ['AU', 'Australia'],
]
const SOURCES = ['Manual', 'Organic search', 'Newsletter', 'Instagram', 'Referral', 'Direct', 'Phone order']

/** Adds a customer by hand, e.g. from a phone order. They start in the "New" segment. */
export function CustomerForm({ onClose, onCreated }: { onClose: () => void; onCreated: (c: Customer) => void }) {
  const [f, setF] = useState({
    name: '', email: '', phone: '', country: 'US', line1: '', city: '', postalCode: '', source: 'Manual', tags: [] as string[], acceptsMarketing: false,
  })
  const create = useCreateCustomer()
  const errors = create.error instanceof ApiError ? create.error.fields : undefined
  const set = <K extends keyof typeof f>(k: K, v: (typeof f)[K]) => setF((s) => ({ ...s, [k]: v }))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    create.mutate(
      {
        name: f.name, email: f.email, phone: f.phone, country: f.country, tags: f.tags, acceptsMarketing: f.acceptsMarketing, source: f.source,
        address: { line1: f.line1, city: f.city, postalCode: f.postalCode, country: f.country },
      },
      { onSuccess: onCreated },
    )
  }

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      title="Add customer"
      description="For orders taken by phone or in person. The customer starts in the New segment."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" type="submit" form="customer-form" disabled={create.isPending || !f.name.trim() || !f.email.trim()}>
            <UserPlus />
            {create.isPending ? 'Adding…' : 'Add customer'}
          </Button>
        </>
      }
    >
      <form id="customer-form" onSubmit={submit} className="flex flex-col gap-4">
        <Field label="Name" error={errors?.name}>
          <Input value={f.name} onChange={(e) => set('name', e.target.value)} placeholder="Full name" autoFocus maxLength={80} />
        </Field>
        <div className="grid gap-3.5 sm:grid-cols-2">
          <Field label="Email" error={errors?.email}>
            <Input type="email" value={f.email} onChange={(e) => set('email', e.target.value)} placeholder="name@example.com" />
          </Field>
          <Field label="Phone" error={errors?.phone}>
            <Input className="num" value={f.phone} onChange={(e) => set('phone', e.target.value)} placeholder="+49 30 1234567" />
          </Field>
        </div>
        <Field label="Street address" error={errors?.address}>
          <Input value={f.line1} onChange={(e) => set('line1', e.target.value)} />
        </Field>
        <div className="grid gap-3.5 sm:grid-cols-3">
          <Field label="City">
            <Input value={f.city} onChange={(e) => set('city', e.target.value)} />
          </Field>
          <Field label="Postal code">
            <Input className="num" value={f.postalCode} onChange={(e) => set('postalCode', e.target.value)} />
          </Field>
          <Field label="Country" error={errors?.country}>
            <Select value={f.country} onChange={(e) => set('country', e.target.value)}>
              {COUNTRIES.map(([code, name]) => (
                <option key={code} value={code}>
                  {name}
                </option>
              ))}
            </Select>
          </Field>
        </div>
        <div className="grid gap-3.5 sm:grid-cols-2">
          <Field label="Acquired via" error={errors?.source}>
            <Select value={f.source} onChange={(e) => set('source', e.target.value)}>
              {SOURCES.map((s) => (
                <option key={s}>{s}</option>
              ))}
            </Select>
          </Field>
          <Field label="Tags" error={errors?.tags} hint="Enter or comma to add">
            <TagInput value={f.tags} onChange={(v) => set('tags', v)} />
          </Field>
        </div>
        <div className="flex items-center justify-between gap-3.5 rounded-xl border border-line bg-panel px-3.5 py-3">
          <div>
            <b className="block font-medium">Accepts marketing</b>
            <small className="text-xs text-dim">Included when you email a segment.</small>
          </div>
          <Switch checked={f.acceptsMarketing} onChange={(v) => set('acceptsMarketing', v)} label="Accepts marketing" />
        </div>
      </form>
    </Sheet>
  )
}
