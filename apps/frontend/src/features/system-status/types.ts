export type DatabaseInfo = {
  connected: boolean
  version: string
  database_name: string
  user: string
  server_address: string
  server_port: number
  database_time: string
}

export type PoolInfo = {
  max_connections: number
  total_connections: number
  acquired_connections: number
  idle_connections: number
  new_connections: number
  closed_connections: number
}

export type SystemInfo = {
  service: string
  status: string
  checked_at: string
  database: DatabaseInfo
  pool: PoolInfo
}
