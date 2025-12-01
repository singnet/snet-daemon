package handler

import (
	"net/http"

	"github.com/rs/cors"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/logger"
)

func Cors() *cors.Cors {
	return cors.New(cors.Options{
		OptionsPassthrough:  false,
		AllowCredentials:    false,
		AllowPrivateNetwork: true,
		AllowedOrigins:      []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodOptions,
			http.MethodHead,
			http.MethodConnect,
		},
		Debug:          "debug" == config.GetString(logger.LogLevelKey),
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"x-grpc-web", "grpc-encoding", "grpc-status", "grpc-message", "grpc-timeout", "grpc-accept-encoding", "grpc-status-details-bin", "Retry-After"},
	})
}
