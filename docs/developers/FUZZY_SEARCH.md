# Fuzzy Search

## Overview

Fuzzy search finds quickstarts even when users make typos, using PostgreSQL's Levenshtein distance algorithm.

**How it works:**
- Splits query and display names into words
- Matches each query word to closest display word
- Returns results with at least one matching word within threshold (default: 3 characters)
- Ranks by: match count (DESC) → total distance (ASC)

## Configuration

```bash
FUZZY_SEARCH_DISTANCE_THRESHOLD=3  # 1-2: strict, 3: moderate (default), 4-5: lenient
```

PostgreSQL `fuzzystrmatch` extension is automatically enabled. SQLite falls back to ILIKE.

## Usage

Add `fuzzy=true` to any search query:

```bash
# Single-word typo
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=ansibel&fuzzy=true"
# Finds: "Ansible Automation Platform", "Ansible Playbook"

# Multi-word typo
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=geting%20startd&fuzzy=true"
# Finds: "Getting started with automation hub", "Getting started with RHEL"

# Partial matching (ranked by match count)
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=ansible%20automation%20playbook&fuzzy=true"
# Results ranked:
# 1. "Ansible Automation Platform" (2/3 words = 66%)
# 2. "Ansible Playbook" (2/3 words = 66%)
# 3. "Getting started with automation hub" (1/3 words = 33%)

# With tag filters
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=ansibel&fuzzy=true&bundle=ansible"
```

## Examples

| Query | Finds | Reason |
|-------|-------|--------|
| `ansibel` | "Ansible" | 2-char distance |
| `kuberntes` | "Kubernetes" | 1-char distance |
| `geting startd` | "Getting started" | 1-char each word |
| `automaton hb` | "automation hub" | 1-char each word |
| `ansible` | "Ansible Automation Platform" | ILIKE fallback (exact spelling) |

## Ranking

Results sorted by:
1. **Number of matching words** (more = better)
2. **Total Levenshtein distance** (lower = better)

Example: Query `"ansible automation"`
- "Ansible Automation Platform" → 2/2 words (100%) → rank 1
- "Create your first Ansible Playbook" → 1/2 words (50%) → rank 3

## Database Support

- **PostgreSQL**: Full fuzzy search with Levenshtein
- **SQLite**: Automatic fallback to ILIKE (tests skip fuzzy tests)

## Testing

```bash
# Unit tests (SQLite - fuzzy tests skipped)
make test

# Manual testing (PostgreSQL)
make infra && make migrate && go run main.go
```

## Troubleshooting

**No results found:**
- Check threshold (may be too low for query)
- Verify PostgreSQL is running (not SQLite)
- Check extension: `SELECT * FROM pg_extension WHERE extname = 'fuzzystrmatch';`

**Unexpected results:**
- Fuzzy search may return broader matches than expected (by design)
- Use lower threshold for stricter matching
- Disable fuzzy with `fuzzy=false` for exact ILIKE matching
