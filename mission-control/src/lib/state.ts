export function stateChip(state: string): string {
  switch (state) {
    case 'succeeded':
      return 'chip chip-ok'
    case 'running':
    case 'queued':
      return 'chip chip-live'
    case 'failed':
      return 'chip chip-fail'
    case 'cancelled':
    case 'awaiting_approval':
      return 'chip chip-warn'
    default:
      return 'chip'
  }
}
