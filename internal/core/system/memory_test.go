package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetAllMemoryPathsIncludesCompatibleFallbacks(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)

	paths := GetAllMemoryPaths(cwd)
	assertMemoryPaths(t, paths.Global, []string{
		filepath.Join(home, ".gen", "GEN.md"),
		filepath.Join(home, ".claude", "CLAUDE.md"),
		filepath.Join(home, ".codex", "AGENTS.md"),
	})
	assertMemoryPaths(t, paths.Project, []string{
		filepath.Join(cwd, ".gen", "GEN.md"),
		filepath.Join(cwd, "GEN.md"),
		filepath.Join(cwd, ".claude", "CLAUDE.md"),
		filepath.Join(cwd, "CLAUDE.md"),
		filepath.Join(cwd, "AGENTS.md"),
	})
}

func TestLoadMemoryFilesFallsBackToCodexAndPrefersClaude(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)

	writeMemoryFile(t, filepath.Join(home, ".codex", "AGENTS.md"), "global codex")
	writeMemoryFile(t, filepath.Join(cwd, "AGENTS.md"), "project codex")

	files := LoadMemoryFiles(cwd)
	if len(files) != 2 || !strings.Contains(files[0].Content, "global codex") ||
		!strings.Contains(files[1].Content, "project codex") {
		t.Fatalf("expected Codex fallback memory, got %+v", files)
	}

	writeMemoryFile(t, filepath.Join(cwd, "CLAUDE.md"), "project claude")
	files = LoadMemoryFiles(cwd)
	if !strings.Contains(files[1].Content, "project claude") ||
		strings.Contains(files[1].Content, "project codex") {
		t.Fatalf("expected CLAUDE.md to take precedence over AGENTS.md, got %q", files[1].Content)
	}
}

func TestFindActiveMemoryFileSkipsEmptyHigherPrecedenceCandidate(t *testing.T) {
	cwd := t.TempDir()
	writeMemoryFile(t, filepath.Join(cwd, "CLAUDE.md"), " \n")
	writeMemoryFile(t, filepath.Join(cwd, "AGENTS.md"), "project codex")

	paths := GetAllMemoryPaths(cwd)
	if got := FindActiveMemoryFile(paths.Project); got != filepath.Join(cwd, "AGENTS.md") {
		t.Fatalf("FindActiveMemoryFile(project) = %q, want AGENTS.md", got)
	}
	if got := FindMemoryFile(paths.Project); got != filepath.Join(cwd, "CLAUDE.md") {
		t.Fatalf("FindMemoryFile(project) = %q, want existing empty draft", got)
	}
}

func TestLoadMemoryFilesPreservesNativeRulesAndLocalOrder(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Setenv("HOME", home)

	writeMemoryFile(t, filepath.Join(home, ".gen", "GEN.md"), "global gen")
	writeMemoryFile(t, filepath.Join(home, ".claude", "CLAUDE.md"), "global claude")
	writeMemoryFile(t, filepath.Join(home, ".gen", "rules", "01-global.md"), "global rule")
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "GEN.md"), "project gen")
	writeMemoryFile(t, filepath.Join(cwd, "AGENTS.md"), "project codex")
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "rules", "01-project.md"), "project rule")
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "GEN.local.md"), "project local")

	files := LoadMemoryFiles(cwd)
	if len(files) != 5 {
		t.Fatalf("expected five memory files, got %d: %+v", len(files), files)
	}
	for i, want := range []string{"global gen", "global rule", "project gen", "project rule", "project local"} {
		if !strings.Contains(files[i].Content, want) {
			t.Errorf("file[%d] = %q, want content %q", i, files[i].Content, want)
		}
	}
}

func TestLoadMemoryFilesResolvesImportsAndCycles(t *testing.T) {
	cwd := t.TempDir()
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "GEN.md"), "# Root\n@a.md")
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "a.md"), "A\n@GEN.md")

	files := LoadMemoryFiles(cwd)
	if !strings.Contains(files[0].Content, "<!-- Imported: a.md -->") ||
		!strings.Contains(files[0].Content, "Skipped (cycle)") {
		t.Fatalf("expected imported content with cycle suppression, got %q", files[0].Content)
	}
}

func TestLoadMemoryFilesBlocksImportOutsideMemoryDirectory(t *testing.T) {
	cwd := t.TempDir()
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "GEN.md"), "@../outside.md")
	writeMemoryFile(t, filepath.Join(cwd, "outside.md"), "must not load")

	files := LoadMemoryFiles(cwd)
	if !strings.Contains(files[0].Content, "Import blocked (outside base)") ||
		strings.Contains(files[0].Content, "must not load") {
		t.Fatalf("outside import was not blocked: %q", files[0].Content)
	}
}

func TestListRulesFilesIsSortedAndMarkdownOnly(t *testing.T) {
	dir := t.TempDir()
	writeMemoryFile(t, filepath.Join(dir, "z.md"), "z")
	writeMemoryFile(t, filepath.Join(dir, "a.md"), "a")
	writeMemoryFile(t, filepath.Join(dir, "ignored.txt"), "ignored")

	assertMemoryPaths(t, ListRulesFiles(dir), []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "z.md"),
	})
}

func TestMemoryFormatsCreateCompatibleTemplates(t *testing.T) {
	cwd := filepath.Join("/tmp", "demo")
	path, ok := NewProjectMemoryFile(cwd, MemoryFormatCodex)
	if !ok || path != filepath.Join(cwd, "AGENTS.md") {
		t.Fatalf("NewProjectMemoryFile(codex) = %q, %v", path, ok)
	}
	body, ok := ProjectMemoryTemplate(cwd, MemoryFormatClaude)
	if !ok || !strings.HasPrefix(body, "# CLAUDE.md") {
		t.Fatalf("ProjectMemoryTemplate(claude) = %q, %v", body, ok)
	}
}

func TestLoadInstructionsIncludesLocalMemory(t *testing.T) {
	cwd := t.TempDir()
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "GEN.md"), "project")
	writeMemoryFile(t, filepath.Join(cwd, ".gen", "GEN.local.md"), "local")
	_, project := LoadInstructions(cwd)
	if !strings.Contains(project, "project") || !strings.Contains(project, "local") {
		t.Fatalf("project memory = %q", project)
	}
}

func TestFormatFileSize(t *testing.T) {
	for _, tc := range []struct {
		size int64
		want string
	}{{500, "500B"}, {1024, "1.0KB"}, {1024 * 1024, "1.0MB"}} {
		if got := FormatFileSize(tc.size); got != tc.want {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tc.size, got, tc.want)
		}
	}
}

func assertMemoryPaths(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("paths[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func writeMemoryFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
