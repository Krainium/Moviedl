package main

// moviedl v1 — Global Movie Downloader
//
// Catalog sources (4, all key-free):
//   · iTunes Movie Store — 175 Apple country storefronts across 6 world regions
//   · YTS              — global torrent-catalog with IMDb ratings
//   · Internet Archive — public-domain + freely-hosted films; direct-URL downloads
//   · Cinemeta         — IMDb-backed metadata (director, genre, runtime, rating)
//
// Download engine (yt-dlp):
//   · Internet Archive → direct URL (fastest path)
//   · YouTube search   → 4 query strategies
//   · Google Video     → fallback
//   · Dailymotion      → fallback
//
// Zero API keys. Zero previews. Full films only.
// Author: krainium

import (
        "bufio"
        "encoding/json"
        "flag"
        "fmt"
        "net/http"
        "net/url"
        "os"
        "os/exec"
        "path/filepath"
        "sort"
        "strconv"
        "strings"
        "sync"
        "sync/atomic"
        "time"
)

// ── ANSI colours ─────────────────────────────────────────────────────────────

const (
        Reset   = "\033[0m"
        Bold    = "\033[1m"
        Dim     = "\033[2m"
        Red     = "\033[91m"
        Green   = "\033[92m"
        Yellow  = "\033[93m"
        Blue    = "\033[94m"
        Magenta = "\033[95m"
        Cyan    = "\033[96m"
        White   = "\033[97m"
)

// ── iTunes storefronts — all 175 Apple iTunes countries ───────────────────────

type storefront struct {
        Code   string
        Name   string
        Region string
}

var storefronts = []storefront{
        // ── North America (3) ──────────────────────────────────────────────────
        {"US", "United States", "North America"},
        {"CA", "Canada", "North America"},
        {"MX", "Mexico", "North America"},

        // ── Latin America & Caribbean (35) ────────────────────────────────────
        {"AI", "Anguilla", "Latin America"},
        {"AG", "Antigua & Barbuda", "Latin America"},
        {"AR", "Argentina", "Latin America"},
        {"BS", "Bahamas", "Latin America"},
        {"BB", "Barbados", "Latin America"},
        {"BZ", "Belize", "Latin America"},
        {"BM", "Bermuda", "Latin America"},
        {"BO", "Bolivia", "Latin America"},
        {"BR", "Brazil", "Latin America"},
        {"VG", "British Virgin Islands", "Latin America"},
        {"KY", "Cayman Islands", "Latin America"},
        {"CL", "Chile", "Latin America"},
        {"CO", "Colombia", "Latin America"},
        {"CR", "Costa Rica", "Latin America"},
        {"DM", "Dominica", "Latin America"},
        {"DO", "Dominican Republic", "Latin America"},
        {"EC", "Ecuador", "Latin America"},
        {"SV", "El Salvador", "Latin America"},
        {"GD", "Grenada", "Latin America"},
        {"GT", "Guatemala", "Latin America"},
        {"GY", "Guyana", "Latin America"},
        {"HN", "Honduras", "Latin America"},
        {"JM", "Jamaica", "Latin America"},
        {"MS", "Montserrat", "Latin America"},
        {"NI", "Nicaragua", "Latin America"},
        {"PA", "Panama", "Latin America"},
        {"PY", "Paraguay", "Latin America"},
        {"PE", "Peru", "Latin America"},
        {"KN", "Saint Kitts & Nevis", "Latin America"},
        {"LC", "Saint Lucia", "Latin America"},
        {"VC", "St. Vincent & Grenadines", "Latin America"},
        {"SR", "Suriname", "Latin America"},
        {"TC", "Turks & Caicos", "Latin America"},
        {"TT", "Trinidad & Tobago", "Latin America"},
        {"UY", "Uruguay", "Latin America"},
        {"VE", "Venezuela", "Latin America"},

        // ── Europe (48) ───────────────────────────────────────────────────────
        {"AL", "Albania", "Europe"},
        {"AM", "Armenia", "Europe"},
        {"AT", "Austria", "Europe"},
        {"AZ", "Azerbaijan", "Europe"},
        {"BY", "Belarus", "Europe"},
        {"BE", "Belgium", "Europe"},
        {"BA", "Bosnia & Herzegovina", "Europe"},
        {"BG", "Bulgaria", "Europe"},
        {"HR", "Croatia", "Europe"},
        {"CY", "Cyprus", "Europe"},
        {"CZ", "Czech Republic", "Europe"},
        {"DK", "Denmark", "Europe"},
        {"EE", "Estonia", "Europe"},
        {"FI", "Finland", "Europe"},
        {"FR", "France", "Europe"},
        {"GE", "Georgia", "Europe"},
        {"DE", "Germany", "Europe"},
        {"GR", "Greece", "Europe"},
        {"HU", "Hungary", "Europe"},
        {"IS", "Iceland", "Europe"},
        {"IE", "Ireland", "Europe"},
        {"IT", "Italy", "Europe"},
        {"KZ", "Kazakhstan", "Europe"},
        {"KG", "Kyrgyzstan", "Europe"},
        {"LV", "Latvia", "Europe"},
        {"LT", "Lithuania", "Europe"},
        {"LU", "Luxembourg", "Europe"},
        {"MK", "North Macedonia", "Europe"},
        {"MT", "Malta", "Europe"},
        {"MD", "Moldova", "Europe"},
        {"ME", "Montenegro", "Europe"},
        {"NL", "Netherlands", "Europe"},
        {"NO", "Norway", "Europe"},
        {"PL", "Poland", "Europe"},
        {"PT", "Portugal", "Europe"},
        {"RO", "Romania", "Europe"},
        {"RU", "Russia", "Europe"},
        {"RS", "Serbia", "Europe"},
        {"SK", "Slovakia", "Europe"},
        {"SI", "Slovenia", "Europe"},
        {"ES", "Spain", "Europe"},
        {"SE", "Sweden", "Europe"},
        {"CH", "Switzerland", "Europe"},
        {"TJ", "Tajikistan", "Europe"},
        {"TM", "Turkmenistan", "Europe"},
        {"TR", "Turkey", "Europe"},
        {"UA", "Ukraine", "Europe"},
        {"GB", "United Kingdom", "Europe"},
        {"UZ", "Uzbekistan", "Europe"},

        // ── Middle East (11) ──────────────────────────────────────────────────
        {"BH", "Bahrain", "Middle East"},
        {"IQ", "Iraq", "Middle East"},
        {"IL", "Israel", "Middle East"},
        {"JO", "Jordan", "Middle East"},
        {"KW", "Kuwait", "Middle East"},
        {"LB", "Lebanon", "Middle East"},
        {"OM", "Oman", "Middle East"},
        {"QA", "Qatar", "Middle East"},
        {"SA", "Saudi Arabia", "Middle East"},
        {"AE", "United Arab Emirates", "Middle East"},
        {"YE", "Yemen", "Middle East"},

        // ── Africa (43) ───────────────────────────────────────────────────────
        {"DZ", "Algeria", "Africa"},
        {"AO", "Angola", "Africa"},
        {"BJ", "Benin", "Africa"},
        {"BW", "Botswana", "Africa"},
        {"BF", "Burkina Faso", "Africa"},
        {"CM", "Cameroon", "Africa"},
        {"CV", "Cape Verde", "Africa"},
        {"TD", "Chad", "Africa"},
        {"CI", "Côte d'Ivoire", "Africa"},
        {"CD", "DR Congo", "Africa"},
        {"CG", "Republic of Congo", "Africa"},
        {"DJ", "Djibouti", "Africa"},
        {"EG", "Egypt", "Africa"},
        {"ET", "Ethiopia", "Africa"},
        {"GA", "Gabon", "Africa"},
        {"GM", "Gambia", "Africa"},
        {"GH", "Ghana", "Africa"},
        {"GN", "Guinea", "Africa"},
        {"GW", "Guinea-Bissau", "Africa"},
        {"KE", "Kenya", "Africa"},
        {"LS", "Lesotho", "Africa"},
        {"LR", "Liberia", "Africa"},
        {"LY", "Libya", "Africa"},
        {"MG", "Madagascar", "Africa"},
        {"MW", "Malawi", "Africa"},
        {"ML", "Mali", "Africa"},
        {"MR", "Mauritania", "Africa"},
        {"MU", "Mauritius", "Africa"},
        {"MA", "Morocco", "Africa"},
        {"MZ", "Mozambique", "Africa"},
        {"NA", "Namibia", "Africa"},
        {"NE", "Niger", "Africa"},
        {"NG", "Nigeria", "Africa"},
        {"RW", "Rwanda", "Africa"},
        {"ST", "São Tomé & Príncipe", "Africa"},
        {"SN", "Senegal", "Africa"},
        {"SC", "Seychelles", "Africa"},
        {"SL", "Sierra Leone", "Africa"},
        {"ZA", "South Africa", "Africa"},
        {"SZ", "Eswatini", "Africa"},
        {"TZ", "Tanzania", "Africa"},
        {"TN", "Tunisia", "Africa"},
        {"UG", "Uganda", "Africa"},
        {"ZM", "Zambia", "Africa"},
        {"ZW", "Zimbabwe", "Africa"},

        // ── Asia Pacific (35) ─────────────────────────────────────────────────
        {"AU", "Australia", "Asia Pacific"},
        {"BD", "Bangladesh", "Asia Pacific"},
        {"BT", "Bhutan", "Asia Pacific"},
        {"BN", "Brunei", "Asia Pacific"},
        {"KH", "Cambodia", "Asia Pacific"},
        {"CN", "China", "Asia Pacific"},
        {"FJ", "Fiji", "Asia Pacific"},
        {"FM", "Micronesia", "Asia Pacific"},
        {"HK", "Hong Kong", "Asia Pacific"},
        {"IN", "India", "Asia Pacific"},
        {"ID", "Indonesia", "Asia Pacific"},
        {"JP", "Japan", "Asia Pacific"},
        {"KR", "South Korea", "Asia Pacific"},
        {"LA", "Laos", "Asia Pacific"},
        {"MO", "Macao", "Asia Pacific"},
        {"MY", "Malaysia", "Asia Pacific"},
        {"MV", "Maldives", "Asia Pacific"},
        {"MN", "Mongolia", "Asia Pacific"},
        {"MM", "Myanmar", "Asia Pacific"},
        {"NP", "Nepal", "Asia Pacific"},
        {"NZ", "New Zealand", "Asia Pacific"},
        {"PG", "Papua New Guinea", "Asia Pacific"},
        {"PH", "Philippines", "Asia Pacific"},
        {"PK", "Pakistan", "Asia Pacific"},
        {"PW", "Palau", "Asia Pacific"},
        {"SG", "Singapore", "Asia Pacific"},
        {"LK", "Sri Lanka", "Asia Pacific"},
        {"TW", "Taiwan", "Asia Pacific"},
        {"TH", "Thailand", "Asia Pacific"},
        {"TL", "Timor-Leste", "Asia Pacific"},
        {"TO", "Tonga", "Asia Pacific"},
        {"VU", "Vanuatu", "Asia Pacific"},
        {"VN", "Vietnam", "Asia Pacific"},
        {"WS", "Samoa", "Asia Pacific"},
        {"PF", "French Polynesia", "Asia Pacific"},
}

