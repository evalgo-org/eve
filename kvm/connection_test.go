package kvm

import (
	"net"
	"os"
	"testing"
	"time"
)

func TestUnixDialer(t *testing.T) {
	t.Run("UnixDialer struct creation", func(t *testing.T) {
		dialer := &UnixDialer{
			Path: "/var/run/libvirt/libvirt-sock",
		}

		if dialer.Path != "/var/run/libvirt/libvirt-sock" {
			t.Errorf("Expected Path to be set correctly")
		}
	})

	t.Run("UnixDialer with empty path", func(t *testing.T) {
		dialer := &UnixDialer{
			Path: "",
		}

		_, err := dialer.Dial()
		if err == nil {
			t.Errorf("Expected error when dialing with empty path")
		}
	})

	t.Run("UnixDialer with non-existent socket", func(t *testing.T) {
		dialer := &UnixDialer{
			Path: "/tmp/nonexistent-socket-that-does-not-exist.sock",
		}

		_, err := dialer.Dial()
		if err == nil {
			t.Errorf("Expected error when dialing non-existent socket")
		}
	})

	t.Run("UnixDialer with test socket", func(t *testing.T) {
		// Create a temporary Unix socket for testing
		tmpDir := t.TempDir()
		socketPath := tmpDir + "/test.sock"

		// Create a listener
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to create test socket: %v", err)
		}
		defer listener.Close()

		// Accept connections in background
		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}()

		// Give listener time to start
		time.Sleep(10 * time.Millisecond)

		// Test dialing
		dialer := &UnixDialer{
			Path: socketPath,
		}

		conn, err := dialer.Dial()
		if err != nil {
			t.Fatalf("Failed to dial test socket: %v", err)
		}
		if conn == nil {
			t.Errorf("Expected non-nil connection")
		}
		conn.Close()
	})
}

func TestConnect(t *testing.T) {
	// Check if libvirt socket exists
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping Connect test: libvirt socket not found")
	}

	t.Run("Connect to libvirt", func(t *testing.T) {
		vir, err := Connect(socketPath)
		if err != nil {
			t.Skipf("Could not connect to libvirt (may not be running): %v", err)
		}
		if vir == nil {
			t.Errorf("Expected non-nil libvirt connection")
		}

		// Clean up
		if vir != nil {
			_ = Disconnect(vir)
		}
	})

	t.Run("Connect with invalid socket path", func(t *testing.T) {
		vir, err := Connect("/tmp/invalid-libvirt-socket.sock")
		if err == nil {
			t.Errorf("Expected error when connecting to invalid socket")
			if vir != nil {
				_ = Disconnect(vir)
			}
		}
	})

	t.Run("Connect with empty path", func(t *testing.T) {
		vir, err := Connect("")
		if err == nil {
			t.Errorf("Expected error when connecting with empty path")
			if vir != nil {
				_ = Disconnect(vir)
			}
		}
	})
}

func TestDisconnect(t *testing.T) {
	t.Run("Disconnect nil connection", func(t *testing.T) {
		err := Disconnect(nil)
		if err != nil {
			t.Errorf("Expected no error when disconnecting nil connection, got: %v", err)
		}
	})

	// Test with real connection if available
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); err == nil {
		t.Run("Disconnect real connection", func(t *testing.T) {
			vir, err := Connect(socketPath)
			if err != nil {
				t.Skipf("Could not connect to libvirt: %v", err)
			}

			err = Disconnect(vir)
			if err != nil {
				t.Errorf("Disconnect failed: %v", err)
			}
		})

		t.Run("Multiple disconnects", func(t *testing.T) {
			vir, err := Connect(socketPath)
			if err != nil {
				t.Skipf("Could not connect to libvirt: %v", err)
			}

			// First disconnect
			err = Disconnect(vir)
			if err != nil {
				t.Errorf("First disconnect failed: %v", err)
			}

			// Second disconnect (should handle gracefully)
			// Note: This may or may not error depending on libvirt implementation
			_ = Disconnect(vir)
		})
	}
}

func TestConnectionIntegration(t *testing.T) {
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Skipping integration test: libvirt socket not found")
	}

	t.Run("Connect and disconnect workflow", func(t *testing.T) {
		// Connect
		vir, err := Connect(socketPath)
		if err != nil {
			t.Skipf("Could not connect to libvirt: %v", err)
		}

		if vir == nil {
			t.Fatal("Expected non-nil connection")
		}

		// Verify connection is usable (try to get version)
		// Note: We can't easily test this without importing more libvirt methods
		// But the connection object should be valid

		// Disconnect
		err = Disconnect(vir)
		if err != nil {
			t.Errorf("Failed to disconnect: %v", err)
		}
	})

	t.Run("Concurrent connections", func(t *testing.T) {
		// Test that we can create multiple connections
		conns := make([]*net.Conn, 3)
		dialers := make([]*UnixDialer, 3)

		for i := 0; i < 3; i++ {
			dialers[i] = &UnixDialer{Path: socketPath}
			conn, err := dialers[i].Dial()
			if err != nil {
				t.Skipf("Could not create connection %d: %v", i, err)
			}
			conns[i] = &conn
		}

		// Close all connections
		for i := 0; i < 3; i++ {
			if conns[i] != nil {
				(*conns[i]).Close()
			}
		}
	})
}

func BenchmarkUnixDialerDial(b *testing.B) {
	// Create a test socket
	tmpDir := b.TempDir()
	socketPath := tmpDir + "/bench.sock"

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		b.Fatalf("Failed to create test socket: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	time.Sleep(10 * time.Millisecond)

	dialer := &UnixDialer{Path: socketPath}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := dialer.Dial()
		if err != nil {
			b.Fatalf("Dial failed: %v", err)
		}
		conn.Close()
	}
}

func BenchmarkConnect(b *testing.B) {
	socketPath := "/var/run/libvirt/libvirt-sock"
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		b.Skip("Skipping benchmark: libvirt socket not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vir, err := Connect(socketPath)
		if err != nil {
			b.Fatalf("Connect failed: %v", err)
		}
		_ = Disconnect(vir)
	}
}
