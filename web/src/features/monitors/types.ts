export interface Monitor {
  id: string
  name: string
  url: string
  interval_seconds: number
  timeout_ms: number
  expected_status_min: number
  expected_status_max: number
  enabled: boolean
  show_public: boolean
  status: 'UNKNOWN' | 'UP' | 'DOWN'
  last_check_at: string | null
  last_latency_ms: number | null
  created_at: string
  updated_at: string
}

export interface MonitorInput {
  name: string
  url: string
  interval_seconds: number
  timeout_ms: number
  expected_status_min: number
  expected_status_max: number
  show_public: boolean
}

export interface MonitorListResult {
  items: Monitor[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface MonitorSummary {
  up: number
  down: number
  unknown: number
}
