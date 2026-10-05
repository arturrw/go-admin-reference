/**
 * Starts a browser download. Same-origin API URLs send the session cookie and
 * name the file via Content-Disposition; blob URLs need `filename`.
 */
export function downloadUrl(url: string, filename = '') {
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.append(a)
  a.click()
  a.remove()
}

const cell = (v: unknown) => {
  const s = v === null || v === undefined ? '' : String(v)
  return /[",\n\r]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

/** Builds a CSV file in the browser and downloads it as `<name>-<date>.csv`. */
export function downloadCsv(name: string, rows: unknown[][]) {
  // BOM so Excel reads the file as UTF-8.
  const text = '\uFEFF' + rows.map((r) => r.map(cell).join(',')).join('\r\n') + '\r\n'
  const url = URL.createObjectURL(new Blob([text], { type: 'text/csv;charset=utf-8' }))
  downloadUrl(url, `${name}-${new Date().toISOString().slice(0, 10)}.csv`)
  setTimeout(() => URL.revokeObjectURL(url), 10_000)
}
