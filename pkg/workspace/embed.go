package workspace

import "embed"

// EmbeddedTemplates holds the built-in workspace template files so they are
// available in a single-binary deployment without requiring a separate
// workspace_templates directory on disk.
//
//go:embed templates
var EmbeddedTemplates embed.FS
