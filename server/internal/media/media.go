// Package media nimmt Spielerbilder und Wappen entgegen, prüft und säubert sie und
// liefert sie mit strengen Sicherheits-Headern aus.
//
// Rasterbilder (PNG, JPEG, WebP) werden dekodiert, quadratisch zugeschnitten, auf 512 px
// skaliert und neu als PNG kodiert – dabei fallen Metadaten weg. SVG (nur Wappen) wird über
// eine Allowlist neu geschrieben: keine Skripte, keine Event-Attribute, keine externen Verweise.
package media

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "golang.org/x/image/webp"
	_ "image/jpeg" // Dekoder registrieren

	"golang.org/x/image/draw"
)

const (
	MaxUpload = 1 << 20 // 1 MB
	Size      = 512     // Kantenlänge der gespeicherten Rasterbilder
)

var ErrUnsupported = errors.New("dateityp nicht erlaubt")

// Store legt Dateien in einem Ordner ab; Namen sind Zufalls-IDs, nie Nutzereingaben.
type Store struct{ dir string }

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Save prüft den Inhalt und speichert ihn. allowSVG nur für Wappen.
// Rückgabe: Dateiname (ID + Endung).
func (s *Store) Save(data []byte, allowSVG bool) (string, error) {
	if len(data) == 0 || len(data) > MaxUpload {
		return "", fmt.Errorf("datei leer oder größer als 1 MB")
	}
	var out []byte
	ext := ".png"
	switch ct := http.DetectContentType(data); {
	case ct == "image/png" || ct == "image/jpeg" || ct == "image/webp":
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("bild lesen: %w", err)
		}
		out, err = squarePNG(img)
		if err != nil {
			return "", err
		}
	case allowSVG && looksLikeSVG(data):
		clean, err := SanitizeSVG(data)
		if err != nil {
			return "", err
		}
		out, ext = clean, ".svg"
	default:
		return "", ErrUnsupported
	}
	id := make([]byte, 16)
	rand.Read(id)
	name := hex.EncodeToString(id) + ext
	if err := os.WriteFile(filepath.Join(s.dir, name), out, 0o644); err != nil {
		return "", err
	}
	return name, nil
}

// squarePNG schneidet mittig quadratisch zu und skaliert auf Size × Size.
func squarePNG(img image.Image) ([]byte, error) {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	if side < 16 {
		return nil, fmt.Errorf("bild zu klein")
	}
	src := image.Rect(b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2, 0, 0)
	src.Max = src.Min.Add(image.Pt(side, side))
	dst := image.NewRGBA(image.Rect(0, 0, Size, Size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, src, draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func looksLikeSVG(data []byte) bool {
	head := strings.ToLower(string(data[:min(len(data), 1024)]))
	return strings.Contains(head, "<svg")
}

// Allowlist für SVG: nur Formen, Gruppen, Verläufe und Beschnitt.
var svgElements = map[string]bool{
	"svg": true, "g": true, "path": true, "rect": true, "circle": true, "ellipse": true,
	"line": true, "polyline": true, "polygon": true, "defs": true, "clipPath": true,
	"linearGradient": true, "radialGradient": true, "stop": true, "title": true, "use": true,
}

var svgAttrs = map[string]bool{
	"viewBox": true, "width": true, "height": true, "x": true, "y": true, "x1": true, "y1": true,
	"x2": true, "y2": true, "cx": true, "cy": true, "r": true, "rx": true, "ry": true, "d": true,
	"points": true, "fill": true, "fill-rule": true, "fill-opacity": true, "stroke": true,
	"stroke-width": true, "stroke-linecap": true, "stroke-linejoin": true, "stroke-opacity": true,
	"stroke-dasharray": true, "opacity": true, "transform": true, "id": true, "clip-path": true,
	"clip-rule": true, "offset": true, "stop-color": true, "stop-opacity": true,
	"gradientUnits": true, "gradientTransform": true, "preserveAspectRatio": true, "href": true,
	"style": true,
}

// url(...) nur mit internen Verweisen (#id), keine Skript-Schemata.
var badStyle = regexp.MustCompile(`(?i)url\(\s*['"]?[^#'"\s)]|javascript:|expression\(|@import|behavior`)

// SanitizeSVG schreibt ein SVG mit erlaubten Elementen und Attributen neu.
func SanitizeSVG(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var out bytes.Buffer
	skip := 0 // Tiefe innerhalb verbotener Elemente
	sawRoot := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("svg lesen: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if skip > 0 || !svgElements[t.Name.Local] {
				skip++
				continue
			}
			if !sawRoot {
				if t.Name.Local != "svg" {
					return nil, fmt.Errorf("kein svg-wurzelelement")
				}
				sawRoot = true
			}
			out.WriteString("<" + t.Name.Local)
			if t.Name.Local == "svg" {
				out.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
			}
			for _, a := range t.Attr {
				name := a.Name.Local
				if !svgAttrs[name] || strings.HasPrefix(strings.ToLower(name), "on") {
					continue
				}
				v := a.Value
				if name == "href" && !strings.HasPrefix(v, "#") {
					continue // nur interne Verweise
				}
				if badStyle.MatchString(v) {
					continue
				}
				out.WriteString(" " + name + `="`)
				xml.EscapeText(&out, []byte(v))
				out.WriteString(`"`)
			}
			out.WriteString(">")
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			out.WriteString("</" + t.Name.Local + ">")
		case xml.CharData:
			if skip == 0 {
				xml.EscapeText(&out, t)
			}
		}
	}
	if !sawRoot {
		return nil, fmt.Errorf("kein svg gefunden")
	}
	return out.Bytes(), nil
}

var fileName = regexp.MustCompile(`^[0-9a-f]{32}\.(png|svg)$`)

// ServeHTTP liefert /media/<id>.<png|svg> mit Sicherheits-Headern aus.
func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/media/")
	if !fileName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := "image/png"
	if strings.HasSuffix(name, ".svg") {
		ct = "image/svg+xml"
	}
	h := w.Header()
	h.Set("Content-Type", ct)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Write(data)
}
