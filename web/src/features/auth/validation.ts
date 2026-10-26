export function validEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())
}

export function safeReturnPath(value: unknown): string {
  return typeof value === 'string' && value.startsWith('/') && !value.startsWith('//')
    ? value
    : '/monitors'
}
