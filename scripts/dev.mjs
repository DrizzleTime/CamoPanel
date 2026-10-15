import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const children = [
  spawn('go', ['run', './cmd/server'], { cwd: `${root}backend`, stdio: 'inherit', detached: process.platform !== 'win32' }),
  spawn(process.execPath, ['run', 'dev'], { cwd: `${root}frontend`, stdio: 'inherit', detached: process.platform !== 'win32' }),
];
let stopping = false;
function stop(code = 0) {
  if (stopping) return;
  stopping = true;
  for (const child of children) {
    try { if (process.platform !== 'win32' && child.pid) process.kill(-child.pid, 'SIGTERM'); else child.kill('SIGTERM'); }
    catch { /* A process may already have exited. */ }
  }
  process.exitCode = code;
}
for (const child of children) {
  child.on('error', err => { console.error(err.message); stop(1); });
  child.on('exit', (code, signal) => { if (!stopping) stop(code ?? (signal ? 1 : 0)); });
}
process.on('SIGINT', () => stop());
process.on('SIGTERM', () => stop());
