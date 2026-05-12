package daemon

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/bcp-technology-ug/lobster/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bbtea "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
)

// SSHServerConfig holds configuration for the Wish SSH server.
type SSHServerConfig struct {
	Addr               string
	HostKeyPath        string
	AuthorizedKeysPath string
	StaticToken        string
	WorkspaceID        string
}

// StartSSHServer starts the Wish SSH server and blocks until ctx is cancelled.
func StartSSHServer(ctx context.Context, cfg SSHServerConfig, conn *grpc.ClientConn) error {
	hostKeyPEM, err := ensureHostKey(cfg.HostKeyPath)
	if err != nil {
		return fmt.Errorf("ssh host key: %w", err)
	}

	opts := []ssh.Option{
		wish.WithAddress(cfg.Addr),
		wish.WithHostKeyPEM(hostKeyPEM),
		wish.WithMiddleware(
			bbtea.Middleware(func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
				m := ui.NewLobbyModel(conn, cfg.WorkspaceID)
				return m, []tea.ProgramOption{tea.WithAltScreen()}
			}),
			logging.Middleware(),
		),
	}

	switch {
	case cfg.AuthorizedKeysPath != "":
		opts = append(opts, wish.WithAuthorizedKeys(cfg.AuthorizedKeysPath))
	case cfg.StaticToken != "":
		token := cfg.StaticToken
		opts = append(opts, wish.WithPasswordAuth(func(_ ssh.Context, password string) bool {
			return password == token
		}))
	default:
		opts = append(opts, wish.WithPublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool {
			return true
		}))
	}

	srv, err := wish.NewServer(opts...)
	if err != nil {
		return fmt.Errorf("create ssh server: %w", err)
	}

	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("ssh listen %s: %w", cfg.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

// ensureHostKey returns a PEM-encoded ED25519 host key at path,
// generating and saving a new one if the file does not exist.
func ensureHostKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read host key %s: %w", path, err)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	pemBlock, err := gossh.MarshalPrivateKey(priv, "lobsterd host key")
	if err != nil {
		return nil, fmt.Errorf("marshal host key: %w", err)
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create host key dir: %w", err)
	}
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write host key %s: %w", path, err)
	}

	return pemBytes, nil
}