// ── Movie data model ──────────────────────────────────────────────────────────

type Movie struct {
        ID        string
        Title     string
        Director  string
        Year      string
        Genre     string
        Rating    string // IMDb rating or content advisory
        Runtime   int    // seconds
        Source    string
        IMDbID    string // non-empty → used for smart dedup and enriched downloads
        DirectURL string // non-empty → skip YouTube search, use this URL with yt-dlp
}

// titleYearKey is used for deduplication when no IMDb ID is available.
func titleYearKey(m Movie) string {
        norm := func(s string) string {
                return strings.ToLower(strings.Join(strings.Fields(s), " "))
        }
        return norm(m.Title) + "|||" + norm(m.Year)
}

// dedup merges results intelligently:
//  1. Movies with IMDb IDs (Cinemeta, YTS) deduplicated by IMDb ID — highest metadata quality
//  2. Movies without IMDb IDs (iTunes, Archive.org) deduplicated by title+year,
//     also suppressed when an IMDb-identified movie already covers the same title+year
func dedup(raw []Movie) []Movie {
        seenImdb := make(map[string]bool)
        seenTY := make(map[string]bool)
        var out []Movie

        // Pass 1: IMDb-identified movies (prefer these — richer metadata)
        for _, m := range raw {
                if m.IMDbID == "" {
                        continue
                }
                k := strings.ToLower(m.IMDbID)
                if seenImdb[k] {
                        continue
                }
                seenImdb[k] = true
                seenTY[titleYearKey(m)] = true // also reserve this title+year slot
                out = append(out, m)
        }

        // Pass 2: non-IMDb movies not already covered by an IMDb match
        for _, m := range raw {
                if m.IMDbID != "" {
                        continue
                }
                k := titleYearKey(m)
                if seenTY[k] {
                        continue
                }
                seenTY[k] = true
                out = append(out, m)
        }
        return out
}

// ── Print helpers ─────────────────────────────────────────────────────────────

