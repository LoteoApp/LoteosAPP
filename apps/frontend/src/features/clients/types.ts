export type Cliente = {
  id: string
  nombre: string
  apellido: string
  dni: string
  celular: string
  email: string
}

export type ClienteFormValues = Omit<Cliente, 'id'>

export function toClienteFormValues(cliente: Cliente): ClienteFormValues {
  const { nombre, apellido, dni, celular, email } = cliente
  return { nombre, apellido, dni, celular, email }
}
