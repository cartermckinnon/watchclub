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
	"github.com/cartermckinnon/watchclub/internal/mail"
	"github.com/cartermckinnon/watchclub/internal/service"
	"github.com/cartermckinnon/watchclub/internal/storage"
)

func NewServerCommand() cli.Command {
	sc := serverCommand{
		c:       flaggy.NewSubcommand("server"),
		address: ":8080",
		storage: "memory",
		baseURL: "http://localhost:3000/",
	}
	sc.c.String(&sc.address, "a", "address", "Address to bind the server to")
	sc.c.String(&sc.storage, "s", "storage", "Storage URI (memory, sqlite://path/to/db)")
	sc.c.String(&sc.baseURL, "u", "base-url", "Base URL for generating login links")
	sc.c.String(&sc.resendAPIKey, "", "resend-api-key", "Resend API key for sending emails (optional)")
	sc.c.String(&sc.resendFrom, "", "resend-from", "Email address to send from (required if using Resend)")
	sc.c.String(&sc.resendFromName, "", "resend-from-name", "Display name for from address (optional)")
	sc.c.Bool(&sc.devMode, "d", "dev", "Development mode (logs emails to console instead of sending)")
	return &sc
}

type serverCommand struct {
	c *flaggy.Subcommand

	address        string
	storage        string
	baseURL        string
	resendAPIKey   string
	resendFrom     string
	resendFromName string
	devMode        bool
}

func (sc *serverCommand) Flaggy() *flaggy.Subcommand {
	return sc.c
}

func (sc *serverCommand) Run(logger *zap.Logger, opts *cli.GlobalOptions) error {
	logger.Info("starting server",
		zap.String("address", sc.address),
		zap.String("storage", sc.storage))

	// Create storage layer
	store, err := storage.NewStorage(sc.storage)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Create email sender
	emailSender := mail.New(mail.Config{
		DevelopmentMode: sc.devMode,
		BaseURL:         sc.baseURL,
		ResendAPIKey:    sc.resendAPIKey,
		ResendFrom:      sc.resendFrom,
		ResendFromName:  sc.resendFromName,
		Logger:          logger,
	})

	// Create service
	svc := service.New(store, emailSender, sc.baseURL)

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
