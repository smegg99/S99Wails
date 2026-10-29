// frontendgen/tests/theme-contrast.spec.ts
import { describe, expect, it } from 'vitest'
import { foregroundOn, parseHex } from '../assets/contrast'

describe('parseHex', () => {
  it('reads six and eight digit colors and ignores only a valid alpha', () => {
    for (const value of ['#123456', '123456', ' #123456ab ']) {
      expect(parseHex(value)).toEqual({ r: 18, g: 52, b: 86 })
    }
  })

  it('expands three and four digit colors', () => {
    for (const value of ['#abc', '#abcd', 'ABC']) {
      expect(parseHex(value)).toEqual({ r: 170, g: 187, b: 204 })
    }
  })

  it('rejects anything else', () => {
    for (const value of ['#12345', '#1234567', '#123456789', '#123xyz', 'red', '', '#']) {
      expect(parseHex(value), value).toBeNull()
    }
  })
})

describe('foregroundOn', () => {
  it('picks the readable ink', () => {
    expect(foregroundOn('#00ff00')).toBe('#000000')
    expect(foregroundOn('#000000')).toBe('#ffffff')
    expect(foregroundOn('#ffffff')).toBe('#000000')
  })
})