var stdin = bufio.NewReader(os.Stdin)

func oinfo(msg string)    { fmt.Printf("  %s%s[*]%s %s\n", Bold, Cyan, Reset, msg) }
func osuccess(msg string) { fmt.Printf("  %s%s[+]%s %s%s%s\n", Bold, Green, Reset, Green, msg, Reset) }
func owarn(msg string)    { fmt.Printf("  %s%s[!]%s %s%s%s\n", Bold, Yellow, Reset, Yellow, msg, Reset) }
func oerror(msg string)   { fmt.Printf("  %s%s[-]%s %s%s%s\n", Bold, Red, Reset, Red, msg, Reset) }
func odivider()           { fmt.Printf("  %s%s%s\n", Dim, strings.Repeat("─", 66), Reset) }

func oheader(title string) {
        fmt.Println()
        odivider()
        fmt.Printf("  %s%s%s\n", Bold+White, title, Reset)
        odivider()
        fmt.Println()
}

func obanner() {
        b := Cyan
        y := Bold + Yellow
        r := Reset
        fmt.Println()
        fmt.Printf("  %s+══════════════════════════════════════════════════════════+%s\n", b, r)
        fmt.Printf("  %s|%s  %s███╗   ███╗ ██████╗ ██╗   ██╗██╗███████╗██████╗ ██╗     %s%s|%s\n", b, r, y, r, b, r)
        fmt.Printf("  %s|%s  %s████╗ ████║██╔═══██╗██║   ██║██║██╔════╝██╔══██╗██║     %s%s|%s\n", b, r, y, r, b, r)
        fmt.Printf("  %s|%s  %s██╔████╔██║██║   ██║██║   ██║██║█████╗  ██║  ██║██║     %s%s|%s\n", b, r, y, r, b, r)
        fmt.Printf("  %s|%s  %s██║╚██╔╝██║██║   ██║╚██╗ ██╔╝██║██╔══╝  ██║  ██║██║     %s%s|%s\n", b, r, y, r, b, r)
        fmt.Printf("  %s|%s  %s██║ ╚═╝ ██║╚██████╔╝ ╚████╔╝ ██║███████╗██████╔╝███████╗%s%s|%s\n", b, r, y, r, b, r)
        fmt.Printf("  %s|%s  %s╚═╝     ╚═╝ ╚═════╝   ╚═══╝  ╚═╝╚══════╝╚═════╝ ╚══════╝%s%s|%s\n", b, r, y, r, b, r)
        fmt.Printf("  %s|%s                                                            %s|%s\n", b, r, b, r)
        fmt.Printf("  %s|%s  %sGlobal Movie Downloader  v1.0%s  %sgithub.com/krainium%s       %s|%s\n", b, r, White, r, Dim, r, b, r)
        fmt.Printf("  %s+══════════════════════════════════════════════════════════+%s\n", b, r)
        odivider()
        fmt.Println()
}

func prompt(msg string) string {
        fmt.Printf("  %s%s[>]%s %s%s%s: ", Bold, Magenta, Reset, White, msg, Reset)
        line, _ := stdin.ReadString('\n')
        return strings.TrimSpace(line)
}

// ── HTTP helper ───────────────────────────────────────────────────────────────

var httpClient = &http.Client{
        Timeout: 15 * time.Second,
        Transport: &http.Transport{
                MaxIdleConnsPerHost: 30,
                DisableCompression:  false,
        },
}

func httpGet(rawURL string) (*http.Response, error) {
        req, err := http.NewRequest("GET", rawURL, nil)
        if err != nil {
                return nil, err
        }
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0")
        req.Header.Set("Accept", "application/json, */*")
        req.Header.Set("Accept-Language", "en-US,en;q=0.9")
        return httpClient.Do(req)
}

// ── iTunes Movie API ──────────────────────────────────────────────────────────

type itunesMovieResp struct {
        ResultCount int `json:"resultCount"`
        Results     []struct {
                Kind                  string `json:"kind"`
                TrackId               int    `json:"trackId"`
                TrackName             string `json:"trackName"`
                ArtistName            string `json:"artistName"`
                TrackTimeMillis       int    `json:"trackTimeMillis"`
                ReleaseDate           string `json:"releaseDate"`
                PrimaryGenreName      string `json:"primaryGenreName"`
                ContentAdvisoryRating string `json:"contentAdvisoryRating"`
        } `json:"results"`
}

func searchItunesMovies(query, country string) []Movie {
        params := url.Values{}
        params.Set("term", query)
        params.Set("media", "movie")
        params.Set("entity", "movie")
        params.Set("limit", "200")
        params.Set("country", country)
        params.Set("lang", "en_us")

        resp, err := httpGet("https://itunes.apple.com/search?" + params.Encode())
        if err != nil {
                return nil
        }
        defer resp.Body.Close()

        var data itunesMovieResp
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
                return nil
        }

        var movies []Movie
        for _, r := range data.Results {
                if r.Kind != "feature-movie" || r.TrackName == "" {
                        continue
                }
                year := ""
                if len(r.ReleaseDate) >= 4 {
                        year = r.ReleaseDate[:4]
                }
                movies = append(movies, Movie{
                        ID:       strconv.Itoa(r.TrackId),
                        Title:    r.TrackName,
                        Director: r.ArtistName,
                        Year:     year,
                        Genre:    r.PrimaryGenreName,
                        Rating:   r.ContentAdvisoryRating,
                        Runtime:  r.TrackTimeMillis / 1000,
                        Source:   "iTunes/" + country,
                })
        }
        return movies
}

// ── YTS Movie API ─────────────────────────────────────────────────────────────

type ytsResp struct {
        Status string `json:"status"`
        Data   struct {
                Movies []struct {
                        ID       int      `json:"id"`
                        Title    string   `json:"title"`
                        Year     int      `json:"year"`
                        Rating   float64  `json:"rating"`
                        Runtime  int      `json:"runtime"` // minutes
                        Genres   []string `json:"genres"`
                        IMDbCode string   `json:"imdb_code"`
                } `json:"movies"`
        } `json:"data"`
}

func searchYTS(query string) []Movie {
        apiURL := "https://yts.mx/api/v2/list_movies.json?query_term=" +
                url.QueryEscape(query) + "&limit=50&sort_by=download_count"

        resp, err := httpGet(apiURL)
        if err != nil {
                return nil
        }
        defer resp.Body.Close()

        var data ytsResp
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
                return nil
        }
        if data.Status != "ok" {
                return nil
        }

        var movies []Movie
        for _, m := range data.Data.Movies {
                if m.Title == "" {
                        continue
                }
                genre := ""
                if len(m.Genres) > 0 {
                        genre = strings.Join(m.Genres[:minInt(2, len(m.Genres))], "/")
                }
                movies = append(movies, Movie{
                        ID:      strconv.Itoa(m.ID),
                        Title:   m.Title,
                        Year:    strconv.Itoa(m.Year),
                        Genre:   genre,
                        Rating:  fmt.Sprintf("%.1f", m.Rating),
                        Runtime: m.Runtime * 60,
                        IMDbID:  m.IMDbCode,
                        Source:  "yts",
                })
        }
        return movies
}

