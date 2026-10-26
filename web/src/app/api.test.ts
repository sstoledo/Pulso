import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError, apiFetch, setUnauthorizedHandler } from './api'

describe('API client', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    setUnauthorizedHandler(() => undefined)
  })

  it('uses credentials and publishes unauthorized responses', async () => {
    const unauthorized = vi.fn()
    setUnauthorizedHandler(unauthorized)
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ error: { code: 'unauthorized', message: 'Authentication required.' } }),
        {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    const request = apiFetch('/api/v1/auth/me')

    await expect(request).rejects.toEqual(
      new ApiError(401, 'unauthorized', 'Authentication required.'),
    )
    expect(unauthorized).toHaveBeenCalledOnce()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/auth/me',
      expect.objectContaining({ credentials: 'include' }),
    )
  })
})
