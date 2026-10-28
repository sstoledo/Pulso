export interface PublicMonitor {
  name: string
  status: string
  availability_24h: number | null
}

export interface PublicIncident {
  monitor_name: string
  opened_at: string
  resolved_at: string | null
  duration_ms: number | null
  status: 'OPEN' | 'RESOLVED'
}

export interface PublicStatusPage {
  title: string
  monitors: PublicMonitor[]
  incidents: PublicIncident[]
}
