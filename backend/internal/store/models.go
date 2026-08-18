package store

// Modelos del dominio. Los campos nullable se mapean a punteros para
// serializarlos como null en JSON (contrato API v1.3).

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	DisplayName  string
	Role         string
	Language     string // auto|es|en
	OCID         string // uuid OpenCloud (identidad canónica del shadow)
	CreatedAt    int64
	LastLoginAt  int64
}

type Folder struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Feed struct {
	ID               int64   `json:"id"`
	URL              string  `json:"url"`
	Title            string  `json:"title"`
	FaviconLink      string  `json:"faviconLink"`
	Added            int64   `json:"added"`
	NextUpdateTime   int64   `json:"nextUpdateTime"`
	FolderID         *int64  `json:"folderId"`
	UnreadCount      int64   `json:"unreadCount"`
	Ordering         int     `json:"ordering"`
	Link             string  `json:"link"`
	Pinned           bool    `json:"pinned"`
	UpdateErrorCount int     `json:"updateErrorCount"`
	LastUpdateError  *string `json:"lastUpdateError"`
	URLHash          string  `json:"urlHash"` // md5(url) para el favicon público
	UserID           int64   `json:"-"`
	NoNewStreak      int     `json:"-"` // interno: scheduler adaptativo
	FullContent      bool    `json:"-"` // el feed ya sirve artículos completos
	RetentionDays    int     `json:"-"` // override de retención por feed; 0 = global
	IsPodcast        bool    `json:"isPodcast"` // feed con enclosures de audio/vídeo
	AuthUser         string  `json:"authUser"`  // usuario Basic del feed ("" = sin auth)
	AuthPassEnc      string  `json:"-"`         // contraseña cifrada (internal/cred); NUNCA sale por API
}

// FeedFilter: keywords que descartan artículos de un feed (News 28.4.0).
// Cada campo es una lista de keywords separadas por coma, matched
// case-insensitive contra title/body/url del item.
type FeedFilter struct {
	FeedID        int64  `json:"feedId"`
	TitleKeywords string `json:"titleKeywords"`
	BodyKeywords  string `json:"bodyKeywords"`
	URLKeywords   string `json:"urlKeywords"`
}

// Matches evalúa si el item debe ser descartado (filtered) por el filtro.
func (f FeedFilter) Matches(it NewItem) bool {
	if f.TitleKeywords != "" && containsAnyFold(it.Title, f.TitleKeywords) {
		return true
	}
	if f.BodyKeywords != "" && containsAnyFold(plain(it.Body), f.BodyKeywords) {
		return true
	}
	if f.URLKeywords != "" && containsAnyFold(it.URL, f.URLKeywords) {
		return true
	}
	return false
}

// HasFilter dice si el filtro tiene alguna keyword configurada.
func (f FeedFilter) HasFilter() bool {
	return f.TitleKeywords != "" || f.BodyKeywords != "" || f.URLKeywords != ""
}

// NewItem: item recibido del fetcher, antes de persistir.
type NewItem struct {
	GUID            string
	GUIDHash        string
	URL             string
	Title           string
	Author          string
	PubDate         int64
	Body            string
	EnclosureMime   *string
	EnclosureLink   *string
	MediaThumbnail  *string
	MediaDescription *string
	Fingerprint     string
	Filtered        bool
}

type Item struct {
	ID              int64   `json:"id"`
	GUID            string  `json:"guid"`
	GUIDHash        string  `json:"guidHash"`
	URL             string  `json:"url"`
	Title           string  `json:"title"`
	Author          string  `json:"author"`
	PubDate         int64   `json:"pubDate"`
	Body            string  `json:"body"`
	EnclosureMime   *string `json:"enclosureMime"`
	EnclosureLink   *string `json:"enclosureLink"`
	MediaThumbnail  *string `json:"mediaThumbnail"`
	MediaDescription *string `json:"mediaDescription"`
	FeedID          int64   `json:"feedId"`
	Unread          bool    `json:"unread"`
	Starred         bool    `json:"starred"`
	Filtered        bool    `json:"filtered"`
	RTL             bool    `json:"rtl"`
	LastModified    int64   `json:"lastModified"`
	Fingerprint     string  `json:"fingerprint"`
	FeedFullContent bool    `json:"feedFullContent"` // el feed ya sirve el artículo entero
}

// ItemFilter define los parámetros de GET /items y /items/updated.
// Type: 0=feed, 1=folder, 2=starred, 3=all (id=0 para starred/all).
type ItemFilter struct {
	UserID      int64
	Type        int
	ID          int64
	GetRead     bool
	BatchSize   int64 // -1 = sin límite
	OffsetID    int64 // cursor por id de item; 0 = sin cursor
	OldestFirst bool
	// IncludeFiltered: si true, devuelve también los items descartados por
	// el filtro de keywords del feed (News 28.4.0: filtered=true).
	IncludeFiltered bool
	// UpdatedSince: si > 0, solo items con last_modified >= valor.
	UpdatedSince int64
}
