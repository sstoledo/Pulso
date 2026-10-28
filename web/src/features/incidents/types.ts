export interface Incident {
  id: string
  monitor_id: string
  opened_at: string
  resolved_at: string | null
  duration_ms: number | null
  reason: string
  status: 'OPEN' | 'RESOLVED'
}

export interface IncidentList {
  items: Incident[]
  total: number
  page: number
  page_size: number
  total_pages: number
}
