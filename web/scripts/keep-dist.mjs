// `vite build` empties dist/, but Go's //go:embed needs the directory to exist
// in a fresh clone, so we keep a tracked placeholder file in it.
import { writeFileSync } from 'node:fs'

writeFileSync(new URL('../dist/.gitkeep', import.meta.url), '')
