import { CircleCheck, Download, FileSpreadsheet, TriangleAlert, Upload } from 'lucide-react'
import { type DragEvent, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Sheet } from '@/components/ui/sheet'
import { ApiError, exportUrl, type ImportResult, type ImportRowError } from '@/lib/api'
import { downloadUrl } from '@/lib/download'
import { errorMessage, useImportProducts } from '@/lib/queries'
import { cn } from '@/lib/utils'

const COLUMNS: [string, string][] = [
  ['sku', 'Required. Existing SKUs are updated, new ones created'],
  ['name, category', 'Required for new products'],
  ['price, compare_at_price, cost', 'Dollars, e.g. 129.99'],
  ['stock, weight_grams', 'Whole numbers'],
  ['status', 'active, draft (default) or archived'],
  ['vendor, tags, description', 'Tags separated by “;”'],
]

/** Upload a CSV to create or update products by SKU. The export is a ready-made template. */
export function ImportSheet({ onClose }: { onClose: () => void }) {
  const input = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [drag, setDrag] = useState(false)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [problems, setProblems] = useState<{ message: string; rows: ImportRowError[] } | null>(null)
  const importProducts = useImportProducts()

  const pick = (f: File | undefined) => {
    if (!f) return
    setFile(f)
    setResult(null)
    setProblems(null)
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    setDrag(false)
    pick(e.dataTransfer.files[0])
  }

  const submit = () =>
    file &&
    importProducts.mutate(file, {
      onSuccess: setResult,
      onError: (err) =>
        setProblems({
          message: errorMessage(err),
          rows: err instanceof ApiError && Array.isArray(err.body?.rows) ? (err.body.rows as ImportRowError[]) : [],
        }),
    })

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      title="Import products"
      description="CSV, up to 1,000 rows and 2 MB. Rows are matched by SKU."
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            {result ? 'Done' : 'Cancel'}
          </Button>
          {!result && (
            <Button variant="primary" disabled={!file || importProducts.isPending} onClick={submit}>
              <Upload />
              {importProducts.isPending ? 'Importing…' : 'Import'}
            </Button>
          )}
        </>
      }
    >
      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDrag(true)
        }}
        onDragLeave={() => setDrag(false)}
        onDrop={onDrop}
        onClick={() => input.current?.click()}
        className={cn(
          'flex cursor-pointer flex-col items-center gap-2 rounded-xl border border-dashed border-line-2 px-4 py-8 text-center transition-colors hover:border-accent/50 hover:bg-panel-2/50',
          drag && 'border-accent bg-accent/5',
        )}
      >
        <FileSpreadsheet className="size-7 text-accent" />
        {file ? (
          <>
            <b className="font-medium">{file.name}</b>
            <small className="num text-xs text-dim">{(file.size / 1024).toFixed(1)} KB · click to choose another</small>
          </>
        ) : (
          <>
            <b className="font-medium">Drop a CSV file here</b>
            <small className="text-xs text-dim">or click to browse</small>
          </>
        )}
        <input
          ref={input}
          type="file"
          accept=".csv,text/csv"
          className="hidden"
          data-testid="csv-input"
          onChange={(e) => {
            pick(e.target.files?.[0])
            e.target.value = ''
          }}
        />
      </div>

      {result && (
        <div className="flex items-start gap-2.5 rounded-xl border border-accent/30 bg-accent/5 px-3.5 py-3">
          <CircleCheck className="mt-0.5 size-4 text-accent" />
          <div>
            <b className="font-medium">Import complete</b>
            <p className="text-[12.5px] text-muted">
              {result.created} created · {result.updated} updated
            </p>
          </div>
        </div>
      )}

      {problems && (
        <div className="rounded-xl border border-danger/30 bg-danger/5 px-3.5 py-3">
          <div className="flex items-center gap-2 font-medium text-danger">
            <TriangleAlert className="size-4" />
            {problems.message}
          </div>
          {problems.rows.length > 0 && (
            <ul className="mt-2 flex max-h-56 flex-col gap-1 overflow-y-auto text-[12.5px]">
              {problems.rows.map((r, i) => (
                <li key={i} className="flex gap-2">
                  <span className="num shrink-0 text-dim">Row {r.row}</span>
                  {r.sku && <code className="num shrink-0 text-muted">{r.sku}</code>}
                  <span>
                    <b className="font-medium">{r.field}</b> {r.message}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <div>
        <div className="mb-2 flex items-center">
          <div className="eyebrow">Columns</div>
          <Button size="sm" variant="ghost" className="ml-auto" onClick={() => downloadUrl(exportUrl('products'))}>
            <Download />
            Template (current catalogue)
          </Button>
        </div>
        <div className="overflow-hidden rounded-xl border border-line">
          {COLUMNS.map(([cols, hint]) => (
            <div key={cols} className="flex gap-3 border-b border-line px-3.5 py-2 text-[12.5px] last:border-0">
              <code className="num w-44 shrink-0 text-fg">{cols}</code>
              <span className="text-muted">{hint}</span>
            </div>
          ))}
        </div>
        <p className="mt-2 text-xs text-dim">
          Header names can be in any order; other columns (id, sold_30d, image_url…) are ignored. If any row is invalid, nothing is imported.
        </p>
      </div>
    </Sheet>
  )
}
