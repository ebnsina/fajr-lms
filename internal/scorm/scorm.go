// Package scorm reads a SCORM package: the zip a school was given by a
// publisher, and the manifest inside it that says where the course starts.
package scorm

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"path"
	"strconv"
	"strings"
)

const (
	// A package is course material, not a video library.
	MaxBytes   = 60 << 20
	MaxFiles   = 3000
	MaxOneFile = 20 << 20
)

// File is one file out of the package, ready to be stored and served.
type File struct {
	Path        string
	ContentType string
	Body        []byte
}

// Package is what a zip turned out to hold.
type Package struct {
	Title   string
	Entry   string
	Version string
	Mastery *int16
	Files   []File
	Bytes   int64
}

type manifest struct {
	XMLName       xml.Name `xml:"manifest"`
	SchemaVersion string   `xml:"metadata>schemaversion"`
	Organizations struct {
		Default string `xml:"default,attr"`
		Items   []struct {
			ID    string `xml:"identifier,attr"`
			Title string `xml:"title"`
			Items []struct {
				Title      string `xml:"title"`
				ResourceID string `xml:"identifierref,attr"`
				Mastery    string `xml:"masteryscore"`
			} `xml:"item"`
		} `xml:"organization"`
	} `xml:"organizations"`
	Resources struct {
		Resources []struct {
			ID         string `xml:"identifier,attr"`
			Type       string `xml:"type,attr"`
			ScormType  string `xml:"scormType,attr"`
			Href       string `xml:"href,attr"`
			Parameters string `xml:"parameters,attr"`
		} `xml:"resource"`
	} `xml:"resources"`
}

// Read unpacks a package, refusing anything that could not be served safely.
func Read(r io.ReaderAt, size int64) (Package, error) {
	if size <= 0 || size > MaxBytes {
		return Package{}, fmt.Errorf("scorm: a package must be under %d MB", MaxBytes>>20)
	}
	archive, err := zip.NewReader(r, size)
	if err != nil {
		return Package{}, fmt.Errorf("scorm: that file is not a zip")
	}
	if len(archive.File) > MaxFiles {
		return Package{}, fmt.Errorf("scorm: a package holds at most %d files", MaxFiles)
	}

	out := Package{Version: "1.2"}
	var manifestBody []byte
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name, err := safePath(entry.Name)
		if err != nil {
			return Package{}, err
		}
		if entry.UncompressedSize64 > MaxOneFile {
			return Package{}, fmt.Errorf("scorm: %s is too large to serve", name)
		}
		out.Bytes += int64(entry.UncompressedSize64)
		if out.Bytes > MaxBytes {
			return Package{}, fmt.Errorf("scorm: the package unpacks to more than %d MB", MaxBytes>>20)
		}

		body, err := readEntry(entry)
		if err != nil {
			return Package{}, err
		}
		if strings.EqualFold(name, "imsmanifest.xml") {
			manifestBody = body
		}
		out.Files = append(out.Files, File{Path: name, ContentType: contentType(name), Body: body})
	}

	if manifestBody == nil {
		return Package{}, fmt.Errorf("scorm: there is no imsmanifest.xml in this package")
	}
	if err := describe(&out, manifestBody); err != nil {
		return Package{}, err
	}
	if !holds(out.Files, out.Entry) {
		return Package{}, fmt.Errorf("scorm: the manifest points at %s, which is not in the package", out.Entry)
	}
	return out, nil
}

func readEntry(entry *zip.File) ([]byte, error) {
	file, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("scorm: %s could not be read", entry.Name)
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, MaxOneFile+1))
	if err != nil {
		return nil, fmt.Errorf("scorm: %s could not be read", entry.Name)
	}
	return body, nil
}

// describe reads the manifest for the title, where the course starts, and the
// mark a learner has to reach.
func describe(pkg *Package, body []byte) error {
	var found manifest
	if err := xml.Unmarshal(body, &found); err != nil {
		return fmt.Errorf("scorm: the manifest could not be read")
	}
	if version := strings.TrimSpace(found.SchemaVersion); version != "" {
		pkg.Version = version
	}

	// The organisation names the resource the course starts at; failing that,
	// the first resource with an address is the entry.
	wanted := ""
	for _, org := range found.Organizations.Items {
		if found.Organizations.Default != "" && org.ID != found.Organizations.Default {
			continue
		}
		if pkg.Title == "" {
			pkg.Title = strings.TrimSpace(org.Title)
		}
		for _, item := range org.Items {
			if item.ResourceID == "" {
				continue
			}
			if wanted == "" {
				wanted = item.ResourceID
			}
			if score, err := strconv.Atoi(strings.TrimSpace(item.Mastery)); err == nil &&
				score >= 0 && score <= 100 && pkg.Mastery == nil {
				mark := int16(score)
				pkg.Mastery = &mark
			}
		}
	}

	for _, resource := range found.Resources.Resources {
		if resource.Href == "" {
			continue
		}
		if wanted != "" && resource.ID != wanted {
			continue
		}
		entry, err := safePath(resource.Href + resource.Parameters)
		if err != nil {
			return err
		}
		pkg.Entry = entry
		break
	}
	if pkg.Entry == "" {
		for _, resource := range found.Resources.Resources {
			if resource.Href == "" {
				continue
			}
			entry, err := safePath(resource.Href)
			if err != nil {
				return err
			}
			pkg.Entry = entry
			break
		}
	}
	if pkg.Entry == "" {
		return fmt.Errorf("scorm: the manifest does not say where the course starts")
	}
	if pkg.Title == "" {
		pkg.Title = "Course package"
	}
	return nil
}

// safePath keeps a package inside its own folder: no absolute paths, no
// climbing out with .., no backslashes pretending to be separators.
func safePath(name string) (string, error) {
	clean := strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"), "./")
	if question := strings.IndexAny(clean, "?#"); question >= 0 {
		clean = clean[:question]
	}
	if clean == "" {
		return "", fmt.Errorf("scorm: the package holds a file with no name")
	}
	if strings.HasPrefix(clean, "/") || strings.Contains(clean, "://") {
		return "", fmt.Errorf("scorm: %s is not a path inside the package", name)
	}
	clean = path.Clean(clean)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("scorm: %s tries to climb out of the package", name)
	}
	if len(clean) > 500 {
		return "", fmt.Errorf("scorm: %s has too long a name", name)
	}
	return clean, nil
}

func holds(files []File, name string) bool {
	for _, file := range files {
		if file.Path == name {
			return true
		}
	}
	return false
}

func contentType(name string) string {
	if kind := mime.TypeByExtension(strings.ToLower(path.Ext(name))); kind != "" {
		return kind
	}
	return "application/octet-stream"
}
