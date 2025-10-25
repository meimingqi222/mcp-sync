package services

import (
	"mcp-sync/models"
	"runtime"
	"testing"
)

func TestWindowsService_WrapNpxCommand(t *testing.T) {
	ws := NewWindowsService()

	tests := []struct {
		name         string
		command      string
		args         []interface{}
		expectedCmd  string
		expectedArgs []interface{}
	}{
		{
			name:         "npx command without args",
			command:      "npx",
			args:         []interface{}{},
			expectedCmd:  "cmd",
			expectedArgs: []interface{}{"/c", "npx"},
		},
		{
			name:         "npx command with args",
			command:      "npx",
			args:         []interface{}{"@modelcontextprotocol/server-filesystem", "/path/to/files"},
			expectedCmd:  "cmd",
			expectedArgs: []interface{}{"/c", "npx", "@modelcontextprotocol/server-filesystem", "/path/to/files"},
		},
		{
			name:         "npx with combined command",
			command:      "npx @modelcontextprotocol/server-filesystem",
			args:         []interface{}{"/path/to/files"},
			expectedCmd:  "cmd",
			expectedArgs: []interface{}{"/c", "npx @modelcontextprotocol/server-filesystem"},
		},
		{
			name:         "non-npx command",
			command:      "python",
			args:         []interface{}{"server.py"},
			expectedCmd:  "python",
			expectedArgs: []interface{}{"server.py"},
		},
	}

	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := ws.WrapNpxCommand(tt.command, tt.args)

			if cmd != tt.expectedCmd {
				t.Errorf("WrapNpxCommand() command = %v, want %v", cmd, tt.expectedCmd)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("WrapNpxCommand() args length = %v, want %v", len(args), len(tt.expectedArgs))
				return
			}

			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("WrapNpxCommand() args[%d] = %v, want %v", i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}

func TestWindowsService_UnwrapNpxCommand(t *testing.T) {
	ws := NewWindowsService()

	tests := []struct {
		name         string
		command      string
		args         []interface{}
		expectedCmd  string
		expectedArgs []interface{}
	}{
		{
			name:         "cmd /c npx without args",
			command:      "cmd",
			args:         []interface{}{"/c", "npx"},
			expectedCmd:  "npx",
			expectedArgs: []interface{}{},
		},
		{
			name:         "cmd /c npx with args",
			command:      "cmd",
			args:         []interface{}{"/c", "npx", "@modelcontextprotocol/server-filesystem", "/path/to/files"},
			expectedCmd:  "npx",
			expectedArgs: []interface{}{"@modelcontextprotocol/server-filesystem", "/path/to/files"},
		},
		{
			name:         "cmd /c npx with combined command",
			command:      "cmd",
			args:         []interface{}{"/c", "npx @modelcontextprotocol/server-filesystem"},
			expectedCmd:  "npx @modelcontextprotocol/server-filesystem",
			expectedArgs: []interface{}{},
		},
		{
			name:         "non-cmd command",
			command:      "python",
			args:         []interface{}{"server.py"},
			expectedCmd:  "python",
			expectedArgs: []interface{}{"server.py"},
		},
		{
			name:         "cmd not /c",
			command:      "cmd",
			args:         []interface{}{"/k", "dir"},
			expectedCmd:  "cmd",
			expectedArgs: []interface{}{"/k", "dir"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := ws.UnwrapNpxCommand(tt.command, tt.args)

			if cmd != tt.expectedCmd {
				t.Errorf("UnwrapNpxCommand() command = %v, want %v", cmd, tt.expectedCmd)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("UnwrapNpxCommand() args length = %v, want %v", len(args), len(tt.expectedArgs))
				return
			}

			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("UnwrapNpxCommand() args[%d] = %v, want %v", i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}

func TestWindowsService_IsNpxCommand(t *testing.T) {
	ws := NewWindowsService()

	tests := []struct {
		name    string
		command string
		args    []interface{}
		want    bool
	}{
		{
			name:    "direct npx command",
			command: "npx",
			args:    []interface{}{"@modelcontextprotocol/server-filesystem"},
			want:    true,
		},
		{
			name:    "npx combined command",
			command: "npx @modelcontextprotocol/server-filesystem",
			args:    []interface{}{},
			want:    true,
		},
		{
			name:    "wrapped npx command",
			command: "cmd",
			args:    []interface{}{"/c", "npx", "@modelcontextprotocol/server-filesystem"},
			want:    true,
		},
		{
			name:    "wrapped npx combined command",
			command: "cmd",
			args:    []interface{}{"/c", "npx @modelcontextprotocol/server-filesystem"},
			want:    true,
		},
		{
			name:    "non-npx command",
			command: "python",
			args:    []interface{}{"server.py"},
			want:    false,
		},
		{
			name:    "cmd not npx",
			command: "cmd",
			args:    []interface{}{"/c", "dir"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ws.IsNpxCommand(tt.command, tt.args); got != tt.want {
				t.Errorf("IsNpxCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWindowsService_ApplyWindowsTransformation(t *testing.T) {
	ws := NewWindowsService()

	tests := []struct {
		name           string
		servers        []models.MCPServer
		wrap           bool
		expected       []models.MCPServer
		skipNonWindows bool
	}{
		{
			name: "wrap npx command on Windows",
			servers: []models.MCPServer{
				{
					ID:      "test-server",
					Name:    "test-server",
					Command: "npx",
					Args:    []string{"@modelcontextprotocol/server-filesystem", "/path"},
					Enabled: true,
				},
			},
			wrap: true,
			expected: []models.MCPServer{
				{
					ID:      "test-server",
					Name:    "test-server",
					Command: "cmd",
					Args:    []string{"/c", "npx", "@modelcontextprotocol/server-filesystem", "/path"},
					Enabled: true,
				},
			},
			skipNonWindows: true,
		},
		{
			name: "don't wrap non-npx command",
			servers: []models.MCPServer{
				{
					ID:      "test-server",
					Name:    "test-server",
					Command: "python",
					Args:    []string{"server.py"},
					Enabled: true,
				},
			},
			wrap: true,
			expected: []models.MCPServer{
				{
					ID:      "test-server",
					Name:    "test-server",
					Command: "python",
					Args:    []string{"server.py"},
					Enabled: true,
				},
			},
		},
		{
			name: "unwrap npx command",
			servers: []models.MCPServer{
				{
					ID:      "test-server",
					Name:    "test-server",
					Command: "cmd",
					Args:    []string{"/c", "npx", "@modelcontextprotocol/server-filesystem", "/path"},
					Enabled: true,
				},
			},
			wrap: false,
			expected: []models.MCPServer{
				{
					ID:      "test-server",
					Name:    "test-server",
					Command: "npx",
					Args:    []string{"@modelcontextprotocol/server-filesystem", "/path"},
					Enabled: true,
				},
			},
		},
		{
			name: "multiple servers mixed",
			servers: []models.MCPServer{
				{
					ID:      "npx-server",
					Name:    "npx-server",
					Command: "npx",
					Args:    []string{"@modelcontextprotocol/server-filesystem", "/path"},
					Enabled: true,
				},
				{
					ID:      "python-server",
					Name:    "python-server",
					Command: "python",
					Args:    []string{"server.py"},
					Enabled: true,
				},
			},
			wrap: true,
			expected: []models.MCPServer{
				{
					ID:      "npx-server",
					Name:    "npx-server",
					Command: "cmd",
					Args:    []string{"/c", "npx", "@modelcontextprotocol/server-filesystem", "/path"},
					Enabled: true,
				},
				{
					ID:      "python-server",
					Name:    "python-server",
					Command: "python",
					Args:    []string{"server.py"},
					Enabled: true,
				},
			},
			skipNonWindows: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipNonWindows && runtime.GOOS != "windows" {
				t.Skip("Skipping Windows-specific test on non-Windows platform")
			}

			got := ws.ApplyWindowsTransformation(tt.servers, tt.wrap)

			if len(got) != len(tt.expected) {
				t.Errorf("ApplyWindowsTransformation() length = %v, want %v", len(got), len(tt.expected))
				return
			}

			for i, server := range got {
				expected := tt.expected[i]
				if server.ID != expected.ID {
					t.Errorf("ApplyWindowsTransformation() server[%d].ID = %v, want %v", i, server.ID, expected.ID)
				}
				if server.Command != expected.Command {
					t.Errorf("ApplyWindowsTransformation() server[%d].Command = %v, want %v", i, server.Command, expected.Command)
				}
				if len(server.Args) != len(expected.Args) {
					t.Errorf("ApplyWindowsTransformation() server[%d].Args length = %v, want %v", i, len(server.Args), len(expected.Args))
					continue
				}
				for j, arg := range server.Args {
					if arg != expected.Args[j] {
						t.Errorf("ApplyWindowsTransformation() server[%d].Args[%d] = %v, want %v", i, j, arg, expected.Args[j])
					}
				}
			}
		})
	}
}

func TestWindowsService_IsAlreadyWrapped(t *testing.T) {
	ws := NewWindowsService()

	tests := []struct {
		name    string
		command string
		args    []interface{}
		want    bool
	}{
		{
			name:    "wrapped command",
			command: "cmd",
			args:    []interface{}{"/c", "npx", "server"},
			want:    true,
		},
		{
			name:    "not wrapped - no args",
			command: "cmd",
			args:    []interface{}{},
			want:    false,
		},
		{
			name:    "not wrapped - single arg",
			command: "cmd",
			args:    []interface{}{"/c"},
			want:    false,
		},
		{
			name:    "not wrapped - different command",
			command: "python",
			args:    []interface{}{"server.py"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ws.IsAlreadyWrapped(tt.command, tt.args); got != tt.want {
				t.Errorf("IsAlreadyWrapped() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWindowsService_ShouldWrapForWindows(t *testing.T) {
	ws := NewWindowsService()

	tests := []struct {
		name    string
		command string
		args    []interface{}
		want    bool
	}{
		{
			name:    "npx command should wrap on Windows",
			command: "npx",
			args:    []interface{}{"@modelcontextprotocol/server-filesystem"},
			want:    runtime.GOOS == "windows",
		},
		{
			name:    "npx combined command should wrap on Windows",
			command: "npx @modelcontextprotocol/server-filesystem",
			args:    []interface{}{},
			want:    runtime.GOOS == "windows",
		},
		{
			name:    "already wrapped should not wrap",
			command: "cmd",
			args:    []interface{}{"/c", "npx", "server"},
			want:    false,
		},
		{
			name:    "non-npx should not wrap",
			command: "python",
			args:    []interface{}{"server.py"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ws.ShouldWrapForWindows(tt.command, tt.args); got != tt.want {
				t.Errorf("ShouldWrapForWindows() = %v, want %v", got, tt.want)
			}
		})
	}
}
