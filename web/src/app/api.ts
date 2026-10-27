export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly details?: Record<string, string>,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

let unauthorizedHandler: (() => void) | undefined

export function setUnauthorizedHandler(handler: () => void): void {
  unauthorizedHandler = handler
}

export async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  if (options.body) {
    headers.set('Content-Type', 'application/json')
  }
  const response = await fetch(path, { ...options, headers, credentials: 'include' })
  if (response.status === 204) {
    return undefined as T
  }

  const payload: unknown = await response.json().catch(() => null)
  if (!response.ok) {
    if (response.status === 401) {
      unauthorizedHandler?.()
    }
    const errorPayload = payload as {
      error?: { code?: string; message?: string; details?: Record<string, string> }
    } | null
    throw new ApiError(
      response.status,
      errorPayload?.error?.code ?? 'request_failed',
      errorPayload?.error?.message ?? 'The request failed.',
      errorPayload?.error?.details,
    )
  }
  return payload as T
}