// ── Internet Archive Movie Search ─────────────────────────────────────────────
// Free public API, zero auth required, fully accessible worldwide.
// yt-dlp downloads directly from archive.org/details/<id>.

type iabResp struct {
        Response struct {
                NumFound int `json:"numFound"`
                Docs     []struct {
                        Identifier string      `json:"identifier"`
                        Title      interface{} `json:"title"`
                        Year       interface{} `json:"year"`
                        Creator    interface{} `json:"creator"`
                        Runtime    interface{} `json:"runtime"`
                        Subject    interface{} `json:"subject"`
                } `json:"docs"`
        } `json:"response"`
}

func iabString(v interface{}) string {
        if v == nil {
                return ""
        }
        switch s := v.(type) {
        case string:
                return s
        case []interface{}:
                if len(s) > 0 {
                        if str, ok := s[0].(string); ok {
                                return str
                        }
                }
        }
        return fmt.Sprintf("%v", v)
}

// parseIABRuntime converts archive.org runtime strings to seconds.
// Handles formats: "1:32:14", "92 min", "92", "5520" (already seconds).
func parseIABRuntime(raw string) int {
        raw = strings.TrimSpace(raw)
        if raw == "" {
                return 0
        }
        raw = strings.ToLower(strings.ReplaceAll(raw, "min", ""))
        raw = strings.TrimSpace(raw)
        if strings.Contains(raw, ":") {
                parts := strings.Split(raw, ":")
                total := 0
                for _, p := range parts {
                        n, _ := strconv.Atoi(strings.TrimSpace(p))
                        total = total*60 + n
                }
                return total
        }
        n, err := strconv.Atoi(raw)
        if err != nil {
                return 0
        }
        if n < 600 { // treat as minutes
                return n * 60
        }
        return n
}

// iabNonFeatureKeywords are title substrings that reliably indicate non-feature
// content: trailers, TV spots, clips, promos, theme songs, intros, etc.
var iabNonFeatureKeywords = []string{
        // Production promotional material
        "trailer", "teaser", "tv spot", "tv-spot", "featurette",
        "making of", "making-of", "behind the scenes", "deleted scene",
        "blooper", "promo", "promotional", "commercial", "preview",
        "sneak peek", "official clip", " clip)", " clip ", "(clip",
        "epk material", "epk)",
        "bande-annonce", "bande annonce",
        // Social media / theatrical marketing copy
        "now playing", "get tickets", "in theaters", "only in theaters",
        // Theme songs and musical content (not films)
        "theme song", "theme music", "opening theme",
        // Audio dubs / radio (not films)
        "dual audio", "dual áudio", "dublado",
}

// iabIsNonFeature returns true when an archive.org title or known runtime
// strongly suggest it is NOT a feature film (trailer, clip, promo, etc.).
func iabIsNonFeature(title string, runtimeSecs int) bool {
        lower := strings.ToLower(title)
        for _, kw := range iabNonFeatureKeywords {
                if strings.Contains(lower, kw) {
                        return true
                }
        }
        // When runtime is explicitly known and under 40 minutes it can't be a feature film.
        if runtimeSecs > 0 && runtimeSecs < 40*60 {
                return true
        }
        return false
}

func iabFetch(q string, rows int) []Movie {
        params := url.Values{}
        params.Set("q", q)
        params.Set("fl[]", "identifier,title,year,creator,runtime,subject")
        params.Set("sort[]", "downloads desc")
        params.Set("output", "json")
        params.Set("rows", strconv.Itoa(rows))

        resp, err := httpGet("https://archive.org/advancedsearch.php?" + params.Encode())
        if err != nil {
                return nil
        }
        defer resp.Body.Close()

        var data iabResp
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
                return nil
        }

        var movies []Movie
        for _, doc := range data.Response.Docs {
                if doc.Identifier == "" {
                        continue
                }
                title := iabString(doc.Title)
                if title == "" {
                        title = doc.Identifier
                }
                year := iabString(doc.Year)
                if len(year) > 4 {
                        year = year[:4]
                }
                director := iabString(doc.Creator)
                genre := iabString(doc.Subject)
                if idx := strings.Index(genre, ";"); idx > 0 {
                        genre = strings.TrimSpace(genre[:idx])
                }
                if len([]rune(genre)) > 20 {
                        genre = string([]rune(genre)[:20])
                }
                runtime := parseIABRuntime(iabString(doc.Runtime))

                // Skip trailers, clips, promos, and known sub-feature runtimes.
                if iabIsNonFeature(title, runtime) {
                        continue
                }

                movies = append(movies, Movie{
                        ID:        doc.Identifier,
                        Title:     title,
                        Director:  director,
                        Year:      year,
                        Genre:     genre,
                        Runtime:   runtime,
                        Source:    "archive.org",
                        DirectURL: "https://archive.org/details/" + doc.Identifier,
                })
        }
        return movies
}

func searchInternetArchive(query string) []Movie {
        // Search ONLY the archive.org feature_films collection — a curated vault of
        // full-length films (public domain and freely hosted). This eliminates trailers,
        // clips, theme songs, social-media dumps, and other non-feature content entirely.
        // Title-only matching prevents unrelated films from sneaking in via keyword hits.
        titleQuery := fmt.Sprintf(`title:("%s") AND collection:feature_films`, query)
        return iabFetch(titleQuery, 50)
}

// ── Cinemeta (Stremio) ────────────────────────────────────────────────────────
// IMDb-backed movie catalog served by the Stremio project.
// Completely free, no API key, accessible worldwide.
// Two-step: catalog search (gets IMDb IDs) → meta enrichment (director, genres, rating).

type cinemataSearchResp struct {
        Metas []struct {
                ID          string `json:"id"`
                ImdbID      string `json:"imdb_id"`
                Name        string `json:"name"`
                ReleaseInfo string `json:"releaseInfo"`
        } `json:"metas"`
}

type cinemataMetaResp struct {
        Meta struct {
                ID         string      `json:"id"`
                Name       string      `json:"name"`
                Year       interface{} `json:"year"`
                Released   string      `json:"released"`
                Runtime    string      `json:"runtime"`
                Genres     []string    `json:"genres"`
                ImdbRating interface{} `json:"imdbRating"`
                Director   interface{} `json:"director"`
        } `json:"meta"`
}

