export interface PasswordStrength {
  score: number // 0-4
  level: 'weak' | 'fair' | 'good' | 'strong' | 'very_strong'
  feedback: string[]
  isValid: boolean
}

export function validatePasswordStrength(password: string): PasswordStrength {
  const feedback: string[] = []
  let score = 0

  if (password.length === 0) {
    return {
      score: 0,
      level: 'weak',
      feedback: ['Password is required'],
      isValid: false,
    }
  }

  // Length check
  if (password.length < 8) {
    feedback.push('Password should be at least 8 characters long')
  } else {
    score++
  }

  if (password.length >= 12) {
    score++
  }

  // Uppercase check
  if (!/[A-Z]/.test(password)) {
    feedback.push('Add uppercase letters (A-Z)')
  } else {
    score++
  }

  // Lowercase check
  if (!/[a-z]/.test(password)) {
    feedback.push('Add lowercase letters (a-z)')
  } else {
    score++
  }

  // Number check
  if (!/[0-9]/.test(password)) {
    feedback.push('Add numbers (0-9)')
  } else {
    score++
  }

  // Special character check
  if (!/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)) {
    feedback.push('Add special characters (!@#$%^&* etc)')
  } else {
    score++
  }

  // Normalize score to 0-4
  score = Math.min(Math.floor(score / 1.5), 4)

  // Check for common patterns
  if (/(.)\1{2,}/.test(password)) {
    feedback.push('Avoid repeating characters')
    score = Math.max(0, score - 1)
  }

  if (/^[a-z]+$/i.test(password) || /^\d+$/.test(password)) {
    feedback.push('Use a mix of different character types')
    score = Math.max(0, score - 1)
  }

  const levels: Array<'weak' | 'fair' | 'good' | 'strong' | 'very_strong'> = [
    'weak',
    'fair',
    'good',
    'strong',
    'very_strong',
  ]

  const level = levels[score]
  const isValid = score >= 2 && password.length >= 8 // At least "good" strength and 8 chars minimum

  return {
    score,
    level,
    feedback,
    isValid,
  }
}

export function getPasswordStrengthColor(level: string): string {
  switch (level) {
    case 'weak':
      return '#ef4444'
    case 'fair':
      return '#f97316'
    case 'good':
      return '#eab308'
    case 'strong':
      return '#84cc16'
    case 'very_strong':
      return '#22c55e'
    default:
      return '#gray'
  }
}
