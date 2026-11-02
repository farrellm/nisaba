// Web Push for run-finished notifications. The browser subscribes with the
// server's VAPID key and registers the subscription with the backend
// (/api/push/*); public/sw.js shows the notifications. On iOS this only works
// once Nisaba is added to the Home Screen and opened from there.

import { api, ApiError } from '../api/client'

export type PushState =
  | 'unsupported' // browser has no Push API
  | 'needs-install' // iOS Safari tab: must be added to the Home Screen first
  | 'server-disabled' // backend has no VAPID keys
  | 'denied' // user blocked notifications
  | 'off'
  | 'on'

const SW_URL = '/sw.js'

function isIOS(): boolean {
  return (
    /iPad|iPhone|iPod/.test(navigator.userAgent) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)
  )
}

function isStandalone(): boolean {
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    (navigator as Navigator & { standalone?: boolean }).standalone === true
  )
}

function hasPushApi(): boolean {
  return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window
}

async function currentSubscription(): Promise<PushSubscription | null> {
  const reg = await navigator.serviceWorker.getRegistration(SW_URL)
  return reg ? reg.pushManager.getSubscription() : null
}

// syncSubscription re-registers this browser's existing subscription with the
// server (an idempotent upsert). The server prunes endpoints a push service
// rejects, and keys can rotate, so the browser's copy is the source of truth.
async function syncSubscription(sub: PushSubscription): Promise<void> {
  await api.post('/api/push/subscriptions', sub.toJSON())
}

// registerServiceWorker installs sw.js once at startup so notification taps
// and pushes work even before the user opens the account menu, and re-syncs
// an existing subscription (a 401 before login is expected and ignored).
export function registerServiceWorker(): void {
  if (!('serviceWorker' in navigator)) return
  navigator.serviceWorker
    .register(SW_URL)
    .then(async (reg) => {
      if (!('PushManager' in window) || Notification.permission !== 'granted') return
      const sub = await reg.pushManager.getSubscription()
      if (sub) await syncSubscription(sub).catch(() => undefined)
    })
    .catch((err: unknown) => {
      console.warn('service worker registration failed', err)
    })
}

async function fetchPublicKey(): Promise<string | null> {
  try {
    const { publicKey } = await api.get<{ publicKey: string }>('/api/push/vapid-public-key')
    return publicKey
  } catch (err) {
    if (err instanceof ApiError && err.status === 503) return null
    throw err
  }
}

export async function getPushState(): Promise<PushState> {
  if (!hasPushApi()) return isIOS() && !isStandalone() ? 'needs-install' : 'unsupported'
  if ((await fetchPublicKey()) === null) return 'server-disabled'
  if (Notification.permission === 'denied') return 'denied'
  const sub = await currentSubscription()
  if (!sub) return 'off'
  await syncSubscription(sub)
  return 'on'
}

function urlBase64ToUint8Array(base64: string): Uint8Array<ArrayBuffer> {
  const padded = (base64 + '='.repeat((4 - (base64.length % 4)) % 4))
    .replace(/-/g, '+')
    .replace(/_/g, '/')
  const raw = atob(padded)
  const out = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i)
  return out
}

// enablePush must be called from a user gesture (iOS rejects the permission
// prompt otherwise). Resolves to the resulting state.
export async function enablePush(): Promise<PushState> {
  const permission = await Notification.requestPermission()
  if (permission !== 'granted') return permission === 'denied' ? 'denied' : 'off'

  const publicKey = await fetchPublicKey()
  if (publicKey === null) return 'server-disabled'

  const reg = await navigator.serviceWorker.register(SW_URL)
  await navigator.serviceWorker.ready
  let sub = await reg.pushManager.getSubscription()
  if (!sub) {
    sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(publicKey),
    })
  }
  await syncSubscription(sub)
  return 'on'
}

export async function disablePush(): Promise<PushState> {
  const sub = await currentSubscription()
  if (sub) {
    await api.del('/api/push/subscriptions', { endpoint: sub.endpoint })
    await sub.unsubscribe()
  }
  return 'off'
}
