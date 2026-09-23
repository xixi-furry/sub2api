// Decimal prices use ten fractional digits, like the backend ledger. Round up
// at the last digit instead of losing a fraction to binary floating point.
const scale = 10n ** 10n
function fixed(value: string): bigint {
  if (!/^\d{1,10}(\.\d{1,10})?$/.test(value)) throw new Error('Invalid decimal')
  const [whole, fraction = ''] = value.split('.')
  return BigInt(whole!) * scale + BigInt(fraction.padEnd(10, '0'))
}
export function convertAuditPrice(usd: string, rate: string): string {
  const amount = fixed(usd), exchange = fixed(rate)
  if (exchange <= 0n || exchange > 1000000n * scale) throw new Error('Invalid exchange rate')
  const converted = (amount * exchange + scale - 1n) / scale
  if (converted > 1000000000n * scale) throw new Error('Price exceeds ledger limit')
  const fraction = (converted % scale).toString().padStart(10, '0').replace(/0+$/, '')
  return `${converted / scale}${fraction ? `.${fraction}` : ''}`
}
