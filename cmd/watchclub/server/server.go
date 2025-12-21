package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/integrii/flaggy"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	v1 "github.com/cartermckinnon/watchclub/internal/api/v1"
	"github.com/cartermckinnon/watchclub/internal/cli"
	"github.com/cartermckinnon/watchclub/internal/email"
	"github.com/cartermckinnon/watchclub/internal/service"
	"github.com/cartermckinnon/watchclub/internal/storage"
)

func NewServerCommand() cli.Command {
	sc := serverCommand{
		c:       flaggy.NewSubcommand("server"),
		address: ":8080",
	}
	sc.c.String(&sc.address, "a", "address", "Address to bind the server to")
	return &sc
}

type serverCommand struct {
	c *flaggy.Subcommand

	address string
}

func (sc *serverCommand) Flaggy() *flaggy.Subcommand {
	return sc.c
}

func (sc *serverCommand) Run(logger *zap.Logger, opts *cli.GlobalOptions) error {
	logger.Info("starting server", zap.String("address", sc.address))

	// Create storage layer
	store := storage.NewMemoryStorage()

	// Create email sender
	emailSender := email.New(email.Config{
		DevelopmentMode: true,
		BaseURL:         "http://localhost:3000/",
		Logger:          logger,
	})

	// Create service
	svc := service.New(store, emailSender, "http://localhost:3000/")

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	v1.RegisterWatchClubServiceServer(grpcServer, svc)

	// Register reflection service for debugging
	reflection.Register(grpcServer)

	// Wrap gRPC server with grpc-web wrapper
	wrappedGrpc := grpcweb.WrapServer(grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool {
			// Allow all origins in development
			// TODO: Make this configurable for production
			return true
		}),
	)

	// Create HTTP handler that can handle both gRPC and gRPC-Web
	httpServer := &http.Server{
		Addr: sc.address,
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			// Check if this is a gRPC-Web request
			if wrappedGrpc.IsGrpcWebRequest(req) || wrappedGrpc.IsAcceptableGrpcCorsRequest(req) {
				wrappedGrpc.ServeHTTP(resp, req)
				return
			}

			// Check if this is a standard gRPC request (HTTP/2 with content-type application/grpc)
			if req.ProtoMajor == 2 && strings.HasPrefix(req.Header.Get("Content-Type"), "application/grpc") {
				grpcServer.ServeHTTP(resp, req)
				return
			}

			// Otherwise return 404
			http.NotFound(resp, req)
		}),
	}

	// Add CORS middleware
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // TODO: Make this configurable for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	httpServer.Handler = corsHandler.Handler(httpServer.Handler)

	logger.Info("server listening (gRPC + gRPC-Web)", zap.String("address", sc.address))

	// Start serving
	if err := httpServer.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}