// parseCinemetaRuntime converts "152 min" → seconds.
func parseCinemetaRuntime(raw string) int {
        raw = strings.TrimSpace(strings.ToLower(raw))
        raw = strings.ReplaceAll(raw, "min", "")
        n, _ := strconv.Atoi(strings.TrimSpace(raw))
        return n * 60
}

func cinemetaYearStr(v interface{}) string {
        if v == nil {
                return ""
        }
        switch y := v.(type) {
        case string:
                if len(y) > 4 {
                        return y[:4]
                }
                return y
        case float64:
                return strconv.Itoa(int(y))
        }
        s := fmt.Sprintf("%v", v)
        if len(s) > 4 {
                return s[:4]
        }
        return s
}

func cinemetaRatingStr(v interface{}) string {
        if v == nil {
                return ""
        }
        switch r := v.(type) {
        case string:
                return r
        case float64:
                return fmt.Sprintf("%.1f", r)
        }
        return fmt.Sprintf("%v", v)
}

func cinemetaDirectors(v interface{}) string {
        if v == nil {
                return ""
        }
        switch d := v.(type) {
        case string:
                return d
        case []interface{}:
                var names []string
                for _, item := range d {
                        if s, ok := item.(string); ok {
                                names = append(names, s)
                        }
                }
                if len(names) > 2 {
                        names = names[:2]
                }
                return strings.Join(names, ", ")
        }
        return ""
}

func cinemetaEnrichMeta(imdbID string) (Movie, bool) {
        client := &http.Client{Timeout: 6 * time.Second}
        req, err := http.NewRequest("GET", "https://v3-cinemeta.strem.io/meta/movie/"+imdbID+".json", nil)
        if err != nil {
                return Movie{}, false
        }
        req.Header.Set("User-Agent", "Mozilla/5.0")
        req.Header.Set("Accept", "application/json")

        resp, err := client.Do(req)
        if err != nil {
                return Movie{}, false
        }
        defer resp.Body.Close()

        var data cinemataMetaResp
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
                return Movie{}, false
        }

        m := data.Meta
        if m.Name == "" {
                return Movie{}, false
        }

        year := cinemetaYearStr(m.Year)
        if year == "" && len(m.Released) >= 4 {
                year = m.Released[:4]
        }

        genre := ""
        if len(m.Genres) > 0 {
                g := m.Genres
                if len(g) > 2 {
                        g = g[:2]
                }
                genre = strings.Join(g, "/")
        }

        return Movie{
                Title:    m.Name,
                Year:     year,
                Director: cinemetaDirectors(m.Director),
                Genre:    genre,
                Rating:   cinemetaRatingStr(m.ImdbRating),
                Runtime:  parseCinemetaRuntime(m.Runtime),
                IMDbID:   imdbID,
                Source:   "cinemeta",
        }, true
}

func searchCinemeta(query string) []Movie {
        searchURL := "https://v3-cinemeta.strem.io/catalog/movie/top/search=" +
                url.QueryEscape(query) + ".json"

        resp, err := httpGet(searchURL)
        if err != nil {
                return nil
        }
        defer resp.Body.Close()

        var data cinemataSearchResp
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
                return nil
        }
        if len(data.Metas) == 0 {
                return nil
        }

        // Enrich each result in parallel (max 5 concurrent meta calls).
        type enriched struct {
                m  Movie
                ok bool
        }
        sem := make(chan struct{}, 5)
        results := make(chan enriched, len(data.Metas))

        goroutines := 0
        for _, meta := range data.Metas {
                id := meta.ImdbID
                if id == "" {
                        id = meta.ID
                }
                if id == "" {
                        continue
                }
                goroutines++
                sem <- struct{}{}
                go func(imdbID, name, releaseInfo string) {
                        defer func() { <-sem }()
                        m, ok := cinemetaEnrichMeta(imdbID)
                        if !ok {
                                // Fall back to basic catalog data
                                year := releaseInfo
                                if len(year) > 4 {
                                        year = year[:4]
                                }
                                m = Movie{Title: name, Year: year, IMDbID: imdbID, Source: "cinemeta"}
                                ok = name != ""
                        }
                        results <- enriched{m, ok}
                }(id, meta.Name, meta.ReleaseInfo)
        }

        var movies []Movie
        for i := 0; i < goroutines; i++ {
                e := <-results
                if e.ok && e.m.Title != "" {
                        movies = append(movies, e.m)
                }
        }
        return movies
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func minInt(a, b int) int {
        if a < b {
                return a
        }
        return b
}

// ── Multi-source parallel search ──────────────────────────────────────────────

func searchAll(query string) []Movie {
        type result struct{ movies []Movie }

        // 175 iTunes storefronts + YTS + Internet Archive + Cinemeta
        total := len(storefronts) + 3
        ch := make(chan result, total)

        // iTunes: one goroutine per storefront (staggered to avoid rate limiting)
        for i, sf := range storefronts {
                sf := sf
                delay := time.Duration(i/20) * 200 * time.Millisecond
                go func() {
                        time.Sleep(delay)
                        ch <- result{searchItunesMovies(query, sf.Code)}
                }()
        }

        // YTS — global catalog with IMDb codes
        go func() { ch <- result{searchYTS(query)} }()

        // Internet Archive — public-domain and freely hosted films
        go func() { ch <- result{searchInternetArchive(query)} }()

        // Cinemeta — IMDb-backed metadata, enriched results
        go func() { ch <- result{searchCinemeta(query)} }()

        // Collect all raw results
        var raw []Movie
        for i := 0; i < total; i++ {
                r := <-ch
                raw = append(raw, r.movies...)
        }

        // Smart dedup: IMDb ID first, then title+year
        all := dedup(raw)

        // Sort: year desc → IMDb-identified first (cinemeta/yts) → title asc
        sort.Slice(all, func(i, j int) bool {
                if all[i].Year != all[j].Year {
                        return all[i].Year > all[j].Year
                }
                iHasID := all[i].IMDbID != ""
                jHasID := all[j].IMDbID != ""
                if iHasID != jHasID {
                        return iHasID // IMDb-enriched results float to the top
                }
                return strings.ToLower(all[i].Title) < strings.ToLower(all[j].Title)
        })

        return all
}

// ── Table display ─────────────────────────────────────────────────────────────

func fmtRuntime(secs int) string {
        if secs <= 0 {
                return "?:??"
        }
        h := secs / 3600
        m := (secs % 3600) / 60
        if h > 0 {
                return fmt.Sprintf("%dh%02dm", h, m)
        }
        return fmt.Sprintf("%dm", m)
}

