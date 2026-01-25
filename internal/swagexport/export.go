package swagexport

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Profile string

const (
	ProfileMinimal  Profile = "minimal"
	ProfileStandard Profile = "standard"
	ProfileFull     Profile = "full"
)

type Options struct {
	SwagDir        string
	OutPath        string
	Profile        Profile
	IncludeSecrets bool
	ProxyConfOnly  bool
	ExcludeGlobs   []string
	Version        string

	Now func() time.Time
}

type Result struct {
	OutPath      string
	FileCount    int
	TotalBytes   int64
	ManifestPath string
}

type fileEntry struct {
	AbsPath string
	RelPath string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
}

type manifest struct {
	FormatVersion  int       `json:"formatVersion"`
	CreatedAt      time.Time `json:"createdAt"`
	SwagDirBase    string    `json:"swagDirBase"`
	SwagCLIVersion string    `json:"swagCliVersion"`
	Profile        Profile   `json:"profile"`
	IncludeSecrets bool      `json:"includeSecrets"`
	ProxyConfOnly  bool      `json:"proxyConfOnly"`
	ExcludeGlobs   []string  `json:"excludeGlobs"`
	Files          []struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
	} `json:"files"`
}

const manifestName = "swag-cli-manifest.json"

func Export(opts Options) (Result, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if strings.TrimSpace(opts.SwagDir) == "" {
		return Result{}, errors.New("swag-dir 为空")
	}

	swagDir := filepath.Clean(strings.TrimSpace(opts.SwagDir))
	if opts.Profile == "" {
		opts.Profile = ProfileStandard
	}

	outPath := filepath.Clean(strings.TrimSpace(opts.OutPath))
	if outPath == "" {
		outPath = defaultOutPath(opts.Now())
	}

	entries, err := buildFilePlan(swagDir, opts)
	if err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("创建导出目录失败: %w", err)
	}

	zf, err := os.Create(outPath)
	if err != nil {
		return Result{}, fmt.Errorf("创建导出文件失败 (%s): %w", outPath, err)
	}
	defer func() { _ = zf.Close() }()

	zw := zip.NewWriter(zf)
	defer func() { _ = zw.Close() }()

	var total int64
	for _, e := range entries {
		n, err := writeZipFile(zw, e)
		if err != nil {
			return Result{}, err
		}
		total += n
	}

	manifestBytes, err := buildManifest(swagDir, opts, entries)
	if err != nil {
		return Result{}, err
	}
	if err := writeZipBytes(zw, manifestName, manifestBytes, 0o644, opts.Now()); err != nil {
		return Result{}, err
	}

	if err := zw.Close(); err != nil {
		return Result{}, fmt.Errorf("写入 zip 失败: %w", err)
	}
	if err := zf.Close(); err != nil {
		return Result{}, fmt.Errorf("关闭导出文件失败: %w", err)
	}

	return Result{
		OutPath:      outPath,
		FileCount:    len(entries),
		TotalBytes:   total,
		ManifestPath: manifestName,
	}, nil
}

func defaultOutPath(now time.Time) string {
	name := fmt.Sprintf("swag-export.%s.zip", now.Format("20060102-150405"))
	return filepath.Join(".", name)
}

func buildManifest(swagDir string, opts Options, entries []fileEntry) ([]byte, error) {
	m := manifest{
		FormatVersion:  1,
		CreatedAt:      opts.Now(),
		SwagDirBase:    filepath.Base(swagDir),
		SwagCLIVersion: strings.TrimSpace(opts.Version),
		Profile:        opts.Profile,
		IncludeSecrets: opts.IncludeSecrets,
		ProxyConfOnly:  opts.ProxyConfOnly,
		ExcludeGlobs:   append([]string(nil), opts.ExcludeGlobs...),
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].RelPath < entries[j].RelPath })
	for _, e := range entries {
		m.Files = append(m.Files, struct {
			Path string `json:"path"`
			Size int64  `json:"size"`
		}{
			Path: e.RelPath,
			Size: e.Size,
		})
	}

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("生成 manifest 失败: %w", err)
	}
	return b, nil
}

func writeZipFile(zw *zip.Writer, e fileEntry) (int64, error) {
	if zw == nil {
		return 0, errors.New("zip writer 为空")
	}

	f, err := os.Open(e.AbsPath)
	if err != nil {
		return 0, fmt.Errorf("打开文件失败 (%s): %w", e.AbsPath, err)
	}
	defer func() { _ = f.Close() }()

	h := &zip.FileHeader{
		Name:     toZipPath(e.RelPath),
		Method:   zip.Deflate,
		Modified: e.ModTime,
	}
	h.SetMode(e.Mode)

	w, err := zw.CreateHeader(h)
	if err != nil {
		return 0, fmt.Errorf("写入 zip 条目失败 (%s): %w", e.RelPath, err)
	}

	n, err := io.Copy(w, f)
	if err != nil {
		return n, fmt.Errorf("写入 zip 内容失败 (%s): %w", e.RelPath, err)
	}
	return n, nil
}

func writeZipBytes(zw *zip.Writer, relPath string, data []byte, mode os.FileMode, modTime time.Time) error {
	if zw == nil {
		return errors.New("zip writer 为空")
	}
	h := &zip.FileHeader{
		Name:     toZipPath(relPath),
		Method:   zip.Deflate,
		Modified: modTime,
	}
	h.SetMode(mode)

	w, err := zw.CreateHeader(h)
	if err != nil {
		return fmt.Errorf("写入 zip 条目失败 (%s): %w", relPath, err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("写入 zip 内容失败 (%s): %w", relPath, err)
	}
	return nil
}

func toZipPath(rel string) string {
	s := filepath.ToSlash(rel)
	s = strings.TrimPrefix(s, "./")
	s = strings.TrimPrefix(s, "/")
	return s
}
