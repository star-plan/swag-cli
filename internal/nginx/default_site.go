package nginx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type HomepageConfig struct {
	Domain                   string
	UpstreamApp              string
	UpstreamPort             int
	UpstreamProto            string
	KeepServerNameUnderscore bool
}

type DefaultSiteEditor struct {
	Path string
}

func NewDefaultSiteEditor(path string) *DefaultSiteEditor {
	return &DefaultSiteEditor{Path: path}
}

type EditResult struct {
	Changed    bool
	BackupPath string
}

func (e *DefaultSiteEditor) SetHomepage(cfg HomepageConfig, dryRun bool) (EditResult, error) {
	if strings.TrimSpace(cfg.Domain) == "" {
		return EditResult{}, fmt.Errorf("domain is required")
	}
	if strings.TrimSpace(cfg.UpstreamApp) == "" {
		return EditResult{}, fmt.Errorf("upstream app is required")
	}
	if cfg.UpstreamPort <= 0 || cfg.UpstreamPort > 65535 {
		return EditResult{}, fmt.Errorf("invalid upstream port: %d", cfg.UpstreamPort)
	}
	if cfg.UpstreamProto != "http" && cfg.UpstreamProto != "https" {
		return EditResult{}, fmt.Errorf("invalid upstream proto: %s", cfg.UpstreamProto)
	}

	original, err := os.ReadFile(e.Path)
	if err != nil {
		return EditResult{}, err
	}

	var serverNameOverride *string
	if !cfg.KeepServerNameUnderscore {
		d := cfg.Domain
		serverNameOverride = &d
	}
	updated, err := updateDefaultSiteConf(string(original), cfg, false, serverNameOverride)
	if err != nil {
		return EditResult{}, err
	}

	if updated == string(original) {
		return EditResult{Changed: false}, nil
	}

	if dryRun {
		return EditResult{Changed: true}, nil
	}

	backupPath, err := backupFile(e.Path, original)
	if err != nil {
		return EditResult{}, err
	}

	if err := writeFileAtomic(e.Path, []byte(updated)); err != nil {
		return EditResult{}, err
	}

	return EditResult{Changed: true, BackupPath: backupPath}, nil
}

func (e *DefaultSiteEditor) ClearHomepage(domain string, restoreServerNameUnderscore bool, dryRun bool) (EditResult, error) {
	original, err := os.ReadFile(e.Path)
	if err != nil {
		return EditResult{}, err
	}

	cfg := HomepageConfig{
		Domain: domain,
	}

	var serverNameOverride *string
	if restoreServerNameUnderscore {
		underscore := "_"
		serverNameOverride = &underscore
	}
	updated, err := updateDefaultSiteConf(string(original), cfg, true, serverNameOverride)
	if err != nil {
		return EditResult{}, err
	}

	if updated == string(original) {
		return EditResult{Changed: false}, nil
	}

	if dryRun {
		return EditResult{Changed: true}, nil
	}

	backupPath, err := backupFile(e.Path, original)
	if err != nil {
		return EditResult{}, err
	}

	if err := writeFileAtomic(e.Path, []byte(updated)); err != nil {
		return EditResult{}, err
	}

	return EditResult{Changed: true, BackupPath: backupPath}, nil
}

func updateDefaultSiteConf(input string, cfg HomepageConfig, clear bool, serverNameOverride *string) (string, error) {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	hadTrailingNewline := strings.HasSuffix(normalized, "\n")
	lines := splitLinesPreserve(input)

	serverBlocks, err := findBlocks(lines, regexp.MustCompile(`^\s*server\s*\{`))
	if err != nil {
		return "", err
	}

	var mainServer *block
	var redirectServer *block

	for i := range serverBlocks {
		b := &serverBlocks[i]
		body := strings.Join(lines[b.Start:b.End+1], "\n")
		if !containsNonComment(body, "default_server") {
			continue
		}

		if containsNonComment(body, "listen 443") && containsNonComment(body, "default_server") {
			mainServer = b
		}
		if containsNonComment(body, "listen 80") && containsNonComment(body, "default_server") {
			redirectServer = b
		}
	}

	if mainServer == nil {
		return "", fmt.Errorf("main 443 default_server block not found")
	}

	if serverNameOverride != nil {
		if redirectServer != nil {
			setServerName(lines, *redirectServer, *serverNameOverride)
		}
		setServerName(lines, *mainServer, *serverNameOverride)
	}

	locBlocks, err := findBlocksInRange(lines, regexp.MustCompile(`^\s*location\s+/\s*\{`), mainServer.Start, mainServer.End)
	if err != nil {
		return "", err
	}
	if len(locBlocks) == 0 {
		return "", fmt.Errorf("location / block not found in main server block")
	}

	loc := locBlocks[0]
	var errUpdate error
	lines, errUpdate = updateLocationRoot(lines, loc, cfg, clear)
	if errUpdate != nil {
		return "", errUpdate
	}

	out := strings.Join(lines, "\n")
	if hadTrailingNewline {
		out += "\n"
	}
	return out, nil
}

