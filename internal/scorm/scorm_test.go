package scorm

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

const manifestXML = `<?xml version="1.0"?>
<manifest identifier="M1" version="1">
  <metadata><schema>ADL SCORM</schema><schemaversion>1.2</schemaversion></metadata>
  <organizations default="ORG1">
    <organization identifier="ORG1">
      <title>Workplace Safety</title>
      <item identifier="I1" identifierref="R1">
        <title>Module one</title>
        <masteryscore>70</masteryscore>
      </item>
    </organization>
  </organizations>
  <resources>
    <resource identifier="R0" type="webcontent" href="unused.html"/>
    <resource identifier="R1" type="webcontent" adlcp:scormtype="sco" href="content/start.html"
              xmlns:adlcp="http://www.adlnet.org/xsd/adlcp_rootv1p2"/>
  </resources>
</manifest>`

func zipOf(t *testing.T, files map[string]string) ([]byte, int64) {
	t.Helper()
	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	for name, body := range files {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes(), int64(buf.Len())
}

func TestRead(t *testing.T) {
	t.Run("a package reads as its manifest describes it", func(t *testing.T) {
		body, size := zipOf(t, map[string]string{
			"imsmanifest.xml":    manifestXML,
			"content/start.html": "<html>hello</html>",
			"unused.html":        "<html>not the entry</html>",
			"content/style.css":  "body{}",
			"content/logo.png":   "\x89PNG",
		})
		pkg, err := Read(bytes.NewReader(body), size)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if pkg.Entry != "content/start.html" {
			t.Fatalf("the course starts at %q", pkg.Entry)
		}
		if pkg.Title != "Workplace Safety" {
			t.Fatalf("title is %q", pkg.Title)
		}
		if pkg.Mastery == nil || *pkg.Mastery != 70 {
			t.Fatalf("mastery is %v", pkg.Mastery)
		}
		if len(pkg.Files) != 5 {
			t.Fatalf("got %d files", len(pkg.Files))
		}
		for _, file := range pkg.Files {
			if file.Path == "content/style.css" && !strings.HasPrefix(file.ContentType, "text/css") {
				t.Fatalf("css came out as %q", file.ContentType)
			}
		}
	})

	t.Run("a package with no manifest is refused", func(t *testing.T) {
		body, size := zipOf(t, map[string]string{"index.html": "<html></html>"})
		if _, err := Read(bytes.NewReader(body), size); err == nil {
			t.Fatal("a zip with no manifest was accepted")
		}
	})

	t.Run("a manifest pointing at nothing is refused", func(t *testing.T) {
		body, size := zipOf(t, map[string]string{"imsmanifest.xml": manifestXML})
		_, err := Read(bytes.NewReader(body), size)
		if err == nil || !strings.Contains(err.Error(), "not in the package") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("a file trying to climb out is refused", func(t *testing.T) {
		body, size := zipOf(t, map[string]string{
			"imsmanifest.xml":  manifestXML,
			"../../etc/passwd": "root:x:0:0",
		})
		_, err := Read(bytes.NewReader(body), size)
		if err == nil || !strings.Contains(err.Error(), "climb out") {
			t.Fatalf("got %v, want the path to be refused", err)
		}
	})

	t.Run("something that is not a zip is refused", func(t *testing.T) {
		body := []byte("this is a pdf, honestly")
		if _, err := Read(bytes.NewReader(body), int64(len(body))); err == nil {
			t.Fatal("a non-zip was accepted")
		}
	})
}

func TestSafePath(t *testing.T) {
	for _, name := range []string{"/etc/passwd", "../secrets", "http://elsewhere/x", ""} {
		if _, err := safePath(name); err == nil {
			t.Errorf("%q was accepted", name)
		}
	}
	for name, want := range map[string]string{
		"content/start.html":      "content/start.html",
		"./content/start.html":    "content/start.html",
		"content\\start.html":     "content/start.html",
		"content/start.html?x=1":  "content/start.html",
		"content/./sub/../a.html": "content/a.html",
	} {
		got, err := safePath(name)
		if err != nil {
			t.Errorf("%q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("%q became %q, want %q", name, got, want)
		}
	}
}
