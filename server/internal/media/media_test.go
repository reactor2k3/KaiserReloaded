package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestSanitizeSVGRemovesActiveContent(t *testing.T) {
	in := `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 50" onload="alert(1)">
<script>alert(1)</script>
<style>@import url(http://evil)</style>
<foreignObject><div>x</div></foreignObject>
<path d="M0 0H40" fill="red" onclick="x()" style="fill:url(http://evil/a.svg)"/>
<use href="http://evil/x.svg#a"/><use href="#ok"/>
<a href="javascript:alert(1)"><rect width="5" height="5"/></a>
<rect width="10" height="10" fill="url(#grad)"/>
</svg>`
	out, err := SanitizeSVG([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, bad := range []string{"script", "onload", "onclick", "@import", "foreignObject", "http://evil", "javascript", "<a"} {
		if strings.Contains(s, bad) {
			t.Errorf("enthält noch %q:\n%s", bad, s)
		}
	}
	for _, good := range []string{`viewBox="0 0 40 50"`, `d="M0 0H40"`, `href="#ok"`, `fill="url(#grad)"`} {
		if !strings.Contains(s, good) {
			t.Errorf("%q fehlt:\n%s", good, s)
		}
	}
}

func TestSaveRasterAndRejectSVGForAvatar(t *testing.T) {
	st, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	img.Set(10, 10, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	png.Encode(&buf, img)
	name, err := st.Save(buf.Bytes(), false)
	if err != nil || !strings.HasSuffix(name, ".png") {
		t.Fatalf("PNG nicht gespeichert: %v %s", err, name)
	}
	if _, err := st.Save([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), false); err == nil {
		t.Error("SVG als Spielerbild angenommen")
	}
	if name, err := st.Save([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`), true); err != nil || !strings.HasSuffix(name, ".svg") {
		t.Errorf("SVG als Wappen abgelehnt: %v", err)
	}
	if _, err := st.Save([]byte("GIF89a....."), true); err == nil {
		t.Error("GIF angenommen")
	}
}
