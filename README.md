# moviedl

```
  +══════════════════════════════════════════════════════════+
  |  ███╗   ███╗ ██████╗ ██╗   ██╗██╗███████╗██████╗ ██╗     |
  |  ████╗ ████║██╔═══██╗██║   ██║██║██╔════╝██╔══██╗██║     |
  |  ██╔████╔██║██║   ██║██║   ██║██║█████╗  ██║  ██║██║     |
  |  ██║╚██╔╝██║██║   ██║╚██╗ ██╔╝██║██╔══╝  ██║  ██║██║     |
  |  ██║ ╚═╝ ██║╚██████╔╝ ╚████╔╝ ██║███████╗██████╔╝███████╗|
  |  ╚═╝     ╚═╝ ╚═════╝   ╚═══╝  ╚═╝╚══════╝╚═════╝ ╚══════╝|
  |                                                            |
  |  Global Movie Downloader  v1.0  github.com/krainium       |
  +══════════════════════════════════════════════════════════+
```

Type a movie title. Get a full catalog. Download the film as MP4. That's it.

No API keys. No accounts. No trailers. No clips.

---

## What it does

You give it a title. It fires 182 requests simultaneously across four independent sources — iTunes storefronts in 175 countries, YTS, the Internet Archive, plus Cinemeta which is backed by the full IMDb database. Results come back in roughly two seconds.

Each result in the table shows the runtime, director, year, IMDb rating, where it came from. Movies identified by IMDb ID are deduplicated automatically, so you never see the same film listed twice.

You pick which ones to download. moviedl handles the rest.

---

## How downloading works

For each movie it tries seven strategies in order. It stops the moment one succeeds.

| # | Strategy | Notes |
|---|----------|-------|
| 0 | Direct archive.org URL | Public domain films only — fastest path, no search needed |
| 1 | YouTube — exact quoted title | `ytsearch10:"TITLE YEAR" full movie` |
| 2 | YouTube — unquoted title | `ytsearch10:TITLE YEAR full movie` |
| 3 | YouTube — alternate phrasing | `ytsearch10:TITLE YEAR full film` |
| 4 | YouTube — free film search | `ytsearch10:TITLE full movie free` |
| 5 | Google Video | `gvsearch10:TITLE YEAR full movie` |
| 6 | Dailymotion | `dailymotionsearch10:TITLE YEAR full movie` |
| 7 | YouTube HD sweep | `ytsearch10:TITLE full movie HD` |

Each search pulls up to 10 candidates. Anything under 55 minutes gets skipped automatically — so no trailer ever gets saved to disk.

---

## What you get

Files land in `~/Movies/<query>/` by default. Each filename is numbered so they sort cleanly.

```
001 - Nosferatu (1922).mp4
002 - The Dark Knight (2008).mp4
003 - Interstellar (2014).mp4
```

Files that already exist on disk above 10 MB are skipped. Running the same search twice won't re-download anything.

---

## Catalog sources

**iTunes Movie Store** hits 175 country storefronts in parallel. Each storefront runs its own region-specific catalog. This surfaces titles that simply do not appear when you search from a single country.

**YTS** is a high-quality encode catalog. Every entry carries an IMDb code so deduplication is exact.

**Internet Archive** is searched exclusively within its curated `feature_films` collection — a vault of public domain full-length films. This keeps out trailers, TV clips, theme song uploads, anything that isn't a feature film.

**Cinemeta** is the metadata engine behind Stremio. It pulls from IMDb directly. Two steps happen per result: first a catalog search gets IMDb IDs, then a parallel meta call fetches the director, genres, runtime in minutes, IMDb rating. Up to five of these run at the same time.

---

## yt-dlp is managed for you

You do not need to install or maintain yt-dlp manually.

| Situation | What happens |
|-----------|--------------|
| Not installed | moviedl installs it via pip before the first download |
| pip not found | You get a clear message with exactly what to install |
| Version is over a week old | Background check runs while you browse the catalog — upgrades silently |
| A download fails | moviedl checks for a newer yt-dlp version unconditionally, upgrades if one exists, retries |
| Many films fail at once | Only one upgrade attempt runs regardless of how many workers are active |

---

## Build

Requires Go 1.21 or newer. Zero external Go dependencies.

```bash
git clone https://github.com/krainium/Moviedl
cd moviedl
go build -o moviedl Moviedl.go
```

---

## Usage

```bash
# Interactive prompt
./moviedl

# Pass a title directly
./moviedl "Inception"
./moviedl "The Dark Knight"
./moviedl "Nosferatu"

# Include the year for precision
./moviedl "Interstellar 2014"

# Set workers or output folder
./moviedl --workers 4 --out ~/Downloads "Oppenheimer"
```

**Flags**

| Flag | Default | What it does |
|------|---------|--------------|
| `--workers N` | 2 | How many films download at the same time |
| `--out DIR` | `~/Movies/<query>` | Where files are saved |

**Selecting which movies to download**

| What you type | What gets downloaded |
|---------------|----------------------|
| Enter with nothing | Everything in the list |
| `3` | Just movie number 3 |
| `1-10` | Movies 1 through 10 |
| `1,4,7` | Specific picks |
| `1-5,10,15-20` | Ranges mixed with singles |

---

## Country coverage

| Region | Storefronts |
|--------|-------------|
| North America | 3 |
| Latin America | 36 |
| Europe | 48 |
| Middle East | 11 |
| Africa | 45 |
| Asia Pacific | 35 |
| **Total** | **175** |

---

## What will download vs what won't

Public domain films from the Internet Archive download instantly at full speed via direct URL. Classic films uploaded to YouTube as full movies download well. Many films from the 1970s through 2000s have free full uploads across YouTube and Dailymotion.

Recent major studio releases — anything still in theaters or just out of theaters — are not freely available on any platform yt-dlp can reach. The tool will tell you "not found" rather than download a trailer or a clip.
