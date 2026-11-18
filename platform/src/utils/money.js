// All marketplace prices are displayed in Uzbek som (sum).
export function money(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '—'
  return `${new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 }).format(number)} sum`
}

export function moneyShort(value) {
  return money(value)
}
