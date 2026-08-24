// Command okf-check validates the embedded OKF bundle and its repository
// discovery inventories without treating those inventories as behavior specs.
package main

import (
	_ "embed"
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
	"time"

	"github.com/goccy/go-yaml"
)

const (
	bundleRoot            = "knowledge"
	markdownInventoryPath = "knowledge/references/repository-documents.md"
	assetInventoryPath    = "knowledge/references/repository-assets.md"
	legacyResearchIndex   = "research/README.md"
	researchRoot          = "knowledge/research/"
	researchNoteType      = "Research Note"
	legacyResearchInitial = 31
)

// legacyResearchBaseline records the immutable initial migration set. The
// active allowlist may shrink as notes move into the OKF bundle, but every
// active entry must remain a member of this baseline.
//
//go:embed legacy-research-baseline.txt
var legacyResearchBaseline string

// legacyResearchMarkdown is the active migration allowlist, not a discovery
// inventory. It may shrink to zero but must not admit a path outside the
// immutable baseline.
//
//go:embed legacy-research-markdown.txt
var legacyResearchMarkdown string

var (
	markdownLinkPattern = regexp.MustCompile(`\]\(([^)]+)\)`)
	footnotePattern     = regexp.MustCompile(`\[\^([A-Za-z0-9._-]+)\]`)
	rfc3339Pattern      = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?(?:Z|[+-]([0-9]{2}):([0-9]{2}))$`)
)

type frontMatter struct {
	OKFVersion    string         `yaml:"okf_version"`
	Type          string         `yaml:"type"`
	Title         string         `yaml:"title"`
	Description   string         `yaml:"description"`
	Resource      string         `yaml:"resource"`
	Tags          []string       `yaml:"tags"`
	Status        string         `yaml:"status"`
	StaleAfter    string         `yaml:"stale_after"`
	Generated     *actorEvent    `yaml:"generated"`
	Verified      verifiedEvents `yaml:"verified"`
	UsageWindow   *usageWindow   `yaml:"usage_window"`
	Sources       []source       `yaml:"sources"`
	AssetFamilies []assetFamily  `yaml:"asset_families"`
}

type source struct {
	ID           string       `yaml:"id"`
	Resource     string       `yaml:"resource"`
	LastModified string       `yaml:"last_modified"`
	UsageWindow  *usageWindow `yaml:"usage_window"`
}

type actorEvent struct {
	By string `yaml:"by"`
	At string `yaml:"at"`
}

type verifiedEvents []actorEvent

func (events *verifiedEvents) UnmarshalYAML(data []byte) error {
	var multiple []actorEvent
	if err := yaml.Unmarshal(data, &multiple); err == nil {
		*events = multiple
		return nil
	}
	var single actorEvent
	if err := yaml.Unmarshal(data, &single); err != nil {
		return err
	}
	*events = []actorEvent{single}
	return nil
}

type usageWindow struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
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
		if err := checkLifecycleAndTrustFields(doc); err != nil {
			return err
		}
		if err := checkResearchNote(doc); err != nil {
			return err
		}
	}
	if err := checkMarkdownInventory(root, documents); err != nil {
		return err
	}
	return checkLegacyResearch(root)
}

func checkLifecycleAndTrustFields(doc document) error {
	if doc.Front.Status != "" && !validStatus(doc.Front.Status) {
		return fmt.Errorf("%s: status = %q; want draft, stable, or deprecated", doc.Path, doc.Front.Status)
	}
	if err := checkOptionalTimestamp(doc.Path, "stale_after", doc.Front.StaleAfter); err != nil {
		return err
	}
	if doc.Front.Generated != nil {
		if strings.TrimSpace(doc.Front.Generated.By) == "" {
			return fmt.Errorf("%s: generated.by is empty", doc.Path)
		}
		if err := checkOptionalTimestamp(doc.Path, "generated.at", doc.Front.Generated.At); err != nil {
			return err
		}
	}
	for i, event := range doc.Front.Verified {
		if strings.TrimSpace(event.By) == "" {
			return fmt.Errorf("%s: verified[%d].by is empty", doc.Path, i)
		}
		if strings.TrimSpace(event.At) == "" {
			return fmt.Errorf("%s: verified[%d].at is empty", doc.Path, i)
		}
		if err := checkOptionalTimestamp(doc.Path, fmt.Sprintf("verified[%d].at", i), event.At); err != nil {
			return err
		}
	}
	if err := checkUsageWindow(doc.Path, "usage_window", doc.Front.UsageWindow); err != nil {
		return err
	}
	for i, source := range doc.Front.Sources {
		if err := checkOptionalTimestamp(doc.Path, fmt.Sprintf("sources[%d].last_modified", i), source.LastModified); err != nil {
			return err
		}
		if err := checkUsageWindow(doc.Path, fmt.Sprintf("sources[%d].usage_window", i), source.UsageWindow); err != nil {
			return err
		}
	}
	return nil
}

func checkResearchNote(doc document) error {
	isReserved := filepath.Base(doc.Path) == "index.md" || filepath.Base(doc.Path) == "log.md"
	isResearchPath := strings.HasPrefix(doc.Path, researchRoot)
	isResearchNote := doc.Front.Type == researchNoteType

	if isReserved && isResearchPath {
		return nil
	}
	if isResearchPath && !isResearchNote {
		return fmt.Errorf("%s: concepts below %s must use type %q", doc.Path, researchRoot, researchNoteType)
	}
	if isResearchNote {
		relative := strings.TrimPrefix(doc.Path, researchRoot)
		if !isResearchPath || !strings.Contains(relative, "/") {
			return fmt.Errorf("%s: Research Note must be below %s<area>/", doc.Path, researchRoot)
		}
	} else {
		return nil
	}

	required := []struct {
		name  string
		value string
	}{
		{name: "title", value: doc.Front.Title},
		{name: "description", value: doc.Front.Description},
		{name: "status", value: doc.Front.Status},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s: Research Note has an empty %s", doc.Path, field.name)
		}
	}
	if !validStatus(doc.Front.Status) {
		return fmt.Errorf("%s: Research Note status = %q; want draft, stable, or deprecated", doc.Path, doc.Front.Status)
	}
	if len(doc.Front.Tags) == 0 {
		return fmt.Errorf("%s: Research Note has no tags", doc.Path)
	}
	for i, tag := range doc.Front.Tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("%s: Research Note tags[%d] is empty", doc.Path, i)
		}
	}
	if len(doc.Front.Sources) == 0 {
		return fmt.Errorf("%s: Research Note has no sources", doc.Path)
	}
	return nil
}

func validStatus(status string) bool {
	switch status {
	case "draft", "stable", "deprecated":
		return true
	default:
		return false
	}
}

func checkUsageWindow(documentPath, field string, window *usageWindow) error {
	if window == nil {
		return nil
	}
	if strings.TrimSpace(window.From) == "" || strings.TrimSpace(window.To) == "" {
		return fmt.Errorf("%s: %s must contain both from and to", documentPath, field)
	}
	if err := checkOptionalTimestamp(documentPath, field+".from", window.From); err != nil {
		return err
	}
	return checkOptionalTimestamp(documentPath, field+".to", window.To)
}

func checkOptionalTimestamp(documentPath, field, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	match := rfc3339Pattern.FindStringSubmatch(value)
	if match == nil || len(match) == 3 && (match[1] > "23" || match[2] > "59") {
		return fmt.Errorf("%s: %s = %q is not an RFC 3339 datetime with an explicit UTC offset", documentPath, field, value)
	}
	// time.Time cannot represent RFC 3339 leap seconds, so this repository
	// intentionally rejects them rather than silently normalizing the value.
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%s: %s = %q is not an RFC 3339 datetime with an explicit UTC offset: %w", documentPath, field, value, err)
	}
	return nil
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

func checkLegacyResearch(root string) error {
	baseline, err := parseLegacyResearchMarkdown(legacyResearchBaseline)
	if err != nil {
		return fmt.Errorf("legacy research baseline: %w", err)
	}
	if len(baseline) != legacyResearchInitial {
		return fmt.Errorf("legacy research baseline has %d paths; want immutable initial count %d", len(baseline), legacyResearchInitial)
	}
	allowed, err := parseLegacyResearchMarkdown(legacyResearchMarkdown)
	if err != nil {
		return fmt.Errorf("legacy research allowlist: %w", err)
	}
	if unknown := setDifference(allowed, baseline); len(unknown) != 0 {
		return fmt.Errorf("legacy research allowlist contains paths outside the immutable baseline: %v", unknown)
	}

	tracked, err := gitTrackedFiles(root, "*.md")
	if err != nil {
		return err
	}
	return checkLegacyResearchMembership(tracked, baseline, allowed)
}

func parseLegacyResearchMarkdown(contents string) (map[string]struct{}, error) {
	allowed := make(map[string]struct{})
	var previous string
	for lineNumber, line := range strings.Split(contents, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		if !strings.HasPrefix(name, "research/") || name == legacyResearchIndex || filepath.Ext(name) != ".md" {
			return nil, fmt.Errorf("line %d: %q must be a Markdown path below research/ and not %s", lineNumber+1, name, legacyResearchIndex)
		}
		if filepath.ToSlash(filepath.Clean(name)) != name {
			return nil, fmt.Errorf("line %d: path %q is not canonical", lineNumber+1, name)
		}
		if _, exists := allowed[name]; exists {
			return nil, fmt.Errorf("line %d: duplicate path %q", lineNumber+1, name)
		}
		if previous != "" && name < previous {
			return nil, fmt.Errorf("line %d: path %q is out of order after %q", lineNumber+1, name, previous)
		}
		allowed[name] = struct{}{}
		previous = name
	}
	return allowed, nil
}

func checkLegacyResearchMembership(tracked []string, baseline, allowed map[string]struct{}) error {
	if unknown := setDifference(allowed, baseline); len(unknown) != 0 {
		return fmt.Errorf("legacy research allowlist contains paths outside the immutable baseline: %v", unknown)
	}
	actual := make(map[string]struct{})
	rootIndexFound := false
	for _, name := range tracked {
		if name == legacyResearchIndex {
			rootIndexFound = true
			continue
		}
		if strings.HasPrefix(name, "research/") && filepath.Ext(name) == ".md" {
			actual[name] = struct{}{}
		}
	}
	if !rootIndexFound {
		return fmt.Errorf("missing legacy research policy index %q", legacyResearchIndex)
	}

	stale, unapproved := setDifference(allowed, actual), setDifference(actual, allowed)
	if len(stale) != 0 || len(unapproved) != 0 {
		return fmt.Errorf("legacy research membership mismatch; stale_allowlist=%v unapproved=%v; author new research under knowledge/research and remove migrated paths from the allowlist", stale, unapproved)
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
