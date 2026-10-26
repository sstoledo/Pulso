export interface User {
  id: string
  name: string
  email: string
  created_at: string
}

export interface Organization {
  id: string
  name: string
  role: 'owner' | 'admin' | 'viewer'
}

export interface Identity {
  user: User
  organizations: Organization[]
}

export interface RegisterInput {
  name: string
  email: string
  password: string
}

export interface LoginInput {
  email: string
  password: string
}
