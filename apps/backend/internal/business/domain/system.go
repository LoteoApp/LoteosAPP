package domain

import "time"

type Info struct {
	Service   string       `json:"service"`
	Status    string       `json:"status"`
	CheckedAt time.Time    `json:"checked_at"`
	Database  DatabaseInfo `json:"database"`
	Pool      PoolInfo     `json:"pool"`
}

type DatabaseInfo struct {
	Connected     bool      `json:"connected"`
	Version       string    `json:"version"`
	DatabaseName  string    `json:"database_name"`
	User          string    `json:"user"`
	ServerAddress string    `json:"server_address"`
	ServerPort    int32     `json:"server_port"`
	DatabaseTime  time.Time `json:"database_time"`
}

type PoolInfo struct {
	MaxConnections      int32 `json:"max_connections"`
	TotalConnections    int32 `json:"total_connections"`
	AcquiredConnections int32 `json:"acquired_connections"`
	IdleConnections     int32 `json:"idle_connections"`
	NewConnections      int64 `json:"new_connections"`
	ClosedConnections   int64 `json:"closed_connections"`
}
