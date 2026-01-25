package swagexport

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func buildFilePlan(swagDir string, opts Options) ([]fileEntry, error) {
	if strings.TrimSpace(swagDir) == "" {
		return nil, errors.New("swagDir 为空")
	}

	if st, err := os.Stat(swagDir); err != nil || !st.IsDir() {
		if err != nil {
			return nil, fmt.Errorf("swag-dir 不可用 (%s): %w", swagDir, err)
		}
		return nil, fmt.Errorf("swag-dir 不是目录: %s", swagDir)
	}

	excluder, err := newExcluder(opts.ExcludeGlobs)
	if err != nil {
		return nil, err
	}

	var out []fileEntry
	addFile := func(absPath string, relPath string) error {
		relPath = filepath.Clean(relPath)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		if relPath == "." || relPath == "" {
			return nil
		}

		if excluder != nil && excluder.IsExcluded(filepath.ToSlash(relPath)) {
			return nil
		}

		info, err := os.Lstat(absPath)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		out = append(out, fileEntry{
			AbsPath: absPath,
			RelPath: filepath.ToSlash(relPath),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
		})
		return nil
	}

	addDirFiltered := func(absDir string, relBase string, shouldInclude func(rel string, name string, d fs.DirEntry) bool) error {
		st, err := os.Stat(absDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !st.IsDir() {
			return nil
		}

		return filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			relToDir, err := filepath.Rel(absDir, path)
			if err != nil {
				return err
			}
			relToDir = filepath.Clean(relToDir)
			relToZip := filepath.Join(relBase, relToDir)
			relToZip = filepath.ToSlash(relToZip)

			if strings.HasPrefix(relToZip, "config/log/") {
				return nil
			}

			if shouldInclude != nil && !shouldInclude(relToZip, d.Name(), d) {
				return nil
			}

			return addFile(path, relToZip)
		})
	}

	composePath := filepath.Join(swagDir, "compose.yaml")
	if err := addFileIfExists(composePath, "compose.yaml", addFile); err != nil {
		return nil, err
	}

	nginxDir := filepath.Join(swagDir, "config", "nginx")
	if st, err := os.Stat(nginxDir); err != nil || !st.IsDir() {
		if err != nil {
			return nil, fmt.Errorf("nginx 配置目录不存在或不可访问 (%s): %w", nginxDir, err)
		}
		return nil, fmt.Errorf("nginx 配置目录不是目录: %s", nginxDir)
	}

	if err := addNginxTopLevelFiles(nginxDir, addFile, excluder); err != nil {
		return nil, err
	}

	if err := addDirFiltered(filepath.Join(nginxDir, "proxy-confs"), "config/nginx/proxy-confs", func(rel string, name string, d fs.DirEntry) bool {
		if strings.EqualFold(name, "README.md") {
			return false
		}
		l := strings.ToLower(name)
		if strings.Contains(l, "sample") || strings.Contains(l, "example") {
			return false
		}
		if !opts.ProxyConfOnly {
			return true
		}
		return strings.HasSuffix(l, ".conf") || strings.HasSuffix(l, ".conf.disabled")
	}); err != nil {
		return nil, err
	}

	if err := addDirFiltered(filepath.Join(nginxDir, "site-confs"), "config/nginx/site-confs", nil); err != nil {
		return nil, err
	}
	if err := addFileIfExists(filepath.Join(nginxDir, "site-conf", "default"), "config/nginx/site-conf/default", addFile); err != nil {
		return nil, err
	}

	if opts.Profile == ProfileStandard || opts.Profile == ProfileFull {
		standardDirs := []struct {
			abs string
			rel string
		}{
			{abs: filepath.Join(swagDir, "config", "custom-cont-init.d"), rel: "config/custom-cont-init.d"},
			{abs: filepath.Join(swagDir, "config", "custom-services.d"), rel: "config/custom-services.d"},
			{abs: filepath.Join(swagDir, "config", "crontabs"), rel: "config/crontabs"},
			{abs: filepath.Join(swagDir, "config", "php"), rel: "config/php"},
			{abs: filepath.Join(swagDir, "config", "www"), rel: "config/www"},
		}
		for _, d := range standardDirs {
			if err := addDirFiltered(d.abs, d.rel, nil); err != nil {
				return nil, err
			}
		}
	}

	if opts.Profile == ProfileFull {
		if opts.IncludeSecrets {
			if err := addDirFiltered(filepath.Join(swagDir, "config", "dns-conf"), "config/dns-conf", nil); err != nil {
				return nil, err
			}
			if err := addDirFiltered(filepath.Join(swagDir, "config", "keys"), "config/keys", nil); err != nil {
				return nil, err
			}
			if err := addDirFiltered(filepath.Join(swagDir, "config", "etc", "letsencrypt"), "config/etc/letsencrypt", nil); err != nil {
				return nil, err
			}
		}

		if err := addDirFiltered(filepath.Join(swagDir, "config", "fail2ban"), "config/fail2ban", func(rel string, name string, d fs.DirEntry) bool {
			if strings.EqualFold(name, "fail2ban.sqlite3") {
				return false
			}
			l := strings.ToLower(name)
			if strings.HasSuffix(l, ".conf") || strings.HasSuffix(l, ".local") {
				return true
			}
			if strings.HasSuffix(l, ".log") || strings.HasSuffix(l, ".gz") {
				return false
			}
			return false
		}); err != nil {
			return nil, err
		}
	}

	deduped := dedupeByRel(out)
	return deduped, nil
}

func addFileIfExists(absPath string, relPath string, add func(abs string, rel string) error) error {
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return add(absPath, relPath)
}

func addNginxTopLevelFiles(nginxDir string, add func(abs string, rel string) error, excluder *excluder) error {
	entries, err := os.ReadDir(nginxDir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		l := strings.ToLower(name)

		should := false
		if strings.HasSuffix(l, ".conf") || strings.HasSuffix(l, ".pem") {
			should = true
		}
		if name == ".htpasswd" {
			should = true
		}
		if !should {
			continue
		}

		rel := filepath.ToSlash(filepath.Join("config", "nginx", name))
		if excluder != nil && excluder.IsExcluded(rel) {
			continue
		}

		if err := add(filepath.Join(nginxDir, name), rel); err != nil {
			return err
		}
	}
	return nil
}

func dedupeByRel(in []fileEntry) []fileEntry {
	seen := make(map[string]fileEntry)
	for _, e := range in {
		if prev, ok := seen[e.RelPath]; ok {
			if e.ModTime.After(prev.ModTime) {
				seen[e.RelPath] = e
			}
			continue
		}
		seen[e.RelPath] = e
	}

	out := make([]fileEntry, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return out
}
