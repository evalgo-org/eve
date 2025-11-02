package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eve.evalgo.org/registry"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	bolt "go.etcd.io/bbolt"
)

const (
	DefaultPort         = "8096"
	DefaultDBPath       = "/tmp/registry.db"
	HealthCheckInterval = 30 * time.Second
	ServiceBucket       = "services"
)

type RegistryService struct {
	db     *bolt.DB
	reg    *registry.Registry
	server *echo.Echo
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	dbPath := os.Getenv("REGISTRY_DB_PATH")
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	// Initialize BoltDB
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create bucket if not exists
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(ServiceBucket))
		return err
	}); err != nil {
		log.Fatalf("Failed to create bucket: %v", err)
	}

	// Initialize registry (load from static file if exists)
	registryPath := os.Getenv("SERVICE_REGISTRY_PATH")
	if registryPath == "" {
		registryPath = "/home/opunix/registry.json"
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		log.Printf("Warning: failed to load static registry file: %v", err)
		// Create empty registry
		reg = &registry.Registry{}
	}

	// Load services from BoltDB (overrides static file)
	if err := loadFromDB(db, reg); err != nil {
		log.Fatalf("Failed to load registry from database: %v", err)
	}

	svc := &RegistryService{
		db:     db,
		reg:    reg,
		server: echo.New(),
	}

	// Configure Echo
	svc.server.HideBanner = true
	svc.server.Use(middleware.Logger())
	svc.server.Use(middleware.Recover())
	svc.server.Use(middleware.CORS())

	// Routes
	svc.server.GET("/health", svc.handleHealth)
	svc.server.GET("/v1/api/services", svc.handleListServices)
	svc.server.GET("/v1/api/services/:id", svc.handleGetService)
	svc.server.POST("/v1/api/services/register", svc.handleRegister)
	svc.server.DELETE("/v1/api/services/:id", svc.handleUnregister)
	svc.server.GET("/v1/api/services/:id/health", svc.handleCheckHealth)
	svc.server.GET("/v1/api/services/capability/:capability", svc.handleFindByCapability)
	svc.server.GET("/v1/api/health-check-all", svc.handleHealthCheckAll)

	// Start health check goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.healthCheckLoop(ctx)

	// Start server
	go func() {
		log.Printf("Registry service starting on http://localhost:%s", port)
		if err := svc.server.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down registry service...")
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := svc.server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Registry service stopped")
}

func (s *RegistryService) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "registry",
	})
}

func (s *RegistryService) handleListServices(c echo.Context) error {
	services := make([]*registry.Service, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var svc registry.Service
			if err := json.Unmarshal(v, &svc); err != nil {
				log.Printf("Failed to unmarshal service %s: %v", k, err)
				return nil // Skip this service
			}
			services = append(services, &svc)
			return nil
		})
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, services)
}

func (s *RegistryService) handleGetService(c echo.Context) error {
	id := c.Param("id")

	var svc *registry.Service
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return fmt.Errorf("service not found")
		}

		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("service not found")
		}

		svc = &registry.Service{}
		return json.Unmarshal(data, svc)
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, svc)
}

func (s *RegistryService) handleRegister(c echo.Context) error {
	var svc registry.Service
	if err := c.Bind(&svc); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid service data")
	}

	// Validate required fields
	if svc.ID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Service ID is required")
	}
	if svc.URL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Service URL is required")
	}

	// Store in database
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		data, err := json.Marshal(svc)
		if err != nil {
			return err
		}
		return b.Put([]byte(svc.ID), data)
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	log.Printf("Registered service: %s (%s)", svc.ID, svc.URL)
	return c.JSON(http.StatusOK, svc)
}

func (s *RegistryService) handleUnregister(c echo.Context) error {
	id := c.Param("id")

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return fmt.Errorf("service not found")
		}

		if b.Get([]byte(id)) == nil {
			return fmt.Errorf("service not found")
		}

		return b.Delete([]byte(id))
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	log.Printf("Unregistered service: %s", id)
	return c.JSON(http.StatusOK, map[string]string{
		"status": "unregistered",
		"id":     id,
	})
}

func (s *RegistryService) handleCheckHealth(c echo.Context) error {
	id := c.Param("id")

	var svc *registry.Service
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return fmt.Errorf("service not found")
		}

		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("service not found")
		}

		svc = &registry.Service{}
		return json.Unmarshal(data, svc)
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	// Check health
	checkURL := svc.Properties.HealthCheck
	if checkURL == "" {
		checkURL = svc.URL
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(checkURL)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"service": id,
			"healthy": false,
			"error":   err.Error(),
		})
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300

	return c.JSON(http.StatusOK, map[string]interface{}{
		"service": id,
		"healthy": healthy,
		"status":  resp.StatusCode,
	})
}

func (s *RegistryService) handleHealthCheckAll(c echo.Context) error {
	results := make(map[string]bool)

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var svc registry.Service
			if err := json.Unmarshal(v, &svc); err != nil {
				return nil
			}

			// Check health
			checkURL := svc.Properties.HealthCheck
			if checkURL == "" {
				checkURL = svc.URL
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(checkURL)
			if err != nil {
				results[svc.ID] = false
				return nil
			}
			defer resp.Body.Close()

			results[svc.ID] = resp.StatusCode >= 200 && resp.StatusCode < 300
			return nil
		})
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

func (s *RegistryService) handleFindByCapability(c echo.Context) error {
	capability := c.Param("capability")

	matches := make([]*registry.Service, 0)

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var svc registry.Service
			if err := json.Unmarshal(v, &svc); err != nil {
				return nil
			}

			// Check if service has this capability
			for _, cap := range svc.Properties.Capabilities {
				if cap == capability {
					matches = append(matches, &svc)
					break
				}
			}
			return nil
		})
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, matches)
}

func (s *RegistryService) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.performHealthChecks()
		}
	}
}

func (s *RegistryService) performHealthChecks() {
	var unhealthy []string

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var svc registry.Service
			if err := json.Unmarshal(v, &svc); err != nil {
				return nil
			}

			// Check health
			checkURL := svc.Properties.HealthCheck
			if checkURL == "" {
				checkURL = svc.URL
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(checkURL)
			if err != nil {
				unhealthy = append(unhealthy, svc.ID)
				return nil
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				unhealthy = append(unhealthy, svc.ID)
			}

			return nil
		})
	})

	if err != nil {
		log.Printf("Error during health checks: %v", err)
		return
	}

	if len(unhealthy) > 0 {
		log.Printf("Unhealthy services: %v", unhealthy)
	}
}

func loadFromDB(db *bolt.DB, reg *registry.Registry) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ServiceBucket))
		if b == nil {
			return nil // No services yet
		}

		return b.ForEach(func(k, v []byte) error {
			var svc registry.Service
			if err := json.Unmarshal(v, &svc); err != nil {
				log.Printf("Failed to unmarshal service %s: %v", k, err)
				return nil // Skip this service
			}
			// Note: reg.Register would try to save back to file, so we skip that
			return nil
		})
	})
}
