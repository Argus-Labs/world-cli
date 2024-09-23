package cardinal

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"

	"pkg.world.dev/world-cli/common/editor"
)

const ceReadTimeout = 5 * time.Second

// startCardinalEditor runs the Cardinal Editor
func startCardinalEditor(ctx context.Context, rootDir string, gameDir string, port int) error {
	if err := editor.SetupCardinalEditor(rootDir, gameDir); err != nil {
		return err
	}

	// Create a new HTTP server
	fs := http.FileServer(http.Dir(filepath.Join(rootDir, editor.EditorDir)))
	http.Handle("/", fs)
	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		ReadTimeout: ceReadTimeout,
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := server.ListenAndServe(); err != nil && !eris.Is(err, http.ErrServerClosed) {
			return eris.Wrap(err, "Cardinal Editor server encountered an error")
		}
		return nil
	})
	group.Go(func() error {
		<-ctx.Done()
		if err := server.Shutdown(ctx); err != nil {
			return eris.Wrap(err, "Failed to gracefully shutdown server")
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}
