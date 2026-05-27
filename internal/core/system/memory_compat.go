package system

import (
	"fmt"
	"path/filepath"
)

const (
	MemoryFormatGen    = "gen"
	MemoryFormatClaude = "claude"
	MemoryFormatCodex  = "codex"
)

// MemoryFormat describes a supported primary memory-file convention.
// Registration order defines fallback precedence.
type MemoryFormat struct {
	ID              string
	GlobalPaths     func(home string) []string
	ProjectPaths    func(cwd string) []string
	NewProjectPath  func(cwd string) string
	ProjectTemplate func(projectName string) string
}

var memoryFormats = []MemoryFormat{
	{
		ID: MemoryFormatGen,
		GlobalPaths: func(home string) []string {
			return []string{filepath.Join(home, ".gen", "GEN.md")}
		},
		ProjectPaths: func(cwd string) []string {
			return []string{filepath.Join(cwd, ".gen", "GEN.md"), filepath.Join(cwd, "GEN.md")}
		},
		NewProjectPath: func(cwd string) string {
			return filepath.Join(cwd, ".gen", "GEN.md")
		},
		ProjectTemplate: func(projectName string) string {
			return projectMemoryTemplate("GEN.md", "GenCode", projectName)
		},
	},
	{
		ID: MemoryFormatClaude,
		GlobalPaths: func(home string) []string {
			return []string{filepath.Join(home, ".claude", "CLAUDE.md")}
		},
		ProjectPaths: func(cwd string) []string {
			return []string{filepath.Join(cwd, ".claude", "CLAUDE.md"), filepath.Join(cwd, "CLAUDE.md")}
		},
		NewProjectPath: func(cwd string) string {
			return filepath.Join(cwd, ".claude", "CLAUDE.md")
		},
		ProjectTemplate: func(projectName string) string {
			return projectMemoryTemplate("CLAUDE.md", "Claude Code and GenCode", projectName)
		},
	},
	{
		ID: MemoryFormatCodex,
		GlobalPaths: func(home string) []string {
			return []string{filepath.Join(home, ".codex", "AGENTS.md")}
		},
		ProjectPaths: func(cwd string) []string {
			return []string{filepath.Join(cwd, "AGENTS.md")}
		},
		NewProjectPath: func(cwd string) string {
			return filepath.Join(cwd, "AGENTS.md")
		},
		ProjectTemplate: func(projectName string) string {
			return projectMemoryTemplate("AGENTS.md", "Codex and GenCode", projectName)
		},
	},
}

// MemoryFormats returns supported memory file conventions in precedence order.
func MemoryFormats() []MemoryFormat {
	return append([]MemoryFormat(nil), memoryFormats...)
}

// NewProjectMemoryFile returns the project path created for a format.
func NewProjectMemoryFile(cwd, formatID string) (string, bool) {
	for _, format := range memoryFormats {
		if format.ID == formatID {
			return format.NewProjectPath(cwd), true
		}
	}
	return "", false
}

// ProjectMemoryTemplate renders a project memory file for a format.
func ProjectMemoryTemplate(cwd, formatID string) (string, bool) {
	for _, format := range memoryFormats {
		if format.ID == formatID {
			return format.ProjectTemplate(filepath.Base(cwd)), true
		}
	}
	return "", false
}

func projectMemoryTemplate(fileName, consumer, projectName string) string {
	return fmt.Sprintf(`# %s

This file provides guidance to %s when working with code in this repository.

## Project Overview

%s - Describe what this project does.

## Build & Run

`+"`"+`bash
# Add your build commands here
`+"`"+`

## Architecture

<!-- Key directories and their purpose -->

## Key Patterns

<!-- Important conventions to follow -->
`, fileName, consumer, projectName)
}
