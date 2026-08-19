// Package rules: filtros avanzados por regex block/keep en el estilo de
// Miniflux. Cada línea tiene el formato Campo=regex (RE2). Campos soportados:
// EntryTitle, EntryURL, EntryAuthor, EntryContent y EntryDate (con patrones
// de fecha: future, before:, after:, between:, max-age:).
//
// Evaluación (orden de Miniflux, primer match gana):
//  1. Block rules: si alguna casa → el artículo se ignora.
//  2. Keep rules: si hay alguna y ninguna casa → el artículo se ignora.
//
// Global y por feed se componen por separado: primero global block, luego
// feed block, luego global keep, luego feed keep.
package rules

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Field string

const (
	FieldTitle   Field = "EntryTitle"
	FieldURL     Field = "EntryURL"
	FieldAuthor  Field = "EntryAuthor"
	FieldContent Field = "EntryContent"
	FieldDate    Field = "EntryDate"
)

var knownFields = map[Field]bool{
	FieldTitle: true, FieldURL: true, FieldAuthor: true, FieldContent: true, FieldDate: true,
}

// Rule es una regla compilada.
type Rule struct {
	field Field
	re    *regexp.Regexp
	date  datePat
}

type datePat struct {
	kind   string // future|before|after|between|maxage
	t0     int64
	t1     int64
	maxAge time.Duration
}

// RuleSet agrupa las reglas block y keep compiladas.
type RuleSet struct {
	block []Rule
	keep  []Rule
}

// Fields son los campos evaluables de un artículo.
type Fields struct {
	Title   string
	URL     string
	Author  string
	Content string // texto plano del cuerpo
	PubDate int64
}

// Parse compila las reglas de un texto multilínea. Devuelve error con la
// línea problemática si alguna regla es inválida.
func Parse(block, keep string) (*RuleSet, error) {
	b, err := parseLines(block)
	if err != nil {
		return nil, err
	}
	k, err := parseLines(keep)
	if err != nil {
		return nil, err
	}
	return &RuleSet{block: b, keep: k}, nil
}

func parseLines(text string) ([]Rule, error) {
	var out []Rule
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			return nil, fmt.Errorf("regla sin 'Campo=regex': %q", line)
		}
		field := Field(strings.TrimSpace(line[:eq]))
		if !knownFields[field] {
			return nil, fmt.Errorf("campo desconocido %q en %q", field, line)
		}
		pat := strings.TrimSpace(line[eq+1:])
		if pat == "" {
			return nil, fmt.Errorf("regex vacía en %q", line)
		}
		if field == FieldDate {
			d, err := parseDatePat(pat)
			if err != nil {
				return nil, err
			}
			out = append(out, Rule{field: field, date: d})
			continue
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, fmt.Errorf("regex inválida en %q: %w", line, err)
		}
		out = append(out, Rule{field: field, re: re})
	}
	return out, nil
}

// Ignore decide si el artículo debe descartarse según ESTE conjunto de reglas
// (block/keep de un mismo nivel).
func (rs *RuleSet) Ignore(f Fields) bool {
	if rs == nil {
		return false
	}
	for _, r := range rs.block {
		if r.matches(f) {
			return true
		}
	}
	if len(rs.keep) > 0 {
		for _, r := range rs.keep {
			if r.matches(f) {
				return false
			}
		}
		return true
	}
	return false
}

func (r Rule) matches(f Fields) bool {
	switch r.field {
	case FieldTitle:
		return r.re.MatchString(f.Title)
	case FieldURL:
		return r.re.MatchString(f.URL)
	case FieldAuthor:
		return r.re.MatchString(f.Author)
	case FieldContent:
		return r.re.MatchString(f.Content)
	case FieldDate:
		return r.dateMatches(f.PubDate)
	}
	return false
}

func (r Rule) dateMatches(pub int64) bool {
	switch r.date.kind {
	case "future":
		return pub > time.Now().Unix()
	case "before":
		return pub < r.date.t0
	case "after":
		return pub > r.date.t0
	case "between":
		return pub >= r.date.t0 && pub <= r.date.t1
	case "maxage":
		return pub < time.Now().Add(-r.date.maxAge).Unix()
	}
	return false
}

func parseDatePat(pat string) (datePat, error) {
	p := strings.ToLower(strings.TrimSpace(pat))
	switch {
	case p == "future":
		return datePat{kind: "future"}, nil
	case strings.HasPrefix(p, "before:"):
		t, err := parseDate(strings.TrimSpace(p[len("before:"):]))
		return datePat{kind: "before", t0: t}, err
	case strings.HasPrefix(p, "after:"):
		t, err := parseDate(strings.TrimSpace(p[len("after:"):]))
		return datePat{kind: "after", t0: t}, err
	case strings.HasPrefix(p, "between:"):
		parts := strings.Split(strings.TrimSpace(p[len("between:"):]), ",")
		if len(parts) != 2 {
			return datePat{}, fmt.Errorf("between requiere dos fechas: %q", pat)
		}
		t0, err := parseDate(strings.TrimSpace(parts[0]))
		if err != nil {
			return datePat{}, err
		}
		t1, err := parseDate(strings.TrimSpace(parts[1]))
		if err != nil {
			return datePat{}, err
		}
		return datePat{kind: "between", t0: t0, t1: t1}, nil
	case strings.HasPrefix(p, "max-age:"):
		d, err := parseDuration(strings.TrimSpace(p[len("max-age:"):]))
		if err != nil || d <= 0 {
			return datePat{}, fmt.Errorf("max-age inválido %q (ns/µs/ms/s/m/h/d)", pat)
		}
		return datePat{kind: "maxage", maxAge: d}, nil
	}
	return datePat{}, fmt.Errorf("patrón de fecha inválido: %q (future|before:AAAAMMDD|after:|between:,|max-age:)", pat)
}

var durationRe = regexp.MustCompile(`^(\d+)(ns|µs|ms|s|m|h|d)?$`)

// parseDuration: time.ParseDuration no soporta días; los añadimos aquí.
func parseDuration(s string) (time.Duration, error) {
	m := durationRe.FindStringSubmatch(strings.ToLower(strings.TrimSpace(s)))
	if m == nil {
		return 0, fmt.Errorf("duración inválida %q", s)
	}
	n, err := fmtParseInt(m[1])
	if err != nil {
		return 0, err
	}
	unit := m[2]
	if unit == "" {
		unit = "m"
	}
	switch unit {
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}

func fmtParseInt(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("número inválido %q", s)
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

// parseDate convierte YYYY-MM-DD a Unix (medianoche UTC).
func parseDate(s string) (int64, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		return 0, fmt.Errorf("fecha inválida %q (formato YYYY-MM-DD)", s)
	}
	return t.Unix(), nil
}
