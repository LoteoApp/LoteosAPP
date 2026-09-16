const messagesByCode: Record<string, string> = {
  invalid_credentials: 'Correo o contraseña incorrectos.',
  email_not_confirmed: 'Falta confirmar el correo electrónico de la cuenta.',
  over_request_rate_limit:
    'Demasiados intentos seguidos. Esperá unos minutos y volvé a probar.',
}

function readErrorCode(error: unknown): string | undefined {
  if (typeof error !== 'object' || error === null) {
    return undefined
  }

  const { code } = error as { code?: unknown }
  return typeof code === 'string' ? code : undefined
}

export function describeAuthError(error: unknown): string {
  const code = readErrorCode(error)
  if (code && Object.hasOwn(messagesByCode, code)) {
    return messagesByCode[code]
  }

  if (error instanceof Error && error.message !== '') {
    return `No se pudo iniciar sesión: ${error.message}`
  }

  return 'No se pudo iniciar sesión. Probá de nuevo en unos minutos.'
}
