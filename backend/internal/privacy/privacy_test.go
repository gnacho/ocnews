package privacy

import "testing"

func TestStripParams(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://x.com/a?utm_source=rss&b=2", "https://x.com/a?b=2"},
		{"https://x.com/a?fbclid=abc&utm_medium=feed&c=3", "https://x.com/a?c=3"},
		{"https://x.com/a?gclid=xyz", "https://x.com/a"},
		{"https://x.com/a?b=2", "https://x.com/a?b=2"},
		{"https://x.com/a", "https://x.com/a"},
		{"https://x.com/a?utm_campaign=uno#frag", "https://x.com/a#frag"},
		{"not a url", "not a url"},
	}
	for _, c := range cases {
		if got := StripParams(c.in); got != c.want {
			t.Errorf("StripParams(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestStripParamsCaseInsensitive(t *testing.T) {
	if got := StripParams("https://x.com/a?UTM_SOURCE=rss"); got != "https://x.com/a" {
		t.Errorf("params en mayúsculas no limpiados: %q", got)
	}
}

func TestRemovePixels(t *testing.T) {
	body := `<p>Hola</p>
<img src="data:image/gif;base64,R0lGODlhAQABAAAAACw=" width="1" height="1" alt="">
<img src="https://site/a.jpg?utm_source=feed" width="400" alt="foto">
<img width="1" height="1" src="https://tracker.example/p.gif">
<p>fin</p>`
	out := RemovePixels(body)
	if contains(out, "R0lGODlh") {
		t.Errorf("pixel data:gif no eliminado: %s", out)
	}
	if contains(out, "tracker.example") {
		t.Errorf("pixel 1x1 no eliminado: %s", out)
	}
	if contains(out, "utm_source") {
		t.Errorf("utm_source no limpiado: %s", out)
	}
	if !contains(out, "site/a.jpg") {
		t.Errorf("imagen real perdida: %s", out)
	}
}

func TestRemovePixelsNoImgs(t *testing.T) {
	if got := RemovePixels("<p>sin img</p>"); got != "<p>sin img</p>" {
		t.Errorf("body sin img alterado: %q", got)
	}
}

func TestIsTrackingPixel(t *testing.T) {
	if !IsTrackingPixel(`<img src="about:blank" width="1" height="1">`) {
		t.Error("about:blank 1x1 no detectado")
	}
	if IsTrackingPixel(`<img src="https://site/a.jpg" width="400" height="300" alt="foto">`) {
		t.Error("imagen real marcada como pixel")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