func truncStr(s string, n int) string {
        runes := []rune(s)
        if len(runes) <= n {
                return s + strings.Repeat(" ", n-len(runes))
        }
        return string(runes[:n-1]) + "…"
}

func printTable(movies []Movie) {
        fmt.Printf("  %s%-4s  %-7s  %-38s  %-22s  %-4s  %-6s  %s%s\n",
                Bold+White, "#", "Runtime", "Title", "Director", "Year", "Rating", "Source", Reset)
        odivider()
        for i, m := range movies {
                row := i + 1
                col := Cyan
                if row%2 == 0 {
                        col = Blue
                }
                rating := m.Rating
                if rating == "" || rating == "0.0" {
                        rating = "----"
                }
                director := m.Director
                if director == "" {
                        director = "Unknown"
                }
                fmt.Printf("  %s%-4d%s  %s%-7s%s  %s%-38s%s  %s%-22s%s  %s%-4s%s  %s%-6s%s  %s%s%s\n",
                        col, row, Reset,
                        Dim, fmtRuntime(m.Runtime), Reset,
                        White, truncStr(m.Title, 38), Reset,
                        Dim, truncStr(director, 22), Reset,
                        Yellow, truncStr(m.Year, 4), Reset,
                        Dim, truncStr(rating, 6), Reset,
                        Dim, m.Source, Reset,
                )
        }
}

// ── Selection parser ──────────────────────────────────────────────────────────

func parseSelection(input string, max int) []int {
        if strings.TrimSpace(input) == "" {
                all := make([]int, max)
                for i := range all {
                        all[i] = i + 1
                }
                return all
        }
        seen := make(map[int]bool)
        var out []int
        for _, part := range strings.Split(input, ",") {
                part = strings.TrimSpace(part)
                if strings.Contains(part, "-") {
                        bounds := strings.SplitN(part, "-", 2)
                        lo, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
                        hi, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
                        if err1 != nil || err2 != nil || lo < 1 || hi > max || lo > hi {
                                continue
                        }
                        for n := lo; n <= hi; n++ {
                                if !seen[n] {
                                        seen[n] = true
                                        out = append(out, n)
                                }
                        }
                } else {
                        n, err := strconv.Atoi(part)
                        if err != nil || n < 1 || n > max {
                                continue
                        }
                        if !seen[n] {
                                seen[n] = true
                                out = append(out, n)
                        }
                }
        }
        sort.Ints(out)
        return out
}

// ── yt-dlp management ─────────────────────────────────────────────────────────

func ytdlpCandidates() []string {
        return []string{
                "yt-dlp",
                os.ExpandEnv("$HOME/.local/bin/yt-dlp"),
                "/home/runner/workspace/.pythonlibs/bin/yt-dlp",
                "/usr/local/bin/yt-dlp",
                "/usr/bin/yt-dlp",
        }
}

func findYtdlp() string {
        for _, p := range ytdlpCandidates() {
                if path, err := exec.LookPath(p); err == nil {
                        return path
                }
        }
        return ""
}

func installYtdlp() string {
        owarn("yt-dlp not found — installing automatically (one-time setup)...")
        pips := []string{"pip", "pip3", "python3 -m pip", "python -m pip"}
        anyPipFound := false
        for _, pip := range pips {
                parts := strings.Fields(pip)
                if _, err := exec.LookPath(parts[0]); err != nil {
                        continue
                }
                anyPipFound = true
                args := append(parts[1:], "install", "--quiet", "yt-dlp")
                cmd := exec.Command(parts[0], args...)
                cmd.Stdout = os.Stdout
                cmd.Stderr = os.Stderr
                if err := cmd.Run(); err == nil {
                        osuccess("yt-dlp installed successfully")
                        return findYtdlp()
                }
                owarn(fmt.Sprintf("%s install failed — trying next installer...", parts[0]))
        }
        if !anyPipFound {
                oerror("pip / pip3 not found on this system.")
                oerror("Install Python first, then run: pip install yt-dlp")
        }
        return ""
}

func checkAndUpdateYtdlp(ytdlpPath string, ch chan<- string) {
        home, err := os.UserHomeDir()
        if err != nil {
                ch <- ""
                return
        }
        cacheDir := filepath.Join(home, ".moviedl")
        _ = os.MkdirAll(cacheDir, 0755)
        stampFile := filepath.Join(cacheDir, ".last_update_check")

        const week = 7 * 24 * time.Hour
        if data, err := os.ReadFile(stampFile); err == nil {
                if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); err == nil {
                        if time.Since(ts) < week {
                                ch <- ""
                                return
                        }
                }
        }

        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Get("https://pypi.org/pypi/yt-dlp/json")
        if err != nil {
                ch <- ""
                return
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                ch <- ""
                return
        }

        var pypi struct {
                Info struct {
                        Version string `json:"version"`
                } `json:"info"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&pypi); err != nil {
                ch <- ""
                return
        }
        latest := strings.TrimSpace(pypi.Info.Version)
        if latest == "" {
                ch <- ""
                return
        }

        out, err := exec.Command(ytdlpPath, "--version").Output()
        if err != nil {
                ch <- ""
                return
        }
        installed := strings.TrimSpace(string(out))
        _ = os.WriteFile(stampFile, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)

        if installed == latest {
                ch <- fmt.Sprintf("yt-dlp is up to date (%s)", installed)
                return
        }

        pips := []string{"pip", "pip3", "python3 -m pip", "python -m pip"}
        for _, pip := range pips {
                parts := strings.Fields(pip)
                if _, err := exec.LookPath(parts[0]); err != nil {
                        continue
                }
                args := append(parts[1:], "install", "--quiet", "--upgrade", "yt-dlp")
                cmd := exec.Command(parts[0], args...)
                if err := cmd.Run(); err == nil {
                        ch <- fmt.Sprintf("yt-dlp updated to %s", latest)
                        return
                }
        }
        ch <- ""
}

func tryUpgradeYtdlp(ytdlpPath string) (string, string, bool) {
        client := &http.Client{Timeout: 10 * time.Second}
        resp, err := client.Get("https://pypi.org/pypi/yt-dlp/json")
        if err != nil {
                return "", "", false
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return "", "", false
        }

        var pypi struct {
                Info struct {
                        Version string `json:"version"`
                } `json:"info"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&pypi); err != nil {
                return "", "", false
        }
        latest := strings.TrimSpace(pypi.Info.Version)
        if latest == "" {
                return "", "", false
        }

        out, err := exec.Command(ytdlpPath, "--version").Output()
        if err != nil {
                return "", "", false
        }
        installed := strings.TrimSpace(string(out))
        if installed == latest {
                return "", "", false
        }

        pips := []string{"pip", "pip3", "python3 -m pip", "python -m pip"}
        for _, pip := range pips {
                parts := strings.Fields(pip)
                if _, err := exec.LookPath(parts[0]); err != nil {
                        continue
                }
                args := append(parts[1:], "install", "--quiet", "--upgrade", "yt-dlp")
                cmd := exec.Command(parts[0], args...)
                if err := cmd.Run(); err == nil {
                        if home, err := os.UserHomeDir(); err == nil {
                                cacheDir := filepath.Join(home, ".moviedl")
                                _ = os.MkdirAll(cacheDir, 0755)
                                stampFile := filepath.Join(cacheDir, ".last_update_check")
                                _ = os.WriteFile(stampFile, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
                        }
                        if newPath := findYtdlp(); newPath != "" {
                                return newPath, latest, true
                        }
                        return ytdlpPath, latest, true
                }
        }
        return "", "", false
}

