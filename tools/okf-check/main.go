// Command okf-check validates the embedded OKF bundle and its repository
// discovery inventories without treating those inventories as behavior specs.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	bundleRoot            = "knowledge"
	markdownInventoryPath = "knowledge/references/repository-documents.md"
	assetInventoryPath    = "knowledge/references/repository-assets.md"
)

var (
	markdownLinkPattern = regexp.MustCompile(`\]\(([^)]+)\)`)
	footnotePattern     = regexp.MustCompile(`\[\^([A-Za-z0-9._-]+)\]`)
)

type frontMatter struct {
	OKFVersion    string        `yaml:"okf_version"`
	Type          string        `yaml:"type"`
	Resource      string        `yaml:"resource"`
	Sources       []source      `yaml:"sources"`
	AssetFamilies []assetFamily `yaml:"asset_families"`
}

type source struct {
	ID       string `yaml:"id"`
	Resource string `yaml:"resource"`
}

type assetFamily struct {
	ID    string   `yaml:"id"`
	Paths []string `yaml:"paths"`
}

type document struct {
	Path     string
	Front    frontMatter
	RawFront map[string]any
	Body     string
	HasFront bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("okf-check", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "", "repository root (auto-detected when omitted)")
	gate := flags.String("gate", "all", "gate to run: conformance, quality, assets, or all")
	if err := flags.Parse(args); err != nil {
		return err
	}

	root, err := resolveRepoRoot(*repoRoot)
	if err != nil {
		return err
	}

	checks := map[string]func(string) error{
		"conformance": checkConformance,
		"quality":     checkQuality,
		"assets":      checkAssets,
	}
	if *gate != "all" {
		check, ok := checks[*gate]
		if !ok {
			return fmt.Errorf("unknown gate %q; use conformance, quality, assets, or all", *gate)
		}
		if err := check(root); err != nil {
			return fmt.Errorf("%s gate: %w", *gate, err)
		}
		fmt.Printf("OKF %s gate: OK\n", *gate)
		return nil
	}

	for _, name := range []string{"conformance", "quality", "assets"} {
		if err := checks[name](root); err != nil {
			return fmt.Errorf("%s gate: %w", name, err)
		}
		fmt.Printf("OKF %s gate: OK\n", name)
	}
	return nil
}

func checkConformance(root string) error {
	documents, err := readBundle(root)
	if err != nil {
		return err
	}
	rootIndex := filepath.ToSlash(filepath.Join(bundleRoot, "index.md"))
	doc, ok := documents[rootIndex]
	if !ok {
		return fmt.Errorf("missing bundle root %q", rootIndex)
	}
	if !doc.HasFront {
		return fmt.Errorf("%s: root index must declare okf_version", rootIndex)
	}
	if doc.Front.OKFVersion != "0.2" {
		return fmt.Errorf("%s: okf_version = %q, want %q", rootIndex, doc.Front.OKFVersion, "0.2")
	}
	if len(doc.RawFront) != 1 {
		return fmt.Errorf("%s: root index frontmatter may contain only okf_version", rootIndex)
	}

	for name, doc := range documents {
		base := filepath.Base(name)
		switch base {
		case "index.md":
			if name != rootIndex && doc.HasFront {
				return fmt.Errorf("%s: subdirectory index must not have frontmatter", name)
			}
		case "log.md":
			if doc.HasFront {
				return fmt.Errorf("%s: reserved log must not have frontmatter", name)
			}
		default:
			if !doc.HasFront {
				return fmt.Errorf("%s: concept has no YAML frontmatter", name)
			}
			if strings.TrimSpace(doc.Front.Type) == "" {
				return fmt.Errorf("%s: concept has an empty type", name)
			}
			for i, source := range doc.Front.Sources {
				if strings.TrimSpace(source.Resource) == "" {
					return fmt.Errorf("%s: sources[%d] has no resource", name, i)
				}
			}
		}
	}
	return nil
}

