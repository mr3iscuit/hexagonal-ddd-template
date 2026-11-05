package main

import "time"

const (
	addressEnvironmentVariable = "ADDR"
	defaultAddress             = ":8080"

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second

	listenNetwork = "tcp"

	logListening     = "listening"
	logServerStopped = "server stopped"
	logKeyAddress    = "address"
	logKeyError      = "error"
)