// ── Downloader ────────────────────────────────────────────────────────────────

var (
        printMu sync.Mutex
        dlDone  int64
        dlFail  int64
        dlSkip  int64
)

type upgradeSession struct {
        once       sync.Once
        noticeOnce sync.Once
        newPath    string
        newVersion string
        upgraded   bool
}

func (s *upgradeSession) tryOnce(ytdlp string) (string, string, bool) {
        s.once.Do(func() {
                s.newPath, s.newVersion, s.upgraded = tryUpgradeYtdlp(ytdlp)
        })
        return s.newPath, s.newVersion, s.upgraded
}

func (s *upgradeSession) printUpgradeNotice() {
        s.noticeOnce.Do(func() {
                printMu.Lock()
                defer printMu.Unlock()
                fmt.Printf("\n  %s%s[+]%s %s%syt-dlp upgraded to %s — retrying failed downloads%s\n\n",
                        Bold, Green, Reset, Bold, Green, s.newVersion, Reset)
        })
}

func safeName(s string) string {
        var b strings.Builder
        for _, r := range s {
                switch r {
                case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\x00':
                        b.WriteByte('_')
                default:
                        b.WriteRune(r)
                }
        }
        result := strings.TrimSpace(b.String())
        if len(result) > 180 {
                result = result[:180]
        }
        return result
}

func printResult(idx int, m Movie, status, extra string) {
        printMu.Lock()
        defer printMu.Unlock()
        var col, sym string
        switch status {
        case "done":
                col, sym = Green, "✓"
        case "skip":
                col, sym = Cyan, "→"
        case "fail":
                col, sym = Red, "✗"
        }
        fmt.Printf("  %s%s[%s]%s  %s%3d%s  %-38s  %s%-7s%s  %s%s%s\n",
                Bold, col, sym, Reset,
                Cyan, idx, Reset,
                truncStr(m.Title, 38),
                Dim, fmtRuntime(m.Runtime), Reset,
                Dim, extra, Reset)
}

// ── Download engine ───────────────────────────────────────────────────────────
//
// Download strategy for each movie:
//   0. (archive.org results) Direct URL — yt-dlp extracts best quality natively
//   1. ytsearch10:"TITLE YEAR" full movie   (exact quoted title, up to 10 candidates)
//   2. ytsearch10:TITLE YEAR full movie
//   3. ytsearch10:TITLE YEAR full film
//   4. ytsearch10:TITLE full movie free
//   5. gvsearch10:TITLE YEAR full movie      (Google Video)
//   6. dailymotionsearch10:TITLE YEAR full movie  (Dailymotion)
//   7. ytsearch10:TITLE full movie HD
//
// Each search query pulls up to 10 results; --match-filter duration>=3300
// skips trailers and clips (must be at least 55 min to qualify).
// yt-dlp format: bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best[ext=mp4]/best → mp4

func ytdlpFileCreated(outBase string) bool {
        for _, ext := range []string{"mp4", "mkv", "webm", "avi"} {
                if _, err := os.Stat(outBase + "." + ext); err == nil {
                        return true
                }
        }
        return false
}

// runYtdlpDirect downloads from a known direct URL (e.g. archive.org) without
// any duration filter — the URL already points to a feature film.
func runYtdlpDirect(ytdlp, outBase, url string) bool {
        cmd := exec.Command(ytdlp,
                "--no-playlist",
                "--format", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best[ext=mp4]/best",
                "--merge-output-format", "mp4",
                "--output", outBase+".%(ext)s",
                "--no-progress",
                "--quiet",
                "--no-warnings",
                url,
        )
        cmd.Run()
        return ytdlpFileCreated(outBase)
}

// runYtdlpSearch runs a yt-dlp search query and enforces a minimum duration so
// we never download trailers or clips. It searches up to 10 results and picks
// the first one that is at least 55 minutes long.
func runYtdlpSearch(ytdlp, outBase, query string) bool {
        cmd := exec.Command(ytdlp,
                "--no-playlist",
                "--format", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best[ext=mp4]/best",
                "--merge-output-format", "mp4",
                "--output", outBase+".%(ext)s",
                "--no-progress",
                "--quiet",
                "--no-warnings",
                // Require at least 55 minutes — clips and trailers are always shorter.
                "--match-filter", "duration>=3300",
                query,
        )
        cmd.Run()
        return ytdlpFileCreated(outBase)
}

func downloadMovie(ytdlp string, m Movie, outBase string) error {
        // Strategy 0: Direct URL (Internet Archive — fastest, no search needed)
        if m.DirectURL != "" {
                if runYtdlpDirect(ytdlp, outBase, m.DirectURL) {
                        return nil
                }
        }

        titleYear := m.Title
        if m.Year != "" && m.Year != "----" {
                titleYear = m.Title + " " + m.Year
        }

        // Search up to 10 YouTube results per query so the duration filter has
        // enough candidates to find a full-length upload and skip any clips/trailers.
        queries := []string{
                // YouTube — primary source (4 strategies)
                fmt.Sprintf(`ytsearch10:"%s" full movie`, titleYear),
                fmt.Sprintf("ytsearch10:%s full movie", titleYear),
                fmt.Sprintf("ytsearch10:%s full film", titleYear),
                fmt.Sprintf("ytsearch10:%s full movie free", m.Title),
                // Google Video — secondary (crawls many hosting sites)
                fmt.Sprintf("gvsearch10:%s full movie", titleYear),
                // Dailymotion — tertiary (hosts indie and public-domain films)
                fmt.Sprintf("dailymotionsearch10:%s full movie", titleYear),
                // Final YouTube sweep — broader phrasing
                fmt.Sprintf("ytsearch10:%s full movie HD", m.Title),
        }

        for _, q := range queries {
                if runYtdlpSearch(ytdlp, outBase, q) {
                        return nil
                }
        }
        return fmt.Errorf("all download strategies failed")
}

