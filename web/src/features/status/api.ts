import { apiFetch } from '../../app/api'
import type { PublicStatusPage } from './types'

export const statusApi = {
  get(slug: string): Promise<PublicStatusPage> {
    return apiFetch<PublicStatusPage>(`/api/v1/public/status/${encodeURIComponent(slug)}`)
  },
}
