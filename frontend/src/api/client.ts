// Thin fetch wrapper for the JSON API. Always sends the session cookie and
// surfaces server error messages as a typed ApiError so forms can display them.

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    let message = res.statusText
    try {
      const data = await res.json()
      if (data && typeof data.error === 'string') message = data.error
    } catch {
      // non-JSON error body — fall back to status text
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

// DeltaKind mirrors the backend's llm.DeltaKind: ordinary model text vs. a
// completed tool-call block (used to refresh a non-streamed preview at tool
// boundaries).
export type DeltaKind = 'text' | 'tool'

// postStream POSTs a JSON body and consumes a newline-delimited JSON (NDJSON)
// response stream. Each line is one event: {type:"delta",kind?,text} streams
// text to onDelta (kind defaults to "text"), {type:"ping"} is an ignored
// keepalive, {type:"error",message} throws, and {type:"done",<key>} resolves
// the promise with that payload (the server sends the final value under
// `doneKey`, e.g. "block"). Mirrors RunBlockStream on the backend.
async function postStream<T>(
  path: string,
  body: unknown,
  onDelta: (text: string, kind: DeltaKind) => void,
  doneKey: string,
): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (!res.ok || !res.body) {
    let message = res.statusText
    try {
      const data = await res.json()
      if (data && typeof data.error === 'string') message = data.error
    } catch {
      // non-JSON error body — fall back to status text
    }
    throw new ApiError(res.status, message)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let result: T | undefined

  const handle = (line: string) => {
    const trimmed = line.trim()
    if (!trimmed) return
    const event = JSON.parse(trimmed)
    if (event.type === 'delta') onDelta(event.text as string, (event.kind as DeltaKind) ?? 'text')
    else if (event.type === 'ping')
      return // keepalive; nothing to do
    else if (event.type === 'error') throw new ApiError(502, event.message ?? 'Stream error')
    else if (event.type === 'done') result = event[doneKey] as T
  }

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() ?? ''
    for (const line of lines) handle(line)
  }
  if (buffer) handle(buffer)

  if (result === undefined) throw new ApiError(502, 'Stream ended without a result')
  return result
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  del: <T>(path: string) => request<T>('DELETE', path),
  postStream: <T>(
    path: string,
    body: unknown,
    onDelta: (text: string, kind: DeltaKind) => void,
    doneKey: string,
  ) => postStream<T>(path, body, onDelta, doneKey),
}
