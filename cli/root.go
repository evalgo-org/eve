package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"eve.evalgo.org/api"
	eve "eve.evalgo.org/common"
	"eve.evalgo.org/db"
	"eve.evalgo.org/queue"
	"eve.evalgo.org/security"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "eve",
	Short: "a sample service implementation for processing flow messages with RabbitMQ and CouchDB",
	Run:   runServer,
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.flow-service.yaml)")
	RootCmd.PersistentFlags().String("port", "", "Server port")
	RootCmd.PersistentFlags().String("rabbitmq-url", "", "RabbitMQ connection URL")
	RootCmd.PersistentFlags().String("queue-name", "", "RabbitMQ queue name")
	RootCmd.PersistentFlags().String("couchdb-url", "", "CouchDB connection URL")
	RootCmd.PersistentFlags().String("database-name", "", "CouchDB database name")
	RootCmd.PersistentFlags().String("jwt-secret", "", "JWT secret key")

	viper.BindPFlag("port", RootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("rabbitmq.url", RootCmd.PersistentFlags().Lookup("rabbitmq-url"))
	viper.BindPFlag("rabbitmq.queue_name", RootCmd.PersistentFlags().Lookup("queue-name"))
	viper.BindPFlag("couchdb.url", RootCmd.PersistentFlags().Lookup("couchdb-url"))
	viper.BindPFlag("couchdb.database_name", RootCmd.PersistentFlags().Lookup("database-name"))
	viper.BindPFlag("jwt.secret", RootCmd.PersistentFlags().Lookup("jwt-secret"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".flow-service")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func runServer(cmd *cobra.Command, args []string) {
	config := eve.FlowConfig{
		RabbitMQURL:  viper.GetString("rabbitmq.url"),
		QueueName:    viper.GetString("rabbitmq.queue_name"),
		CouchDBURL:   viper.GetString("couchdb.url"),
		DatabaseName: viper.GetString("couchdb.database_name"),
		ApiKey:       viper.GetString("jwt.secret"),
	}

	// Initialize services
	rabbitMQService, err := queue.NewRabbitMQService(config)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ service: %v", err)
	}
	defer rabbitMQService.Close()

	couchDBService, err := db.NewCouchDBService(config)
	if err != nil {
		log.Fatalf("Failed to initialize CouchDB service: %v", err)
	}
	defer couchDBService.Close()

	jwtService := security.NewJWTService(config.ApiKey)

	// Initialize Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize handlers
	handlers := &api.Handlers{
		RabbitMQ: rabbitMQService,
		CouchDB:  couchDBService,
		JWT:      jwtService,
	}

	// Routes
	api.SetupRoutes(e, handlers, &config)

	// Start server
	port := viper.GetString("port")
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