func downloadOne(m Movie, outDir string, rank int, ytdlp string, wg *sync.WaitGroup, upg *upgradeSession) {
        defer wg.Done()

        outBase := filepath.Join(outDir, safeName(fmt.Sprintf("%03d - %s (%s)", rank, m.Title, m.Year)))

        // Skip if already downloaded (any video extension, > 10 MB)
        for _, ext := range []string{"mp4", "mkv", "webm", "avi"} {
                path := outBase + "." + ext
                if info, err := os.Stat(path); err == nil && info.Size() > 10*1024*1024 {
                        printResult(rank, m, "skip", "exists")
                        atomic.AddInt64(&dlSkip, 1)
                        return
                }
        }

        dlErr := downloadMovie(ytdlp, m, outBase)
        if dlErr != nil {
                if newPath, _, upgraded := upg.tryOnce(ytdlp); upgraded {
                        dlErr = downloadMovie(newPath, m, outBase)
                        if dlErr == nil {
                                upg.printUpgradeNotice()
                        }
                }
        }

        if dlErr == nil {
                for _, ext := range []string{"mp4", "mkv", "webm", "avi"} {
                        path := outBase + "." + ext
                        if info, err := os.Stat(path); err == nil {
                                size := fmt.Sprintf("%.0fMB", float64(info.Size())/(1<<20))
                                printResult(rank, m, "done", size)
                                atomic.AddInt64(&dlDone, 1)
                                return
                        }
                }
        }

        printResult(rank, m, "fail", "not found")
        atomic.AddInt64(&dlFail, 1)
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
        workers := flag.Int("workers", 2, "Parallel download workers")
        outDir := flag.String("out", "", "Output directory (default: ~/Movies/<query>)")
        flag.Parse()

        obanner()

        // ── Locate / install yt-dlp ───────────────────────────────────────────
        ytdlp := findYtdlp()
        if ytdlp == "" {
                ytdlp = installYtdlp()
                if ytdlp == "" {
                        os.Exit(1)
                }
        }
        osuccess(fmt.Sprintf("yt-dlp ready: %s", ytdlp))

        // Weekly update check in background
        updateCh := make(chan string, 1)
        go checkAndUpdateYtdlp(ytdlp, updateCh)

        // ── Get movie title / search query ────────────────────────────────────
        query := strings.Join(flag.Args(), " ")
        if query == "" {
                query = prompt("Movie Title / Search")
        }
        if query == "" {
                oerror("No search query specified.")
                os.Exit(1)
        }

        // ── Search all sources in parallel ────────────────────────────────────
        oheader(fmt.Sprintf("Searching: %s%s%s", Bold+Yellow, query, Reset))
        oinfo(fmt.Sprintf("Querying %d sources simultaneously: iTunes (%d countries) · YTS · archive.org · Cinemeta",
                len(storefronts)+3, len(storefronts)))

        start := time.Now()
        movies := searchAll(query)
        elapsed := time.Since(start)

        if len(movies) == 0 {
                oerror(fmt.Sprintf("No movies found for: %s", query))
                os.Exit(0)
        }

        // Count sources for summary
        sourceCount := make(map[string]int)
        imdbCount := 0
        for _, m := range movies {
                src := m.Source
                if strings.HasPrefix(src, "iTunes/") {
                        src = "iTunes"
                }
                sourceCount[src]++
                if m.IMDbID != "" {
                        imdbCount++
                }
        }

        osuccess(fmt.Sprintf("Found %s%d movies%s in %.1fs  (%d with IMDb IDs)",
                Bold, len(movies), Reset, elapsed.Seconds(), imdbCount))

        // Non-blocking drain of the update check
        updatePrinted := false
        select {
        case msg := <-updateCh:
                updatePrinted = true
                if msg != "" {
                        oinfo(msg)
                }
        default:
        }
        fmt.Println()

        // ── Display catalog table ─────────────────────────────────────────────
        oheader("Movie Catalog")
        printTable(movies)
        fmt.Println()

        // ── Selection ─────────────────────────────────────────────────────────
        oinfo("Enter movie numbers to download")
        oinfo("  Press Enter = all movies")
        oinfo("  1,3,5     = specific movies")
        oinfo("  1-10      = a range")
        fmt.Println()
        selInput := prompt("Selection")
        selected := parseSelection(selInput, len(movies))
        if len(selected) == 0 {
                oerror("No valid selection.")
                os.Exit(0)
        }

        // ── Set up output directory ───────────────────────────────────────────
        dir := *outDir
        if dir == "" {
                home, _ := os.UserHomeDir()
                dir = filepath.Join(home, "Movies", safeName(query))
        }
        if err := os.MkdirAll(dir, 0755); err != nil {
                oerror(fmt.Sprintf("Cannot create directory: %v", err))
                os.Exit(1)
        }

        // Drain update check (cap at 2s)
        if !updatePrinted {
                select {
                case msg := <-updateCh:
                        if msg != "" {
                                oinfo(msg)
                        }
                case <-time.After(2 * time.Second):
                }
                fmt.Println()
        }

        // ── Download ──────────────────────────────────────────────────────────
        oheader(fmt.Sprintf("Downloading %d movies → %s", len(selected), dir))
        oinfo(fmt.Sprintf("Workers: %d  |  Format: MP4 (best quality)  |  7 search strategies", *workers))
        fmt.Println()

        dlDone, dlFail, dlSkip = 0, 0, 0
        sem := make(chan struct{}, *workers)
        var wg sync.WaitGroup
        upg := &upgradeSession{}

        for _, idx := range selected {
                m := movies[idx-1]
                wg.Add(1)
                sem <- struct{}{}
                go func(m Movie, rank int) {
                        defer func() { <-sem }()
                        downloadOne(m, dir, rank, ytdlp, &wg, upg)
                }(m, idx)
        }
        wg.Wait()

        // ── Summary ───────────────────────────────────────────────────────────
        fmt.Println()
        odivider()
        done := atomic.LoadInt64(&dlDone)
        fail := atomic.LoadInt64(&dlFail)
        skip := atomic.LoadInt64(&dlSkip)
        fmt.Printf("  %s%s[+]%s %sDone%s  %s✓ %d downloaded%s  %s→ %d skipped%s  %s✗ %d failed%s\n",
                Bold, Green, Reset,
                Bold, Reset,
                Green, done, Reset,
                Cyan, skip, Reset,
                Red, fail, Reset)
        odivider()
        fmt.Println()
}
