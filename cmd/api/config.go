package main

type Configuration struct {
	DBHost string
	DBUser string
	DBPass string
	DBName string
	DBSSL  string
	// Allowed origins for CORS
	AllowedOrigins string
	Port           string
}