type block struct {
	Start int
	End   int
}

func findBlocks(lines []string, startPattern *regexp.Regexp) ([]block, error) {
	return findBlocksInRange(lines, startPattern, 0, len(lines)-1)
}

func findBlocksInRange(lines []string, startPattern *regexp.Regexp, from int, to int) ([]block, error) {
	if from < 0 {
		from = 0
	}
	if to >= len(lines) {
		to = len(lines) - 1
	}
	if from > to {
		return []block{}, nil
	}

	var blocks []block
	for i := from; i <= to; i++ {
		if !startPattern.MatchString(lines[i]) {
			continue
		}

		depth := braceDelta(lines[i])
		start := i
		end := -1

		for j := i + 1; j <= to; j++ {
			depth += braceDelta(lines[j])
			if depth == 0 {
				end = j
				i = j
				break
			}
		}

		if end == -1 {
			return nil, fmt.Errorf("unclosed block starting at line %d", start+1)
		}
		blocks = append(blocks, block{Start: start, End: end})
	}

	return blocks, nil
}

func braceDelta(line string) int {
	trimmed := line
	if idx := strings.IndexByte(trimmed, '#'); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	open := strings.Count(trimmed, "{")
	close := strings.Count(trimmed, "}")
	return open - close
}

func setServerName(lines []string, b block, serverName string) {
	re := regexp.MustCompile(`^(\s*server_name\s+)([^;]+)(;.*)$`)
	for i := b.Start; i <= b.End; i++ {
		trim := strings.TrimSpace(lines[i])
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if !strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), "server_name") {
			continue
		}
		m := re.FindStringSubmatch(lines[i])
		if len(m) == 4 {
			lines[i] = m[1] + serverName + m[3]
			return
		}
	}
}

func updateLocationRoot(lines []string, loc block, cfg HomepageConfig, clear bool) ([]string, error) {
	if loc.End-loc.Start < 2 {
		return lines, nil
	}

	locationLine := lines[loc.Start]
	indent := leadingWhitespace(locationLine) + "    "

	bodyStart := loc.Start + 1
	bodyEnd := loc.End - 1

	var newBody []string
	for i := bodyStart; i <= bodyEnd; i++ {
		line := lines[i]
		if shouldRemoveFromLocation(line) {
			continue
		}
		newBody = append(newBody, line)
	}

	insertPos := firstNonCommentNonEmptyIndex(newBody)
	if insertPos < 0 {
		insertPos = len(newBody)
	}

	var inject []string
	if clear {
		inject = []string{
			indent + "try_files $uri $uri/ /index.html /index.php?$args =404;",
		}
	} else {
		inject = []string{
			indent + "include /config/nginx/proxy.conf;",
			indent + "include /config/nginx/resolver.conf;",
			indent + fmt.Sprintf("set $upstream_app %s;", cfg.UpstreamApp),
			indent + fmt.Sprintf("set $upstream_port %d;", cfg.UpstreamPort),
			indent + fmt.Sprintf("set $upstream_proto %s;", cfg.UpstreamProto),
			indent + "proxy_pass $upstream_proto://$upstream_app:$upstream_port;",
		}
	}

	newBody = append(newBody[:insertPos], append(inject, newBody[insertPos:]...)...)

	out := make([]string, 0, len(lines)-(bodyEnd-bodyStart+1)+len(newBody))
	out = append(out, lines[:bodyStart]...)
	out = append(out, newBody...)
	out = append(out, lines[bodyEnd+1:]...)
	return out, nil
}

func shouldRemoveFromLocation(line string) bool {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, "#") {
		return false
	}

	needles := []string{
		"try_files ",
		"include /config/nginx/proxy.conf;",
		"include /config/nginx/resolver.conf;",
		"set $upstream_app ",
		"set $upstream_port ",
		"set $upstream_proto ",
		"proxy_pass ",
	}

	for _, n := range needles {
		if strings.HasPrefix(s, n) {
			return true
		}
	}
	return false
}

func firstNonCommentNonEmptyIndex(lines []string) int {
	for i, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		return i
	}
	return -1
}

func leadingWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

func splitLinesPreserve(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return []string{""}
	}
	if strings.HasSuffix(s, "\n") {
		s = strings.TrimSuffix(s, "\n")
	}
	return strings.Split(s, "\n")
}

func containsNonComment(body string, needle string) bool {
	for _, line := range strings.Split(body, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func backupFile(path string, content []byte) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ts := time.Now().Format("20060102-150405")
	backup := filepath.Join(dir, fmt.Sprintf("%s.bak-%s", base, ts))
	if err := os.WriteFile(backup, content, 0o644); err != nil {
		return "", err
	}
	return backup, nil
}

func writeFileAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	tmp := filepath.Join(dir, fmt.Sprintf(".%s.tmp", filepath.Base(path)))

	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
