package pixelforge_cart

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
)

const maxBundleBytes = 64 << 20

// EncodeBundle packages project JSON and assets without filesystem extraction.
// Asset keys use the runtime's assets/<relative path> convention.
func EncodeBundle(project []byte, assets map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	files := map[string][]byte{"project.pforge": project}
	for name, data := range assets {
		if !validAsset(name) {
			return nil, fmt.Errorf("cart bundle: unsafe asset path %q", name)
		}
		files[name] = data
	}
	total := 0
	// Stable ordering makes the same inputs produce the same bundle.
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data := files[name]
		total += len(data)
		if total > maxBundleBytes {
			return nil, fmt.Errorf("cart bundle exceeds 64 MiB")
		}
		f, e := w.Create(name)
		if e != nil {
			return nil, e
		}
		if _, e = f.Write(data); e != nil {
			return nil, e
		}
	}
	if e := w.Close(); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
func validAsset(name string) bool {
	return strings.HasPrefix(name, "assets/") && fs.ValidPath(name) && path.Clean(name) == name && !strings.Contains(name, "\\")
}

// DecodeBundle accepts asset-bearing ZIP carts and legacy JSON-only carts.
// ZIP assets stay read-only in memory. No archive entry is written to disk.
func DecodeBundle(data []byte) ([]byte, fs.FS, error) {
	if !bytes.HasPrefix(data, []byte{'P', 'K', 3, 4}) {
		return data, nil, nil
	}
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return nil, nil, fmt.Errorf("cart bundle: %w", e)
	}
	total := uint64(0)
	seen := map[string]bool{}
	for _, f := range z.File {
		if seen[f.Name] {
			return nil, nil, fmt.Errorf("cart bundle: duplicate %q", f.Name)
		}
		seen[f.Name] = true
		if f.Name != "project.pforge" && !validAsset(f.Name) {
			return nil, nil, fmt.Errorf("cart bundle: unsafe path %q", f.Name)
		}
		if f.UncompressedSize64 > maxBundleBytes-total {
			return nil, nil, fmt.Errorf("cart bundle exceeds 64 MiB")
		}
		total += f.UncompressedSize64
	}
	f, e := z.Open("project.pforge")
	if e != nil {
		return nil, nil, fmt.Errorf("cart bundle: missing project: %w", e)
	}
	defer f.Close()
	project, e := io.ReadAll(io.LimitReader(f, maxBundleBytes+1))
	if e != nil {
		return nil, nil, e
	}
	if len(project) > maxBundleBytes {
		return nil, nil, fmt.Errorf("cart bundle project too large")
	}
	return project, z, nil
}
