import { apiFetch } from '../../app/api'
import type { IncidentList } from './types'

export const incidentsApi = {
  list(
    organizationID: string,
    monitorID: string,
    page: number,
    pageSize: number,
  ): Promise<IncidentList> {
    return apiFetch<IncidentList>(
      `/api/v1/orgs/${encodeURIComponent(organizationID)}/incidents?monitor_id=${encodeURIComponent(monitorID)}&page=${page}&page_size=${pageSize}`,
    )
  },
}
