// pkg/output/renderer.go
package output

import (
	"context"
	"io"

	"github.com/damirmur/swisseph_build/pkg/astro"
)

type Renderer interface {
	Render(ctx context.Context, result *astro.AstroResult, w io.Writer) error
}
