export type Surveyor = {
  id: string
  nombre: string
  apellido: string
  email: string
  fechaBaja: string | null
}

export type SurveyorFormValues = {
  nombre: string
  apellido: string
  email: string
}

export function isActive(surveyor: Surveyor): boolean {
  return surveyor.fechaBaja === null
}

export function toSurveyorFormValues(surveyor: Surveyor): SurveyorFormValues {
  const { nombre, apellido, email } = surveyor
  return { nombre, apellido, email }
}

export function fullName(surveyor: Surveyor): string {
  return `${surveyor.nombre} ${surveyor.apellido}`.trim()
}
