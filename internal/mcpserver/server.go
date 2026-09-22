package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/bridge"
)

type emptyInput struct{}

type pathInput struct {
	Path string `json:"path" jsonschema:"absolute workstation path"`
}

type readBytesInput struct {
	Path     string `json:"path" jsonschema:"absolute workstation path"`
	Offset   int64  `json:"offset,omitempty" jsonschema:"byte offset,minimum=0"`
	MaxBytes int64  `json:"max_bytes,omitempty" jsonschema:"maximum bytes to return,minimum=1"`
}

type listDirectoryInput struct {
	Path       string `json:"path" jsonschema:"absolute workstation directory path"`
	MaxEntries int    `json:"max_entries,omitempty" jsonschema:"maximum entries to return,minimum=1"`
}

type writeTextInput struct {
	Path      string `json:"path" jsonschema:"absolute workstation path"`
	Content   string `json:"content" jsonschema:"UTF-8 content to write"`
	Overwrite bool   `json:"overwrite,omitempty" jsonschema:"allow replacing an existing file"`
}

type makeDirectoryInput struct {
	Path    string `json:"path" jsonschema:"absolute workstation directory path"`
	Parents bool   `json:"parents,omitempty" jsonschema:"create missing parent directories"`
}

type movePathInput struct {
	Source      string `json:"source" jsonschema:"absolute existing source path"`
	Destination string `json:"destination" jsonschema:"absolute destination path that must not already exist"`
}

type runProcessInput struct {
	Executable     string   `json:"executable" jsonschema:"absolute executable path; must be explicitly allowlisted"`
	Args           []string `json:"args,omitempty" jsonschema:"argument vector excluding executable"`
	Cwd            string   `json:"cwd" jsonschema:"absolute allowed working directory"`
	TimeoutSeconds float64  `json:"timeout_seconds,omitempty" jsonschema:"runtime timeout in seconds,minimum=0.01"`
}

func New(b *bridge.Bridge, version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "WorkBridgeMCP", Version: version}, nil)

	mcp.AddTool(s, readOnlyTool("workstation_info", "Report workstation bridge identity and active capability ceilings"),
		func(ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, map[string]any, error) {
			return nil, b.Info(), nil
		})

	if b.HasRead() {
		mcp.AddTool(s, readOnlyTool("read_text", "Read a bounded UTF-8 file from an allowed read root"),
			func(ctx context.Context, req *mcp.CallToolRequest, in pathInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.ReadText(in.Path)
				return nil, out, err
			})
		mcp.AddTool(s, readOnlyTool("read_bytes", "Read bounded binary bytes as base64 from an allowed read root"),
			func(ctx context.Context, req *mcp.CallToolRequest, in readBytesInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.ReadBytes(in.Path, in.Offset, in.MaxBytes)
				return nil, out, err
			})
		mcp.AddTool(s, readOnlyTool("stat_path", "Read metadata for a path under an allowed read root"),
			func(ctx context.Context, req *mcp.CallToolRequest, in pathInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.Stat(in.Path)
				return nil, out, err
			})
		mcp.AddTool(s, readOnlyTool("list_directory", "List a bounded number of entries under an allowed read root"),
			func(ctx context.Context, req *mcp.CallToolRequest, in listDirectoryInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.ListDirectory(in.Path, in.MaxEntries)
				return nil, out, err
			})
	}

	if b.HasWrite() {
		mcp.AddTool(s, mutationTool("write_text", "Write bounded UTF-8 content under an allowed write root", true),
			func(ctx context.Context, req *mcp.CallToolRequest, in writeTextInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.WriteText(in.Path, in.Content, in.Overwrite)
				return nil, out, err
			})
		mcp.AddTool(s, mutationTool("make_directory", "Create a directory under an allowed write root", true),
			func(ctx context.Context, req *mcp.CallToolRequest, in makeDirectoryInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.MakeDirectory(in.Path, in.Parents)
				return nil, out, err
			})
		mcp.AddTool(s, mutationTool("move_path", "Move a path between locations under allowed write roots without overwriting the destination", false),
			func(ctx context.Context, req *mcp.CallToolRequest, in movePathInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.MovePath(in.Source, in.Destination)
				return nil, out, err
			})
	}

	if b.HasProcess() {
		mcp.AddTool(s, mutationTool("run_process", "Run one bounded allowlisted executable in an allowed working directory", false),
			func(ctx context.Context, req *mcp.CallToolRequest, in runProcessInput) (*mcp.CallToolResult, map[string]any, error) {
				out, err := b.RunProcess(ctx, in.Executable, in.Args, in.Cwd, in.TimeoutSeconds)
				return nil, out, err
			})
	}
	return s
}

func readOnlyTool(name, description string) *mcp.Tool {
	openWorld := false
	destructive := false
	return &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: &destructive,
			IdempotentHint:  true,
			OpenWorldHint:   &openWorld,
		},
	}
}

func mutationTool(name, description string, idempotent bool) *mcp.Tool {
	openWorld := false
	destructive := true
	return &mcp.Tool{
		Name:        name,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &destructive,
			IdempotentHint:  idempotent,
			OpenWorldHint:   &openWorld,
		},
	}
}
