package config

type Configuration struct {
	// Postgres URI
	PostgresURI string
	// Tendermint websocket URI
	TendermintWsURI string
	// Derivation path: "tiov" or "iov"
	Hrp string
	// Allowed origins for CORS
	AllowedOrigins string
	Port           string
}
