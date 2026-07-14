// Persists an in-flight block run to localStorage so the document view can
// restore the running state and partial streamed text after a page reload.
// Backend runs are detached — they finish and save server-side even if the
// browser disconnects — so a restored card only needs to wait for the saved
// response to appear. All storage access is wrapped in try/catch: when
// localStorage is unavailable (private mode, quota), runs behave as before,
// they just don't survive a reload.

export interface RunEntry {
  blockId: number
  documentId: number
  // Date.now() when the run started.
  startedAt: number
  // block.responses.length before the run; the run is complete once the
  // block has more responses than this.
  baseResponseCount: number
  // Streamed preview text persisted so far (what the user saw before reload).
  text: string
}

// Mirrors the backend's maxRunDuration: a run older than this can no longer
// produce a response, so its entry is dropped as stale.
export const RUN_STALE_MS = 15 * 60 * 1000

// localStorage writes are synchronous, so per-delta appends are batched.
const WRITE_THROTTLE_MS = 500

function storageKey(documentId: number, blockId: number): string {
  return `nisaba.run.v1.${documentId}.${blockId}`
}

export function isRunStale(entry: RunEntry, now = Date.now()): boolean {
  return now - entry.startedAt > RUN_STALE_MS
}

function isRunEntry(value: unknown): value is RunEntry {
  if (typeof value !== 'object' || value === null) return false
  const v = value as Record<string, unknown>
  return (
    typeof v.blockId === 'number' &&
    typeof v.documentId === 'number' &&
    typeof v.startedAt === 'number' &&
    typeof v.baseResponseCount === 'number' &&
    typeof v.text === 'string'
  )
}

// loadRunEntry returns the persisted entry for a block, or null when there is
// none — including corrupt, wrong-shaped, and stale entries, which it removes.
export function loadRunEntry(documentId: number, blockId: number): RunEntry | null {
  let raw: string | null
  try {
    raw = localStorage.getItem(storageKey(documentId, blockId))
  } catch {
    return null
  }
  if (raw === null) return null
  let entry: RunEntry | null = null
  try {
    const parsed: unknown = JSON.parse(raw)
    if (isRunEntry(parsed)) entry = parsed
  } catch {
    // corrupt JSON — treated like a bad shape below
  }
  if (entry === null || isRunStale(entry)) {
    clearRunEntry(documentId, blockId)
    return null
  }
  return entry
}

export function clearRunEntry(documentId: number, blockId: number): void {
  try {
    localStorage.removeItem(storageKey(documentId, blockId))
  } catch {
    // localStorage unavailable — nothing to clear
  }
}

// RunRecorder owns one run's entry for the duration of the run: it writes the
// entry immediately on construction (so the run is remembered before any text
// arrives), batches text appends, and removes the entry on clear().
export class RunRecorder {
  private entry: RunEntry
  private lastWrite = 0
  private pending: ReturnType<typeof setTimeout> | null = null
  private stopped = false

  constructor(documentId: number, blockId: number, baseResponseCount: number) {
    this.entry = { blockId, documentId, startedAt: Date.now(), baseResponseCount, text: '' }
    this.write()
  }

  append(text: string) {
    if (this.stopped) return
    this.entry.text += text
    const elapsed = Date.now() - this.lastWrite
    if (elapsed >= WRITE_THROTTLE_MS) {
      this.write()
    } else if (this.pending === null) {
      // One trailing write picks up everything appended during the window.
      this.pending = setTimeout(() => {
        this.pending = null
        if (!this.stopped) this.write()
      }, WRITE_THROTTLE_MS - elapsed)
    }
  }

  // clear removes the entry once the run has settled (saved or failed with the
  // user watching). Idempotent; later append() calls become no-ops.
  clear() {
    this.stopped = true
    if (this.pending !== null) {
      clearTimeout(this.pending)
      this.pending = null
    }
    clearRunEntry(this.entry.documentId, this.entry.blockId)
  }

  private write() {
    this.lastWrite = Date.now()
    try {
      localStorage.setItem(
        storageKey(this.entry.documentId, this.entry.blockId),
        JSON.stringify(this.entry),
      )
    } catch {
      // quota/private mode — the run still works, it just won't survive reload
    }
  }
}
