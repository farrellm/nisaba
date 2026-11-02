// Service worker for Web Push run notifications (see src/lib/push.ts and the
// backend's internal/push). Payload: {title, body, url}.

function readPayload(event) {
  if (event.data === null) return {}
  try {
    const data = event.data.json()
    return typeof data === 'object' && data !== null ? data : {}
  } catch {
    return { body: event.data.text() }
  }
}

self.addEventListener('push', (event) => {
  const data = readPayload(event)
  event.waitUntil(
    self.registration.showNotification(String(data.title ?? 'Nisaba'), {
      body: String(data.body ?? ''),
      icon: '/icon-192.png',
      badge: '/icon-192.png',
      data: { url: String(data.url ?? '/') },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = new URL(String(event.notification.data?.url ?? '/'), self.location.origin).href
  event.waitUntil(
    (async () => {
      const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
      const exact = windows.find((client) => client.url === url)
      if (exact !== undefined) return exact.focus()
      if (windows.length > 0) {
        const client = await windows[0].focus()
        return client.navigate(url)
      }
      return self.clients.openWindow(url)
    })(),
  )
})
