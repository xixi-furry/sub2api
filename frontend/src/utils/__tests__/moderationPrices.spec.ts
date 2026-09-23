import { describe, expect, it } from 'vitest'
import { convertAuditPrice } from '../moderationPrices'

describe('catalog price currency conversion', () => {
  it('converts exactly and conservatively rounds tiny fractions', () => {
    expect(convertAuditPrice('0.15', '7.2')).toBe('1.08')
    expect(convertAuditPrice('0.003', '7.2')).toBe('0.0216')
    expect(convertAuditPrice('0.0000000001', '0.1')).toBe('0.0000000001')
    expect(convertAuditPrice('0', '7.2')).toBe('0')
    expect(convertAuditPrice('0.1234567891', '1')).toBe('0.1234567891')
  })
  it('does not assume a currency rate or turn unknown prices into zero', () => {
    for (const rate of ['', '0', '-1', 'NaN', 'Infinity', '1e3', '1000001']) expect(() => convertAuditPrice('1', rate)).toThrow()
    expect(() => convertAuditPrice('', '7')).toThrow()
    expect(() => convertAuditPrice('1000000000', '7')).toThrow()
  })
})
