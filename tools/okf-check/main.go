// Command okf-check validates the embedded OKF bundle and its repository
// discovery inventories without treating those inventories as behavior specs.
package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

const (
	bundleRoot                   = "knowledge"
	markdownInventoryPath        = "knowledge/references/repository-documents.md"
	assetInventoryPath           = "knowledge/references/repository-assets.md"
	legacyMigrationPath          = "knowledge/references/legacy-research-migrations.md"
	legacyResearchIndex          = "research/README.md"
	legacyResearchBaselinePath   = "tools/okf-check/legacy-research-baseline.txt"
	legacyResearchPlanvocabPath  = "tools/okf-check/legacy-research-planvocab.txt"
	publicationInventoryPath     = "tools/okf-check/publication-inventory.json"
	planvocabSourcePath          = "plancontract/planvocab/catalog_source.json"
	researchRoot                 = "knowledge/research/"
	researchNoteType             = "Research Note"
	legacyResearchInitial        = 31
	legacyResearchPlanvocabCount = 16
	publicationSchemaVersion     = "spanalyzer.okf-publication/v0alpha1"
	publicationRepositoryURL     = "https://github.com/apstndb/spanalyzer"
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

// legacyResearchPlanvocab records which initial legacy bodies were hashed
// planvocab inputs. It stays immutable after a body moves so the migration
// gate can continue to require an explicit evidence disposition.
//
//go:embed legacy-research-planvocab.txt
var legacyResearchPlanvocab string

// publicationInventory pins path-derived Knowledge Catalog entry identities.
// The normal gate fails closed when a concept, index, or hierarchy edge moves
// until the inventory change and its remote deletion consequences are reviewed.
//
//go:embed publication-inventory.json
var publicationInventoryJSON []byte

var (
	markdownLinkPattern = regexp.MustCompile(`\]\(([^)]+)\)`)
	footnotePattern     = regexp.MustCompile(`\[\^([A-Za-z0-9._-]+)\]`)
	rfc3339Pattern      = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)?(?:Z|[+-]([0-9]{2}):([0-9]{2}))$`)
	gitObjectIDPattern  = regexp.MustCompile(`^[0-9a-f]{40}(?:[0-9a-f]{24})?$`)
)

type frontMatter struct {
	OKFVersion               string                    `yaml:"okf_version"`
	Type                     string                    `yaml:"type"`
	Title                    string                    `yaml:"title"`
	Description              string                    `yaml:"description"`
	Resource                 string                    `yaml:"resource"`
	Tags                     []string                  `yaml:"tags"`
	Status                   string                    `yaml:"status"`
	StaleAfter               string                    `yaml:"stale_after"`
	Generated                *actorEvent               `yaml:"generated"`
	Verified                 verifiedEvents            `yaml:"verified"`
	UsageWindow              *usageWindow              `yaml:"usage_window"`
	Sources                  []source                  `yaml:"sources"`
	AssetFamilies            []assetFamily             `yaml:"asset_families"`
	LegacyResearchMigrations []legacyResearchMigration `yaml:"legacy_research_migrations"`
}

type legacyResearchMigration struct {
	LegacyPath         string                      `yaml:"legacy_path"`
	State              string                      `yaml:"state"`
	SourceRef          string                      `yaml:"source_ref"`
	SourceBlob         string                      `yaml:"source_blob"`
	Dispositions       []legacyResearchDisposition `yaml:"dispositions"`
	PlanvocabAction    string                      `yaml:"planvocab_action"`
	PlanvocabEvidence  []string                    `yaml:"planvocab_evidence"`
	PlanvocabSelectors []string                    `yaml:"planvocab_selectors"`
}

type legacyResearchDisposition struct {
	Kind       string   `yaml:"kind"`
	Scope      string   `yaml:"scope"`
	Successors []string `yaml:"successors"`
	Reason     string   `yaml:"reason"`
}

type source struct {
	ID           string       `yaml:"id"`
	Resource     string       `yaml:"resource"`
	Title        string       `yaml:"title"`
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
	Raw      []byte
	HasFront bool
}

type publicationInventory struct {
	SchemaVersion string                      `json:"schema_version"`
	Entries       []publicationInventoryEntry `json:"entries"`
	Retired       []publicationRetiredEntry   `json:"retired_entries"`
}

type publicationInventoryEntry struct {
	EntryID      string `json:"entry_id"`
	Path         string `json:"path"`
	ParentEntry  string `json:"parent_entry_id,omitempty"`
	DocumentKind string `json:"document_kind"`
}

type publicationRetiredEntry struct {
	EntryID          string `json:"entry_id"`
	FormerPath       string `json:"former_path"`
	RemovedAtCommit  string `json:"removed_at_commit"`
	Reason           string `json:"reason"`
	SuccessorEntryID string `json:"successor_entry_id,omitempty"`
}

type publicationManifest struct {
	SchemaVersion string                     `json:"schema_version"`
	OKFVersion    string                     `json:"okf_version"`
	Repository    string                     `json:"repository"`
	SourceCommit  string                     `json:"source_commit"`
	SourceState   string                     `json:"source_state"`
	BundleSHA256  string                     `json:"bundle_sha256"`
	DocumentCount int                        `json:"document_count"`
	Documents     []publicationManifestEntry `json:"documents"`
	Retired       []publicationRetiredEntry  `json:"retired_entries,omitempty"`
}

type publicationManifestEntry struct {
	publicationInventoryEntry
	Type           string                `json:"type,omitempty"`
	Title          string                `json:"title,omitempty"`
	Status         string                `json:"status,omitempty"`
	DocumentSHA256 string                `json:"document_sha256"`
	SourceURL      string                `json:"source_url,omitempty"`
	Resources      []publicationResource `json:"resources,omitempty"`
	Links          []publicationResource `json:"links,omitempty"`
}

type publicationResource struct {
	Original  string `json:"original"`
	Published string `json:"published"`
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
	gate := flags.String("gate", "all", "gate to run: conformance, quality, assets, publication, or all")
	publicationSourceCommit := flags.String("publication-source-commit", "", "full source commit for publication URLs (defaults to HEAD)")
	publicationManifestPath := flags.String("publication-manifest", "", "optional path for the ephemeral publication manifest")
	publicationRequireClean := flags.Bool("publication-require-clean", false, "require the repository worktree to match the source commit")
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
		"publication": func(root string) error {
			manifest, err := checkPublication(root, *publicationSourceCommit, *publicationRequireClean)
			if err != nil {
				return err
			}
			if *publicationManifestPath != "" {
				if err := writePublicationManifest(root, *publicationManifestPath, manifest); err != nil {
					return err
				}
			}
			return nil
		},
	}
	if *gate != "all" {
		check, ok := checks[*gate]
		if !ok {
			return fmt.Errorf("unknown gate %q; use conformance, quality, assets, publication, or all", *gate)
		}
		if err := check(root); err != nil {
			return fmt.Errorf("%s gate: %w", *gate, err)
		}
		fmt.Printf("OKF %s gate: OK\n", *gate)
		return nil
	}

	for _, name := range []string{"conformance", "quality", "assets", "publication"} {
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
	return checkLegacyResearch(root, documents)
}

func checkPublication(root, sourceCommit string, requireClean bool) (publicationManifest, error) {
	documents, err := readBundle(root)
	if err != nil {
		return publicationManifest{}, err
	}
	headCommit, err := publicationGitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return publicationManifest{}, fmt.Errorf("resolve publication checkout commit: %w", err)
	}
	if sourceCommit == "" {
		sourceCommit = headCommit
	}
	if !gitObjectIDPattern.MatchString(sourceCommit) {
		return publicationManifest{}, fmt.Errorf("publication source commit %q is not a full hexadecimal object ID", sourceCommit)
	}
	if sourceCommit != headCommit {
		return publicationManifest{}, fmt.Errorf("publication source commit %s does not match checkout HEAD %s", sourceCommit, headCommit)
	}

	sourceState, err := publicationSourceState(root)
	if err != nil {
		return publicationManifest{}, err
	}
	if requireClean && sourceState != "committed-tree" {
		return publicationManifest{}, errors.New("repository worktree differs from the publication source commit")
	}
	commitPinned := sourceState == "committed-tree"
	if commitPinned {
		if err := verifyPublicationBlob(root, sourceCommit, publicationInventoryPath, publicationInventoryJSON); err != nil {
			return publicationManifest{}, fmt.Errorf("verify publication inventory provenance: %w", err)
		}
	}

	rootDoc, ok := documents[path.Join(bundleRoot, "index.md")]
	if !ok {
		return publicationManifest{}, errors.New("publication bundle has no root index")
	}
	manifest := publicationManifest{
		SchemaVersion: publicationSchemaVersion,
		OKFVersion:    rootDoc.Front.OKFVersion,
		Repository:    publicationRepositoryURL,
		SourceCommit:  sourceCommit,
		SourceState:   sourceState,
	}
	var bundleMaterial strings.Builder
	for _, name := range sortedDocumentNames(documents) {
		doc := documents[name]
		sourceURL := ""
		if commitPinned {
			if err := verifyPublicationDocument(root, sourceCommit, doc); err != nil {
				return publicationManifest{}, err
			}
			sourceURL = publicationPathURL("blob", sourceCommit, name)
		}
		entry := publicationManifestEntry{
			publicationInventoryEntry: publicationIdentity(name),
			Type:                      doc.Front.Type,
			Title:                     doc.Front.Title,
			Status:                    doc.Front.Status,
			DocumentSHA256:            fmt.Sprintf("%x", sha256.Sum256(doc.Raw)),
			SourceURL:                 sourceURL,
		}
		bundleMaterial.WriteString(name)
		bundleMaterial.WriteByte(0)
		bundleMaterial.WriteString(entry.DocumentSHA256)
		bundleMaterial.WriteByte('\n')

		resources := []string{doc.Front.Resource}
		for _, source := range doc.Front.Sources {
			resources = append(resources, source.Resource)
		}
		for _, resource := range resources {
			if strings.TrimSpace(resource) == "" {
				continue
			}
			resolved, err := publicationReference(root, name, resource, sourceCommit, commitPinned)
			if err != nil {
				return publicationManifest{}, fmt.Errorf("%s: publication resource %q: %w", name, resource, err)
			}
			entry.Resources = append(entry.Resources, resolved)
		}
		for _, target := range markdownTargets(doc.Body) {
			resolved, err := publicationReference(root, name, target, sourceCommit, commitPinned)
			if err != nil {
				return publicationManifest{}, fmt.Errorf("%s: publication link %q: %w", name, target, err)
			}
			entry.Links = append(entry.Links, resolved)
		}
		manifest.Documents = append(manifest.Documents, entry)
	}
	manifest.DocumentCount = len(manifest.Documents)
	manifest.BundleSHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(bundleMaterial.String())))

	candidate := publicationInventory{SchemaVersion: publicationSchemaVersion}
	for _, entry := range manifest.Documents {
		candidate.Entries = append(candidate.Entries, entry.publicationInventoryEntry)
	}
	retained, err := checkPublicationInventory(candidate)
	if err != nil {
		return publicationManifest{}, err
	}
	manifest.Retired = retained.Retired
	return manifest, nil
}

func publicationIdentity(documentPath string) publicationInventoryEntry {
	relative := strings.TrimPrefix(documentPath, bundleRoot+"/")
	entryID := strings.TrimSuffix(relative, filepath.Ext(relative))
	kind := "concept"
	if filepath.Base(documentPath) == "index.md" || filepath.Base(documentPath) == "log.md" {
		kind = "navigation"
	}
	parent := ""
	dir := path.Dir(entryID)
	if entryID != "index" {
		if path.Base(entryID) == "index" {
			parentDir := path.Dir(dir)
			if parentDir == "." {
				parent = "index"
			} else {
				parent = path.Join(parentDir, "index")
			}
		} else {
			parent = path.Join(dir, "index")
		}
	}
	return publicationInventoryEntry{
		EntryID:      entryID,
		Path:         documentPath,
		ParentEntry:  parent,
		DocumentKind: kind,
	}
}

func checkPublicationInventory(candidate publicationInventory) (publicationInventory, error) {
	var retained publicationInventory
	if err := json.Unmarshal(publicationInventoryJSON, &retained); err != nil {
		return publicationInventory{}, fmt.Errorf("parse %s: %w", publicationInventoryPath, err)
	}
	if retained.SchemaVersion != publicationSchemaVersion {
		return publicationInventory{}, fmt.Errorf("%s: schema_version = %q, want %q", publicationInventoryPath, retained.SchemaVersion, publicationSchemaVersion)
	}
	seen := make(map[string]string, len(retained.Entries))
	for i, entry := range retained.Entries {
		if entry.EntryID == "" || entry.Path == "" || entry.DocumentKind == "" {
			return publicationInventory{}, fmt.Errorf("%s: entries[%d] has an empty identity field", publicationInventoryPath, i)
		}
		if previous, ok := seen[entry.EntryID]; ok {
			return publicationInventory{}, fmt.Errorf("%s: duplicate entry_id %q for %s and %s", publicationInventoryPath, entry.EntryID, previous, entry.Path)
		}
		seen[entry.EntryID] = entry.Path
		if i > 0 && retained.Entries[i-1].Path >= entry.Path {
			return publicationInventory{}, fmt.Errorf("%s: entries are not strictly ordered by path at %q", publicationInventoryPath, entry.Path)
		}
	}
	if err := checkPublicationParents(retained.Entries); err != nil {
		return publicationInventory{}, fmt.Errorf("%s: %w", publicationInventoryPath, err)
	}
	for i, entry := range retained.Retired {
		if entry.EntryID == "" || entry.FormerPath == "" || entry.Reason == "" || !gitObjectIDPattern.MatchString(entry.RemovedAtCommit) {
			return publicationInventory{}, fmt.Errorf("%s: retired_entries[%d] has incomplete identity, reason, or removal commit", publicationInventoryPath, i)
		}
		if previous, ok := seen[entry.EntryID]; ok {
			return publicationInventory{}, fmt.Errorf("%s: retired entry_id %q conflicts with %s", publicationInventoryPath, entry.EntryID, previous)
		}
		seen[entry.EntryID] = entry.FormerPath
		if entry.SuccessorEntryID != "" {
			if _, ok := findPublicationEntry(retained.Entries, entry.SuccessorEntryID); !ok {
				return publicationInventory{}, fmt.Errorf("%s: retired entry_id %q names missing successor %q", publicationInventoryPath, entry.EntryID, entry.SuccessorEntryID)
			}
		}
	}
	if !reflect.DeepEqual(retained.Entries, candidate.Entries) {
		return publicationInventory{}, fmt.Errorf("%s: active publication identity mismatch; record removed or moved IDs in retired_entries", publicationInventoryPath)
	}
	return retained, nil
}

func checkPublicationParents(entries []publicationInventoryEntry) error {
	for _, entry := range entries {
		if entry.ParentEntry == "" {
			continue
		}
		parent, ok := findPublicationEntry(entries, entry.ParentEntry)
		if !ok {
			return fmt.Errorf("entry_id %q names missing parent_entry_id %q", entry.EntryID, entry.ParentEntry)
		}
		if parent.DocumentKind != "navigation" {
			return fmt.Errorf("entry_id %q names non-navigation parent_entry_id %q", entry.EntryID, entry.ParentEntry)
		}
	}
	return nil
}

func findPublicationEntry(entries []publicationInventoryEntry, entryID string) (publicationInventoryEntry, bool) {
	for _, entry := range entries {
		if entry.EntryID == entryID {
			return entry, true
		}
	}
	return publicationInventoryEntry{}, false
}

func publicationReference(root, documentPath, reference, sourceCommit string, commitPinned bool) (publicationResource, error) {
	result := publicationResource{Original: reference, Published: reference}
	trimmed := strings.TrimSpace(reference)
	if strings.HasPrefix(trimmed, "#") {
		if commitPinned {
			result.Published = publicationPathURL("blob", sourceCommit, documentPath) + trimmed
		}
		return result, nil
	}
	resolved, local, err := resolveLocalTarget(root, documentPath, reference)
	if err != nil || !local {
		return result, err
	}
	if !commitPinned {
		return result, nil
	}
	entry, err := publicationTreeEntry(root, sourceCommit, resolved)
	if err != nil {
		return result, err
	}
	kind := entry.ObjectType
	result.Published = publicationPathURL(kind, sourceCommit, resolved)
	parsed, err := url.Parse(normalizeMarkdownTarget(reference))
	if err == nil {
		if parsed.RawQuery != "" {
			result.Published += "?" + parsed.RawQuery
		}
		if parsed.Fragment != "" {
			result.Published += "#" + parsed.Fragment
		}
	}
	return result, nil
}

type publicationGitTreeEntry struct {
	Mode       string
	ObjectType string
	ObjectID   string
}

func publicationTreeEntry(root, sourceCommit, repositoryPath string) (publicationGitTreeEntry, error) {
	if repositoryPath == "." {
		objectID, err := publicationGitOutput(root, "rev-parse", "--verify", sourceCommit+"^{tree}")
		if err != nil || !gitObjectIDPattern.MatchString(objectID) {
			return publicationGitTreeEntry{}, fmt.Errorf("resolve repository root tree at publication source commit %s", sourceCommit)
		}
		return publicationGitTreeEntry{Mode: "040000", ObjectType: "tree", ObjectID: objectID}, nil
	}
	command := publicationGitCommand(root, "ls-tree", "-z", sourceCommit, "--", repositoryPath)
	output, err := command.Output()
	if err != nil {
		return publicationGitTreeEntry{}, fmt.Errorf("resolve %s at publication source commit %s: %w", repositoryPath, sourceCommit, err)
	}
	output = bytes.TrimSuffix(output, []byte{0})
	if len(output) == 0 || bytes.Contains(output, []byte{0}) {
		return publicationGitTreeEntry{}, fmt.Errorf("%s does not name one object at publication source commit %s", repositoryPath, sourceCommit)
	}
	header, foundPath, ok := bytes.Cut(output, []byte{'\t'})
	if !ok || string(foundPath) != repositoryPath {
		return publicationGitTreeEntry{}, fmt.Errorf("%s does not resolve exactly at publication source commit %s", repositoryPath, sourceCommit)
	}
	fields := strings.Fields(string(header))
	if len(fields) != 3 || (fields[1] != "blob" && fields[1] != "tree") || !gitObjectIDPattern.MatchString(fields[2]) {
		return publicationGitTreeEntry{}, fmt.Errorf("%s has an unsupported Git tree entry at publication source commit %s", repositoryPath, sourceCommit)
	}
	return publicationGitTreeEntry{Mode: fields[0], ObjectType: fields[1], ObjectID: fields[2]}, nil
}

func verifyPublicationDocument(root, sourceCommit string, doc document) error {
	if err := verifyPublicationBlob(root, sourceCommit, doc.Path, doc.Raw); err != nil {
		return fmt.Errorf("%s: verify publication document provenance: %w", doc.Path, err)
	}
	return nil
}

func verifyPublicationBlob(root, sourceCommit, repositoryPath string, worktreeData []byte) error {
	entry, err := publicationTreeEntry(root, sourceCommit, repositoryPath)
	if err != nil {
		return err
	}
	if entry.ObjectType != "blob" || entry.Mode == "120000" {
		return fmt.Errorf("%s is not a regular blob at publication source commit %s", repositoryPath, sourceCommit)
	}
	committed, err := publicationGitCommand(root, "cat-file", "blob", entry.ObjectID).Output()
	if err != nil {
		return fmt.Errorf("read committed blob %s for %s: %w", entry.ObjectID, repositoryPath, err)
	}
	if !bytes.Equal(committed, worktreeData) {
		return fmt.Errorf("%s worktree content differs from publication source commit %s", repositoryPath, sourceCommit)
	}
	return nil
}

func publicationPathURL(kind, sourceCommit, repositoryPath string) string {
	if repositoryPath == "." {
		return publicationRepositoryURL + "/" + kind + "/" + sourceCommit
	}
	parts := strings.Split(filepath.ToSlash(repositoryPath), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return publicationRepositoryURL + "/" + kind + "/" + sourceCommit + "/" + strings.Join(parts, "/")
}

func normalizeMarkdownTarget(target string) string {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "<") && strings.Contains(target, ">") {
		return strings.TrimPrefix(strings.SplitN(target, ">", 2)[0], "<")
	}
	if cut := strings.IndexAny(target, " \t"); cut >= 0 {
		return target[:cut]
	}
	return target
}

func publicationSourceState(root string) (string, error) {
	output, err := publicationGitCommand(root, "status", "--porcelain", "--", ".").Output()
	if err != nil {
		return "", fmt.Errorf("inspect publication source state: %w", err)
	}
	if strings.TrimSpace(string(output)) == "" {
		return "committed-tree", nil
	}
	return "dirty-worktree", nil
}

func publicationGitOutput(root string, args ...string) (string, error) {
	output, err := publicationGitCommand(root, args...).Output()
	return strings.TrimSpace(string(output)), err
}

func publicationGitCommand(root string, args ...string) *exec.Cmd {
	commandArgs := append([]string{"-C", root}, args...)
	command := exec.Command("git", commandArgs...)
	command.Env = make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "GIT_NO_REPLACE_OBJECTS=") {
			command.Env = append(command.Env, item)
		}
	}
	command.Env = append(command.Env, "GIT_NO_REPLACE_OBJECTS=1")
	return command
}

func writePublicationManifest(root, name string, manifest publicationManifest) error {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("resolve publication manifest path %q: %w", name, err)
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return fmt.Errorf("compare publication manifest path %q with repository root: %w", name, err)
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("publication manifest %q must be written outside the repository", name)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode publication manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(name, data, 0o644); err != nil {
		return fmt.Errorf("write publication manifest %q: %w", name, err)
	}
	return nil
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

func checkLegacyResearch(root string, documents map[string]document) error {
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
	initialPlanvocab, err := parseLegacyResearchMarkdown(legacyResearchPlanvocab)
	if err != nil {
		return fmt.Errorf("legacy research planvocab baseline: %w", err)
	}
	if len(initialPlanvocab) != legacyResearchPlanvocabCount {
		return fmt.Errorf("legacy research planvocab baseline has %d paths; want immutable initial count %d", len(initialPlanvocab), legacyResearchPlanvocabCount)
	}
	if unknown := setDifference(initialPlanvocab, baseline); len(unknown) != 0 {
		return fmt.Errorf("legacy research planvocab baseline contains paths outside the immutable baseline: %v", unknown)
	}

	trackedMarkdown, err := gitTrackedFiles(root, "*.md")
	if err != nil {
		return err
	}
	if err := checkLegacyResearchMembership(trackedMarkdown, baseline, allowed); err != nil {
		return err
	}
	tracked, err := gitTrackedFiles(root)
	if err != nil {
		return err
	}
	currentPlanvocab, err := readPlanvocabLocalEvidence(root)
	if err != nil {
		return err
	}
	return checkLegacyResearchMigrations(root, documents, baseline, allowed, initialPlanvocab, currentPlanvocab, tracked)
}

func readPlanvocabLocalEvidence(root string) (map[string]struct{}, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(planvocabSourcePath)))
	if err != nil {
		return nil, fmt.Errorf("read planvocab source: %w", err)
	}
	var source struct {
		Info struct {
			LocalEvidence []string `json:"local_evidence"`
		} `json:"info"`
	}
	if err := json.Unmarshal(data, &source); err != nil {
		return nil, fmt.Errorf("parse %s: %w", planvocabSourcePath, err)
	}
	result := make(map[string]struct{}, len(source.Info.LocalEvidence))
	for _, name := range source.Info.LocalEvidence {
		result[filepath.ToSlash(name)] = struct{}{}
	}
	return result, nil
}

func checkLegacyResearchMigrations(
	root string,
	documents map[string]document,
	baseline, allowed, initialPlanvocab, currentPlanvocab map[string]struct{},
	tracked []string,
) error {
	doc, ok := documents[legacyMigrationPath]
	if !ok {
		return fmt.Errorf("missing legacy research migration ledger %q", legacyMigrationPath)
	}
	if doc.Front.Type != "Reference" {
		return fmt.Errorf("%s: type = %q, want Reference", legacyMigrationPath, doc.Front.Type)
	}

	entries := doc.Front.LegacyResearchMigrations
	seen := make(map[string]struct{}, len(entries))
	pending := make(map[string]struct{})
	completed := make(map[string]struct{})
	trackedSet := make(map[string]struct{}, len(tracked))
	for _, name := range tracked {
		trackedSet[name] = struct{}{}
	}
	var previous string
	for i, entry := range entries {
		name := entry.LegacyPath
		if _, ok := baseline[name]; !ok {
			return fmt.Errorf("%s: legacy_research_migrations[%d] path %q is outside the immutable baseline", legacyMigrationPath, i, name)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("%s: duplicate migration entry for %q", legacyMigrationPath, name)
		}
		if previous != "" && name < previous {
			return fmt.Errorf("%s: migration path %q is out of order after %q", legacyMigrationPath, name, previous)
		}
		seen[name] = struct{}{}
		previous = name

		_, wasPlanvocab := initialPlanvocab[name]
		switch entry.State {
		case "pending":
			pending[name] = struct{}{}
			if err := checkPendingLegacyResearchMigration(entry, wasPlanvocab, currentPlanvocab); err != nil {
				return fmt.Errorf("%s: %w", legacyMigrationPath, err)
			}
		case "completed":
			completed[name] = struct{}{}
			if err := checkCompletedLegacyResearchMigration(root, documents, entry, wasPlanvocab, currentPlanvocab, trackedSet); err != nil {
				return fmt.Errorf("%s: %w", legacyMigrationPath, err)
			}
			if err := checkRemovedLegacyResearchReferences(root, entry, tracked); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: migration entry %q state = %q; want pending or completed", legacyMigrationPath, name, entry.State)
		}
	}

	if missing := setDifference(baseline, seen); len(missing) != 0 {
		return fmt.Errorf("%s: baseline paths missing from migration ledger: %v", legacyMigrationPath, missing)
	}
	if missing, extra := setDifference(allowed, pending), setDifference(pending, allowed); len(missing) != 0 || len(extra) != 0 {
		return fmt.Errorf("legacy research active/ledger mismatch; active_without_pending=%v pending_without_active=%v", missing, extra)
	}
	if overlap := setIntersection(pending, completed); len(overlap) != 0 {
		return fmt.Errorf("legacy research pending/completed sets overlap: %v", overlap)
	}
	return nil
}

func checkPendingLegacyResearchMigration(entry legacyResearchMigration, wasPlanvocab bool, currentPlanvocab map[string]struct{}) error {
	if entry.SourceRef != "" || entry.SourceBlob != "" || len(entry.Dispositions) != 0 || len(entry.PlanvocabEvidence) != 0 || len(entry.PlanvocabSelectors) != 0 {
		return fmt.Errorf("pending migration %q must not declare completed disposition fields", entry.LegacyPath)
	}
	_, currentlyHashed := currentPlanvocab[entry.LegacyPath]
	if wasPlanvocab != currentlyHashed {
		return fmt.Errorf("pending migration %q planvocab baseline/current mismatch", entry.LegacyPath)
	}
	want := "not-applicable"
	if wasPlanvocab {
		want = "pending"
	}
	if entry.PlanvocabAction != want {
		return fmt.Errorf("pending migration %q planvocab_action = %q, want %q", entry.LegacyPath, entry.PlanvocabAction, want)
	}
	return nil
}

func checkCompletedLegacyResearchMigration(
	root string,
	documents map[string]document,
	entry legacyResearchMigration,
	wasPlanvocab bool,
	currentPlanvocab, tracked map[string]struct{},
) error {
	if !gitObjectIDPattern.MatchString(entry.SourceRef) || !gitObjectIDPattern.MatchString(entry.SourceBlob) {
		return fmt.Errorf("completed migration %q must declare full hexadecimal source_ref and source_blob object IDs", entry.LegacyPath)
	}
	if err := checkLegacySourceObject(root, entry); err != nil {
		return err
	}
	if len(entry.Dispositions) == 0 {
		return fmt.Errorf("completed migration %q has no dispositions", entry.LegacyPath)
	}
	for i, disposition := range entry.Dispositions {
		if err := checkLegacyResearchDisposition(documents, tracked, entry.LegacyPath, i, disposition); err != nil {
			return err
		}
	}
	if _, ok := currentPlanvocab[entry.LegacyPath]; ok {
		return fmt.Errorf("completed migration %q remains in planvocab local_evidence", entry.LegacyPath)
	}
	if !wasPlanvocab {
		if entry.PlanvocabAction != "not-applicable" || len(entry.PlanvocabEvidence) != 0 || len(entry.PlanvocabSelectors) != 0 {
			return fmt.Errorf("completed non-planvocab migration %q must use planvocab_action not-applicable without evidence or selectors", entry.LegacyPath)
		}
		return nil
	}
	if entry.PlanvocabAction != "path-moved" && entry.PlanvocabAction != "evidence-replaced" && entry.PlanvocabAction != "evidence-retained" {
		return fmt.Errorf("completed hashed migration %q planvocab_action = %q; want path-moved, evidence-replaced, or evidence-retained", entry.LegacyPath, entry.PlanvocabAction)
	}
	if len(entry.PlanvocabEvidence) == 0 || len(entry.PlanvocabSelectors) == 0 {
		return fmt.Errorf("completed hashed migration %q must map replacement evidence and catalog selectors", entry.LegacyPath)
	}
	for _, name := range entry.PlanvocabEvidence {
		if _, ok := currentPlanvocab[name]; !ok {
			return fmt.Errorf("completed hashed migration %q replacement evidence %q is not in current planvocab local_evidence", entry.LegacyPath, name)
		}
		if _, ok := tracked[name]; !ok {
			return fmt.Errorf("completed hashed migration %q replacement evidence %q is not tracked", entry.LegacyPath, name)
		}
	}
	for i, selector := range entry.PlanvocabSelectors {
		if strings.TrimSpace(selector) == "" {
			return fmt.Errorf("completed hashed migration %q planvocab_selectors[%d] is empty", entry.LegacyPath, i)
		}
	}
	return nil
}

func checkLegacyResearchDisposition(documents map[string]document, tracked map[string]struct{}, legacyPath string, index int, disposition legacyResearchDisposition) error {
	validKinds := map[string]struct{}{
		"retain-move": {}, "split": {}, "merge": {}, "absorb": {}, "executable-replacement": {}, "retire": {},
	}
	if _, ok := validKinds[disposition.Kind]; !ok {
		return fmt.Errorf("completed migration %q dispositions[%d] kind = %q", legacyPath, index, disposition.Kind)
	}
	if strings.TrimSpace(disposition.Scope) == "" {
		return fmt.Errorf("completed migration %q dispositions[%d] has an empty scope", legacyPath, index)
	}
	if disposition.Kind == "retire" {
		validReasons := map[string]struct{}{
			"resolved-history": {}, "delivered-history": {}, "obsolete": {}, "superseded": {}, "reproducible-from-retained-evidence": {},
		}
		if _, ok := validReasons[disposition.Reason]; !ok {
			return fmt.Errorf("completed migration %q dispositions[%d] retirement reason = %q", legacyPath, index, disposition.Reason)
		}
	} else if len(disposition.Successors) == 0 {
		return fmt.Errorf("completed migration %q dispositions[%d] kind %q has no successors", legacyPath, index, disposition.Kind)
	}
	for _, successor := range disposition.Successors {
		name, _, _ := strings.Cut(successor, "#")
		if name == "" || filepath.IsAbs(name) || filepath.ToSlash(filepath.Clean(name)) != name || strings.HasPrefix(name, "../") {
			return fmt.Errorf("completed migration %q successor %q is not a canonical repository-relative path", legacyPath, successor)
		}
		if _, ok := tracked[name]; !ok {
			return fmt.Errorf("completed migration %q successor %q is not tracked", legacyPath, successor)
		}
		if strings.HasPrefix(name, bundleRoot+"/") && filepath.Ext(name) == ".md" {
			if _, ok := documents[name]; !ok {
				return fmt.Errorf("completed migration %q successor %q is not an OKF document", legacyPath, successor)
			}
		}
	}
	return nil
}

func checkLegacySourceObject(root string, entry legacyResearchMigration) error {
	commit, err := gitOutput(root, "rev-parse", "--verify", entry.SourceRef+"^{commit}")
	if err != nil || commit != entry.SourceRef {
		return fmt.Errorf("completed migration %q source_ref %q is not the exact retained commit", entry.LegacyPath, entry.SourceRef)
	}
	blob, err := gitOutput(root, "rev-parse", "--verify", entry.SourceRef+":"+entry.LegacyPath)
	if err != nil || blob != entry.SourceBlob {
		return fmt.Errorf("completed migration %q source_blob %q does not match %s:%s", entry.LegacyPath, entry.SourceBlob, entry.SourceRef, entry.LegacyPath)
	}
	return nil
}

func gitOutput(root string, args ...string) (string, error) {
	commandArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", commandArgs...).Output()
	return strings.TrimSpace(string(output)), err
}

func checkRemovedLegacyResearchReferences(root string, entry legacyResearchMigration, tracked []string) error {
	for _, name := range tracked {
		if name == legacyMigrationPath || name == legacyResearchBaselinePath || name == legacyResearchPlanvocabPath || !trackedTextPath(name) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("read tracked text %q while checking removed legacy references: %w", name, err)
		}
		for lineNumber, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, entry.LegacyPath) {
				continue
			}
			pinned := "/blob/" + entry.SourceRef + "/" + entry.LegacyPath
			if strings.Contains(line, pinned) {
				continue
			}
			return fmt.Errorf("removed legacy path %q still referenced by %s:%d", entry.LegacyPath, name, lineNumber+1)
		}
	}
	return nil
}

func trackedTextPath(name string) bool {
	switch filepath.Ext(name) {
	case ".go", ".json", ".md", ".sql", ".txt", ".yaml", ".yml":
		return true
	default:
		return false
	}
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

func setIntersection(left, right map[string]struct{}) []string {
	var result []string
	for item := range left {
		if _, ok := right[item]; ok {
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
	doc := document{Path: name, Body: body, Raw: data, HasFront: hasFront}
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
