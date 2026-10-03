// Runs the Playwright suite against a fresh Postgres database (goadmin_e2e)
// in the docker compose service. The Go server migrates and seeds it on boot.
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const psql = (sql) =>
  execFileSync('docker', ['compose', 'exec', '-T', 'postgres', 'psql', '-U', 'goadmin', '-d', 'postgres', '-c', sql], {
    cwd: new URL('../..', import.meta.url),
    stdio: 'inherit',
  })

psql('DROP DATABASE IF EXISTS goadmin_e2e WITH (FORCE)')
psql('CREATE DATABASE goadmin_e2e OWNER goadmin')

// Run the Playwright CLI through node directly — portable, no shell needed.
const cli = fileURLToPath(new URL('../node_modules/@playwright/test/cli.js', import.meta.url))
execFileSync(process.execPath, [cli, 'test', ...process.argv.slice(2)], {
  stdio: 'inherit',
  env: { ...process.env, E2E_DATABASE_URL: 'postgres://goadmin:goadmin@localhost:5433/goadmin_e2e?sslmode=disable' },
})