func checkQuality(root string) error {
	documents, err := readBundle(root)
	if err != nil {
		return err
	}
	if err := checkReachability(root, documents); err != nil {
		return err
	}
	for _, name := range sortedDocumentNames(documents) {
		doc := documents[name]
		if err := checkDocumentPaths(root, doc); err != nil {
			return err
		}
		if err := checkSourceIDs(doc); err != nil {
			return err
		}
	}
	return checkMarkdownInventory(root, documents)
}

func checkAssets(root string) error {
	doc, err := readDocument(filepath.Join(root, filepath.FromSlash(assetInventoryPath)), assetInventoryPath)
	if err != nil {
		return err
	}
	if len(doc.Front.AssetFamilies) == 0 {
		return fmt.Errorf("%s: asset_families is empty", assetInventoryPath)
	}
	tracked, err := gitTrackedFiles(root)
	if err != nil {
		return err
	}

	seenFamily := make(map[string]struct{}, len(doc.Front.AssetFamilies))
	owners := make(map[string]map[string]struct{})
	for _, family := range doc.Front.AssetFamilies {
		if strings.TrimSpace(family.ID) == "" {
			return fmt.Errorf("%s: asset family has an empty id", assetInventoryPath)
		}
		if _, ok := seenFamily[family.ID]; ok {
			return fmt.Errorf("%s: duplicate asset family id %q", assetInventoryPath, family.ID)
		}
		seenFamily[family.ID] = struct{}{}
		if len(family.Paths) == 0 {
			return fmt.Errorf("%s: asset family %q has no paths", assetInventoryPath, family.ID)
		}
		for _, pattern := range family.Paths {
			if err := validateAssetPattern(pattern); err != nil {
				return fmt.Errorf("%s: family %q: %w", assetInventoryPath, family.ID, err)
			}
			matches := 0
			for _, trackedPath := range tracked {
				matched, err := path.Match(pattern, trackedPath)
				if err != nil {
					return fmt.Errorf("%s: invalid path pattern %q: %w", assetInventoryPath, pattern, err)
				}
				if !matched {
					continue
				}
				matches++
				if owners[trackedPath] == nil {
					owners[trackedPath] = make(map[string]struct{})
				}
				owners[trackedPath][family.ID] = struct{}{}
			}
			if matches == 0 {
				return fmt.Errorf("%s: family %q pattern %q matches no tracked files", assetInventoryPath, family.ID, pattern)
			}
		}
	}

	for trackedPath, familyOwners := range owners {
		if len(familyOwners) > 1 {
			ids := make([]string, 0, len(familyOwners))
			for id := range familyOwners {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			return fmt.Errorf("%s: tracked asset %q belongs to multiple families: %s", assetInventoryPath, trackedPath, strings.Join(ids, ", "))
		}
	}
	return nil
}

func validateAssetPattern(pattern string) error {
	if pattern == "" {
		return errors.New("empty asset path pattern")
	}
	if filepath.IsAbs(pattern) || strings.HasPrefix(pattern, "/") {
		return fmt.Errorf("asset path pattern %q must be repository-relative", pattern)
	}
	if filepath.ToSlash(filepath.Clean(pattern)) != pattern || strings.HasPrefix(pattern, "../") {
		return fmt.Errorf("asset path pattern %q is not canonical", pattern)
	}
	if strings.HasPrefix(pattern, bundleRoot+"/") {
		return fmt.Errorf("asset path pattern %q points inside the OKF bundle", pattern)
	}
	_, err := path.Match(pattern, "probe")
	if err != nil {
		return fmt.Errorf("invalid asset path pattern %q: %w", pattern, err)
	}
	return nil
}

func checkReachability(root string, documents map[string]document) error {
	rootIndex := filepath.ToSlash(filepath.Join(bundleRoot, "index.md"))
	reachable := map[string]struct{}{rootIndex: {}}
	queue := []string{rootIndex}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if filepath.Base(name) != "index.md" {
			continue
		}
		for _, target := range markdownTargets(documents[name].Body) {
			resolved, local, err := resolveLocalTarget(root, name, target)
			if err != nil {
				return fmt.Errorf("%s: link %q: %w", name, target, err)
			}
			if !local || !strings.HasPrefix(resolved, bundleRoot+"/") || filepath.Ext(resolved) != ".md" {
				continue
			}
			if _, ok := documents[resolved]; !ok {
				continue
			}
			if _, ok := reachable[resolved]; ok {
				continue
			}
			reachable[resolved] = struct{}{}
			queue = append(queue, resolved)
		}
	}

	for name := range documents {
		base := filepath.Base(name)
		if base == "index.md" || base == "log.md" {
			continue
		}
		if _, ok := reachable[name]; !ok {
			return fmt.Errorf("%s: concept is not reachable from %s through directory indexes", name, rootIndex)
		}
	}
	return nil
}

