package pixelforge_cart

import (
	"bytes"
	"io/fs"
	"testing"
)

func TestBundlePreservesAssetsAndLegacyJSON(t *testing.T) {
	project := []byte(`{"name":"test"}`)
	blob, e := EncodeBundle(project, map[string][]byte{"assets/sprites/hero.png": {1, 2, 3}})
	if e != nil {
		t.Fatal(e)
	}
	p, assets, e := DecodeBundle(blob)
	if e != nil || !bytes.Equal(p, project) {
		t.Fatalf("decode: %v", e)
	}
	got, e := fs.ReadFile(assets, "assets/sprites/hero.png")
	if e != nil || !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("missing asset: %v", e)
	}
	p, assets, e = DecodeBundle(project)
	if e != nil || assets != nil || !bytes.Equal(p, project) {
		t.Fatal("legacy JSON broke")
	}
}
func TestBundleRejectsEscapingPaths(t *testing.T) {
	for _, p := range []string{"../secret", "assets/../secret", "assets/../../x", "assets\\x"} {
		if _, e := EncodeBundle([]byte("{}"), map[string][]byte{p: {1}}); e == nil {
			t.Fatalf("accepted %q", p)
		}
	}
}
