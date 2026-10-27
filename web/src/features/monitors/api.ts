import { apiFetch } from '../../app/api'
import type { Monitor, MonitorInput, MonitorListResult, MonitorSummary } from './types'

function organizationPath(organizationID: string): string {
  return `/api/v1/orgs/${encodeURIComponent(organizationID)}`
}

export const monitorsApi = {
  list(organizationID: string, page: number, pageSize: number): Promise<MonitorListResult> {
    return apiFetch<MonitorListResult>(
      `${organizationPath(organizationID)}/monitors?page=${page}&page_size=${pageSize}`,
    )
  },

  summary(organizationID: string): Promise<MonitorSummary> {
    return apiFetch<MonitorSummary>(`${organizationPath(organizationID)}/dashboard/summary`)
  },

  create(organizationID: string, input: MonitorInput): Promise<Monitor> {
    return apiFetch<Monitor>(`${organizationPath(organizationID)}/monitors`, {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },

  update(
    organizationID: string,
    monitorID: string,
    input: Partial<MonitorInput>,
  ): Promise<Monitor> {
    return apiFetch<Monitor>(
      `${organizationPath(organizationID)}/monitors/${encodeURIComponent(monitorID)}`,
      {
        method: 'PATCH',
        body: JSON.stringify(input),
      },
    )
  },

  pause(organizationID: string, monitorID: string): Promise<Monitor> {
    return apiFetch<Monitor>(
      `${organizationPath(organizationID)}/monitors/${encodeURIComponent(monitorID)}/pause`,
      { method: 'POST' },
    )
  },

  resume(organizationID: string, monitorID: string): Promise<Monitor> {
    return apiFetch<Monitor>(
      `${organizationPath(organizationID)}/monitors/${encodeURIComponent(monitorID)}/resume`,
      { method: 'POST' },
    )
  },

  delete(organizationID: string, monitorID: string): Promise<void> {
    return apiFetch<void>(
      `${organizationPath(organizationID)}/monitors/${encodeURIComponent(monitorID)}`,
      { method: 'DELETE' },
    )
  },
}
