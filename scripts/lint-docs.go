package main

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var linkPattern = regexp.MustCompile(`\[.*?\]\(([^)]+)\)`)

const maxDocLines = 100

type checkResult struct {
	Name   string   `json:"name"`
	Errors []string `json:"errors"`
}

type report struct {
	OK     bool          `json:"ok"`
	Checks []checkResult `json:"checks"`
}

func main() {
	r := runChecks()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(r)
	if !r.OK {
		os.Exit(1)
	}
}

func runChecks() report {
	var r report

	if _, err := os.Stat("docs"); os.IsNotExist(err) {
		r.Checks = append(r.Checks, checkResult{Name: "precondition", Errors: []string{"docs/ directory does not exist"}})
		r.OK = false
		return r
	}
	if _, err := os.Stat("CLAUDE.md"); os.IsNotExist(err) {
		r.Checks = append(r.Checks, checkResult{Name: "precondition", Errors: []string{"CLAUDE.md does not exist"}})
		r.OK = false
		return r
	}

	subdirs := findSubdirs()
	r.Checks = append(r.Checks,
		checkSubdirReadmes(subdirs),
		checkClaudeMDLinksToReadmes(subdirs),
		checkBrokenLinks(),
		checkOrphans(),
		checkLineCount(),
	)

	r.OK = true
	for _, c := range r.Checks {
		if len(c.Errors) > 0 {
			r.OK = false
			break
		}
	}
	return r
}

func findSubdirs() []string {
	entries, err := os.ReadDir("docs")
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// Every docs/ subdirectory has a README.md.
func checkSubdirReadmes(subdirs []string) checkResult {
	c := checkResult{Name: "subdir-readmes"}
	for _, dir := range subdirs {
		readme := filepath.Join("docs", dir, "README.md")
		if _, err := os.Stat(readme); os.IsNotExist(err) {
			c.Errors = append(c.Errors, "docs/"+dir+"/ is missing a README.md")
		}
	}
	return c
}

// CLAUDE.md links to each subdirectory's README.md.
func checkClaudeMDLinksToReadmes(subdirs []string) checkResult {
	c := checkResult{Name: "claude-md-links"}
	linkSet := resolvedLinkSet("CLAUDE.md")
	for _, dir := range subdirs {
		readme := filepath.Join("docs", dir, "README.md")
		if _, err := os.Stat(readme); os.IsNotExist(err) {
			continue // reported by subdir-readmes
		}
		if _, ok := linkSet[readme]; !ok {
			c.Errors = append(c.Errors, "CLAUDE.md does not link to "+readme)
		}
	}
	return c
}

// No broken markdown links in CLAUDE.md or any docs/ .md file.
func checkBrokenLinks() checkResult {
	c := checkResult{Name: "broken-links"}
	checkFileLinks(&c, "CLAUDE.md")
	_ = filepath.WalkDir("docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		checkFileLinks(&c, path)
		return nil
	})
	return c
}

func checkFileLinks(c *checkResult, file string) {
	dir := filepath.Dir(file)
	for _, link := range extractLinks(file) {
		resolved := filepath.Join(dir, link)
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			c.Errors = append(c.Errors, "broken link in "+file+": "+link)
		}
	}
}

// Every .md file under docs/ must be reachable from CLAUDE.md via transitive links.
func checkOrphans() checkResult {
	c := checkResult{Name: "orphans"}

	// BFS from CLAUDE.md
	visited := make(map[string]struct{})
	queue := []string{"CLAUDE.md"}
	visited["CLAUDE.md"] = struct{}{}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, target := range resolveLinks(current) {
			if _, seen := visited[target]; seen {
				continue
			}
			// Only follow links to existing .md files
			if !strings.HasSuffix(target, ".md") {
				continue
			}
			if _, err := os.Stat(target); err != nil {
				continue
			}
			visited[target] = struct{}{}
			queue = append(queue, target)
		}
	}

	// Check all docs/ .md files are visited
	_ = filepath.WalkDir("docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if _, ok := visited[path]; !ok {
			c.Errors = append(c.Errors, path+" is not reachable from CLAUDE.md")
		}
		return nil
	})
	return c
}

// Doc files over 100 lines.
func checkLineCount() checkResult {
	c := checkResult{Name: "line-count"}
	_ = filepath.WalkDir("docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		count, err := countLines(path)
		if err != nil {
			c.Errors = append(c.Errors, "reading "+path+": "+err.Error())
			return nil
		}
		if count > maxDocLines {
			c.Errors = append(c.Errors, path+" exceeds 100 lines ("+strconv.Itoa(count)+" lines)")
		}
		return nil
	})
	return c
}

// extractLinks returns raw relative link targets from a markdown file,
// skipping external URLs and anchor-only links.
func extractLinks(file string) []string {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()

	var links []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		for _, match := range linkPattern.FindAllStringSubmatch(scanner.Text(), -1) {
			target := match[1]
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				continue
			}
			if strings.HasPrefix(target, "#") {
				continue
			}
			links = append(links, target)
		}
	}
	return links
}

// resolveLinks returns link targets resolved to clean paths relative to cwd.
func resolveLinks(file string) []string {
	dir := filepath.Dir(file)
	raw := extractLinks(file)
	resolved := make([]string, 0, len(raw))
	for _, link := range raw {
		resolved = append(resolved, filepath.Clean(filepath.Join(dir, link)))
	}
	return resolved
}

// resolvedLinkSet returns a set of resolved link targets from a file.
func resolvedLinkSet(file string) map[string]struct{} {
	links := resolveLinks(file)
	set := make(map[string]struct{}, len(links))
	for _, l := range links {
		set[l] = struct{}{}
	}
	return set
}

func countLines(file string) (int, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