func checkDocumentPaths(root string, doc document) error {
	for _, target := range markdownTargets(doc.Body) {
		resolved, local, err := resolveLocalTarget(root, doc.Path, target)
		if err != nil {
			return fmt.Errorf("%s: path %q: %w", doc.Path, target, err)
		}
		if !local {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(resolved))); err != nil {
			return fmt.Errorf("%s: path %q resolves to missing %q: %w", doc.Path, target, resolved, err)
		}
	}
	resources := []string{doc.Front.Resource}
	for _, source := range doc.Front.Sources {
		resources = append(resources, source.Resource)
	}
	for _, resource := range resources {
		if !explicitLocalResource(resource) {
			continue
		}
		resolved, _, err := resolveLocalTarget(root, doc.Path, resource)
		if err != nil {
			return fmt.Errorf("%s: resource %q: %w", doc.Path, resource, err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(resolved))); err != nil {
			return fmt.Errorf("%s: resource %q resolves to missing %q: %w", doc.Path, resource, resolved, err)
		}
	}
	return nil
}

// sources[].resource may be a non-path scope descriptor under OKF v0.2. Only
// syntactically explicit relative or bundle-absolute paths are checked here.
func explicitLocalResource(resource string) bool {
	return strings.HasPrefix(resource, ".") || strings.HasPrefix(resource, "/")
}

func checkSourceIDs(doc document) error {
	ids := make(map[string]struct{}, len(doc.Front.Sources))
	for _, source := range doc.Front.Sources {
		if source.ID == "" {
			continue
		}
		if _, ok := ids[source.ID]; ok {
			return fmt.Errorf("%s: duplicate source id %q", doc.Path, source.ID)
		}
		ids[source.ID] = struct{}{}
	}
	for _, match := range footnotePattern.FindAllStringSubmatch(doc.Body, -1) {
		id := match[1]
		if _, ok := ids[id]; !ok {
			return fmt.Errorf("%s: footnote %q has no matching sources[].id", doc.Path, id)
		}
	}
	return nil
}

func checkMarkdownInventory(root string, documents map[string]document) error {
	doc, ok := documents[markdownInventoryPath]
	if !ok {
		return fmt.Errorf("missing Markdown inventory %q", markdownInventoryPath)
	}
	listed := make(map[string]struct{})
	for _, target := range markdownTargets(doc.Body) {
		resolved, local, err := resolveLocalTarget(root, doc.Path, target)
		if err != nil {
			return fmt.Errorf("%s: inventory link %q: %w", doc.Path, target, err)
		}
		if !local || filepath.Ext(resolved) != ".md" || strings.HasPrefix(resolved, bundleRoot+"/") {
			continue
		}
		listed[resolved] = struct{}{}
	}

	tracked, err := gitTrackedFiles(root, "*.md")
	if err != nil {
		return err
	}
	want := make(map[string]struct{})
	for _, name := range tracked {
		if !strings.HasPrefix(name, bundleRoot+"/") {
			want[name] = struct{}{}
		}
	}
	missing, extra := setDifference(want, listed), setDifference(listed, want)
	if len(missing) != 0 || len(extra) != 0 {
		return fmt.Errorf("%s: tracked Markdown membership mismatch; missing=%v extra=%v", markdownInventoryPath, missing, extra)
	}
	return nil
}

