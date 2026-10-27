import type { Identity } from '../features/auth/types'

export const identityFixture: Identity = {
  user: {
    id: '11111111-1111-1111-1111-111111111111',
    name: 'Ada Lovelace',
    email: 'ada@example.com',
    created_at: '2026-01-01T00:00:00Z',
  },
  organizations: [
    {
      id: '22222222-2222-2222-2222-222222222222',
      name: "Ada's organization",
      role: 'owner',
    },
  ],
}
