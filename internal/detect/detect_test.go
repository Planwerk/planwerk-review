package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTechnologies_Go(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/foo\n\ngo 1.21\n")

	tags := Technologies(dir)
	assertContains(t, tags, "go")
}

func TestTechnologies_GoWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.work", "go 1.21\n\nuse ./service\n")
	sub := filepath.Join(dir, "service")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, sub, "go.mod", "module example.com/service\n\ngo 1.21\n")

	tags := Technologies(dir)
	assertContains(t, tags, "go")
}

func TestTechnologies_GoModInSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "operators", "keystone")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, sub, "go.mod", "module example.com/op\n\ngo 1.21\n")

	tags := Technologies(dir)
	assertContains(t, tags, "go")
}

func TestTechnologies_DockerInSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "images", "api")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, sub, "Dockerfile", "FROM alpine\n")

	tags := Technologies(dir)
	assertContains(t, tags, "docker")
}

func TestTechnologies_PythonInSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "services", "api")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, sub, "pyproject.toml", "[project]\nname = \"foo\"\n")

	tags := Technologies(dir)
	assertContains(t, tags, "python")
}

func TestTechnologies_Python(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname = \"foo\"\n")

	tags := Technologies(dir)
	assertContains(t, tags, "python")
}

func TestTechnologies_PythonRequirements(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask\n")

	tags := Technologies(dir)
	assertContains(t, tags, "python")
}

func TestTechnologies_TypeScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"foo"}`)
	writeFile(t, dir, "tsconfig.json", `{}`)

	tags := Technologies(dir)
	assertContains(t, tags, "typescript")
	assertNotContains(t, tags, "javascript")
}

func TestTechnologies_JavaScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"foo"}`)

	tags := Technologies(dir)
	assertContains(t, tags, "javascript")
	assertNotContains(t, tags, "typescript")
}

func TestTechnologies_Rust(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"foo\"\n")

	tags := Technologies(dir)
	assertContains(t, tags, "rust")
}

func TestTechnologies_Java(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project/>")

	tags := Technologies(dir)
	assertContains(t, tags, "java")
}

func TestTechnologies_Docker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Dockerfile", "FROM alpine\n")

	tags := Technologies(dir)
	assertContains(t, tags, "docker")
}

func TestTechnologies_DockerCompose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yaml", "version: '3'\n")

	tags := Technologies(dir)
	assertContains(t, tags, "docker")
}

func TestTechnologies_GitHubActions(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, wfDir, "ci.yml", "name: CI\n")

	tags := Technologies(dir)
	assertContains(t, tags, "github-actions")
}

func TestTechnologies_Helm(t *testing.T) {
	dir := t.TempDir()
	chartDir := filepath.Join(dir, "charts", "myapp")
	if err := os.MkdirAll(chartDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, chartDir, "Chart.yaml", "apiVersion: v2\nname: myapp\n")

	tags := Technologies(dir)
	assertContains(t, tags, "helm")
}

func TestTechnologies_Kubernetes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deploy.yaml", "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: foo\n")

	tags := Technologies(dir)
	assertContains(t, tags, "kubernetes")
}

func TestTechnologies_KubernetesNotPlainYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "database:\n  host: localhost\n  port: 5432\n")

	tags := Technologies(dir)
	assertNotContains(t, tags, "kubernetes")
}

func TestTechnologies_KubernetesSkipsGitHubWorkflows(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Enough workflow YAMLs to exhaust a naive scan budget.
	for i := 0; i < 25; i++ {
		writeFile(t, wfDir, fmt.Sprintf("wf%02d.yaml", i), "name: CI\non: push\n")
	}
	deployDir := filepath.Join(dir, "deploy")
	if err := os.MkdirAll(deployDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, deployDir, "manifest.yaml", "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: foo\n")

	tags := Technologies(dir)
	assertContains(t, tags, "kubernetes")
	assertContains(t, tags, "github-actions")
}

func TestTechnologies_DocsOnlyPackageJSON_VitePress(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"docs","devDependencies":{"vitepress":"^1.6.3"}}`)

	tags := Technologies(dir)
	assertNotContains(t, tags, "javascript")
	assertNotContains(t, tags, "typescript")
}

func TestTechnologies_DocsOnlyPackageJSON_Docusaurus(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"devDependencies":{"@docusaurus/core":"^3.0.0","@docusaurus/preset-classic":"^3.0.0"}}`)

	tags := Technologies(dir)
	assertNotContains(t, tags, "javascript")
}

func TestTechnologies_PackageJSONWithRuntimeDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"^18.0.0"}}`)

	tags := Technologies(dir)
	assertContains(t, tags, "javascript")
}

func TestTechnologies_PackageJSONMixedDevDeps(t *testing.T) {
	// VitePress plus a non-docs dev dependency (e.g. a real build tool) → still a JS project.
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"devDependencies":{"vitepress":"^1.6.3","vite":"^5.0.0"}}`)

	tags := Technologies(dir)
	assertContains(t, tags, "javascript")
}

func TestTechnologies_Multiple(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module foo\n\ngo 1.21\n")
	writeFile(t, dir, "Dockerfile", "FROM golang\n")
	writeFile(t, dir, "deploy.yaml", "apiVersion: apps/v1\nkind: Deployment\n")

	tags := Technologies(dir)
	assertContains(t, tags, "go")
	assertContains(t, tags, "docker")
	assertContains(t, tags, "kubernetes")
}

func TestTechnologies_Empty(t *testing.T) {
	dir := t.TempDir()
	tags := Technologies(dir)
	if len(tags) != 0 {
		t.Errorf("expected no tags for empty dir, got %v", tags)
	}
}

func TestTechnologies_Sorted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module foo\n")
	writeFile(t, dir, "Dockerfile", "FROM alpine\n")
	writeFile(t, dir, "pyproject.toml", "[project]\n")

	tags := Technologies(dir)
	if !slices.IsSorted(tags) {
		t.Errorf("tags not sorted: %v", tags)
	}
}

func TestTechnologies_SkipsVendor(t *testing.T) {
	dir := t.TempDir()
	vendorDir := filepath.Join(dir, "vendor", "somelib")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, vendorDir, "Chart.yaml", "apiVersion: v2\nname: vendored\n")

	tags := Technologies(dir)
	assertNotContains(t, tags, "helm")
}

// Helpers

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, tags []string, want string) {
	t.Helper()
	if !slices.Contains(tags, want) {
		t.Errorf("tags %v should contain %q", tags, want)
	}
}

func assertNotContains(t *testing.T, tags []string, unwanted string) {
	t.Helper()
	if slices.Contains(tags, unwanted) {
		t.Errorf("tags %v should not contain %q", tags, unwanted)
	}
}