func setDifference(left, right map[string]struct{}) []string {
	var result []string
	for item := range left {
		if _, ok := right[item]; !ok {
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result
}

func gitTrackedFiles(root string, pathspecs ...string) ([]string, error) {
	args := []string{"-C", root, "ls-files", "--"}
	args = append(args, pathspecs...)
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("list tracked files: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	for i := range lines {
		lines[i] = filepath.ToSlash(lines[i])
	}
	sort.Strings(lines)
	return lines, nil
}

func resolveLocalTarget(root, documentPath, target string) (string, bool, error) {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "<") && strings.HasSuffix(target, ">") {
		target = strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")
	} else if cut := strings.IndexAny(target, " \t"); cut >= 0 {
		target = target[:cut]
	}
	if target == "" || strings.HasPrefix(target, "#") {
		return "", false, nil
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", false, err
	}
	if parsed.IsAbs() || parsed.Host != "" {
		return "", false, nil
	}
	localPath, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", false, err
	}
	if localPath == "" {
		return "", false, nil
	}
	var absolute string
	if strings.HasPrefix(localPath, "/") {
		absolute = filepath.Join(root, bundleRoot, filepath.FromSlash(strings.TrimPrefix(localPath, "/")))
	} else {
		absolute = filepath.Join(root, filepath.Dir(filepath.FromSlash(documentPath)), filepath.FromSlash(localPath))
	}
	absolute = filepath.Clean(absolute)
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return "", false, err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false, errors.New("path escapes repository root")
	}
	return filepath.ToSlash(relative), true, nil
}

func markdownTargets(body string) []string {
	matches := markdownLinkPattern.FindAllStringSubmatch(body, -1)
	targets := make([]string, 0, len(matches))
	for _, match := range matches {
		targets = append(targets, match[1])
	}
	return targets
}

func readBundle(root string) (map[string]document, error) {
	documents := make(map[string]document)
	knowledgeRoot := filepath.Join(root, bundleRoot)
	err := filepath.WalkDir(knowledgeRoot, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		doc, err := readDocument(filePath, name)
		if err != nil {
			return err
		}
		documents[name] = doc
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read OKF bundle: %w", err)
	}
	return documents, nil
}

func readDocument(filePath, name string) (document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return document{}, fmt.Errorf("read %s: %w", name, err)
	}
	front, body, hasFront, err := splitDocument(string(data))
	if err != nil {
		return document{}, fmt.Errorf("%s: %w", name, err)
	}
	doc := document{Path: name, Body: body, HasFront: hasFront}
	if !hasFront {
		return doc, nil
	}
	if err := yaml.Unmarshal([]byte(front), &doc.Front); err != nil {
		return document{}, fmt.Errorf("%s: parse YAML frontmatter: %w", name, err)
	}
	if err := yaml.Unmarshal([]byte(front), &doc.RawFront); err != nil {
		return document{}, fmt.Errorf("%s: parse raw YAML frontmatter: %w", name, err)
	}
	return doc, nil
}

func splitDocument(contents string) (front, body string, hasFront bool, err error) {
	contents = strings.ReplaceAll(contents, "\r\n", "\n")
	lines := strings.Split(contents, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return "", contents, false, nil
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] != "---" {
			continue
		}
		return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), true, nil
	}
	return "", "", false, errors.New("unterminated YAML frontmatter")
}

func sortedDocumentNames(documents map[string]document) []string {
	names := make([]string, 0, len(documents))
	for name := range documents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func resolveRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		root, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve repository root: %w", err)
		}
		return root, nil
	}
	start, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return findRepoRoot(start)
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve repository-root search path: %w", err)
	}
	for {
		if regularFile(filepath.Join(dir, "go.work")) && regularFile(filepath.Join(dir, bundleRoot, "index.md")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %q; pass --repo-root", start)
		}
		dir = parent
	}
}

func regularFile(filePath string) bool {
	info, err := os.Stat(filePath)
	return err == nil && info.Mode().IsRegular()
}
