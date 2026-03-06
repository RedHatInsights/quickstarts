# Fuzzy Search Implementation

## Overview

The fuzzy search feature enables the help panel and other UI components to find quickstarts even when users make typos. It uses PostgreSQL's Levenshtein distance algorithm with intelligent strategy selection:

- **Word-Level Matching**: For single-word queries like "ansibel" → "Ansible"
- **Phrase-Level Matching**: For multi-word queries like "automation hb" → "automation hub"
- **Hybrid Fallback**: Automatic fallback to ILIKE when fuzzy search returns no results

## Features

- ✅ **Word-Level Typo Tolerance**: Finds single-word typos like "ansibel" → "Ansible", "kuberntes" → "Kubernetes"
- ✅ **Phrase-Level Typo Tolerance**: Finds typos in full phrases like "automation hb" → "automation hub"
- ✅ **Smart Detection**: Automatically chooses word-level or phrase-level matching based on query
- ✅ **Hybrid Fallback**: Automatically falls back to partial matching when fuzzy search returns no results
- ✅ **Configurable Threshold**: Adjustable maximum distance for matches (default: 3)
- ✅ **Result Ordering**: Results are ordered by distance (best matches first)
- ✅ **Tag Filtering**: Works in combination with existing tag-based filters
- ✅ **Backward Compatible**: Existing search functionality remains unchanged

## Configuration

### Environment Variable

```bash
FUZZY_SEARCH_DISTANCE_THRESHOLD=3  # Default: 3 character differences
```

Recommended values:
- `1-2`: Strict matching
- `3`: Moderate tolerance (recommended)
- `4-5`: Lenient matching

### Database Setup

The PostgreSQL `fuzzystrmatch` extension is automatically enabled during database initialization.

## API Usage

### Endpoint

`GET /api/quickstarts/v1/quickstarts`

### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `display-name` | string | Search term | `ansibel` |
| `fuzzy` | boolean | Enable fuzzy search | `true` |
| `limit` | integer | Max results | `10` |
| `offset` | integer | Pagination offset | `0` |
| `bundle` | array | Filter by bundle | `ansible` |

### Examples

```bash
# Single-word typo (word-level matching)
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=ansibel&fuzzy=true"
# Finds: "Create your first Ansible Playbook"

# Multi-word typo (phrase-level matching)
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=Getting%20started%20with%20automation%20hb&fuzzy=true"
# Finds: "Getting started with automation hub"

# Partial word (ILIKE fallback)
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=ansible&fuzzy=true"
# Finds: All quickstarts containing "ansible"
```

## Implementation Details

### Smart Multi-Strategy Algorithm

The fuzzy search automatically selects the best strategy based on the query:

#### Strategy 1: Word-Level Matching (Single-word queries)

**When**: Query contains no spaces (e.g., "ansibel", "kuberntes")

**How it works**:
1. Split display names into individual words
2. Calculate Levenshtein distance between search term and each word
3. Return matches where minimum distance ≤ threshold
4. Order by closest match first

**SQL Implementation**:
```sql
WITH word_distances AS (
  SELECT q.*, MIN(levenshtein(LOWER(?), word)) as distance
  FROM quickstarts q,
  LATERAL unnest(regexp_split_to_array(LOWER(q.content->'spec'->>'displayName'), '\s+')) as word
  WHERE q.content->'spec'->>'displayName' IS NOT NULL
  GROUP BY q.id
  HAVING MIN(levenshtein(LOWER(?), word)) <= ?
)
SELECT * FROM word_distances ORDER BY distance ASC
```

**Perfect for**: Typos in single words
- "ansibel" → "Ansible" (distance: 2)
- "kuberntes" → "Kubernetes" (distance: 1)
- "automaton" → "Automation" (distance: 1)

#### Strategy 2: Full-Phrase Matching (Multi-word queries)

**When**: Query contains spaces (e.g., "Getting started with automation hb")

**How it works**:
1. Calculate Levenshtein distance between entire search phrase and entire display name
2. Return matches where distance ≤ threshold
3. Order by distance

**SQL Implementation**:
```sql
SELECT *, levenshtein(LOWER(content->'spec'->>'displayName'), LOWER(?)) as distance
FROM quickstarts
WHERE levenshtein(LOWER(content->'spec'->>'displayName'), LOWER(?)) <= ?
ORDER BY distance ASC
```

**Perfect for**: Typos in full phrases
- "Getting started with automation hb" → "Getting started with automation hub" (distance: 1)
- "Deploy Java applicaton" → "Deploy Java application" (distance: 1)

#### Strategy 3: Hybrid Fallback (When fuzzy returns no results)

**When**: Neither word-level nor phrase-level matching finds results

**How it works**:
Falls back to PostgreSQL ILIKE pattern matching for partial matches

**SQL Implementation**:
```sql
SELECT * FROM quickstarts
WHERE LOWER(content->'spec'->>'displayName') ILIKE LOWER('%?%')
```

**Perfect for**: Partial words with correct spelling
- "ansible" → "Create your first Ansible Playbook"
- "kubernetes" → "Deploy a Java application on Kubernetes"

### Behavior Matrix

| Search Type | Fuzzy=false | Fuzzy=true (Smart) |
|-------------|-------------|---------------------|
| Exact match | ✅ Found | ✅ Found (all strategies) |
| Partial word (correct) | ✅ Found | ✅ Found (ILIKE fallback) |
| Partial word (typo) | ❌ Not found | ✅ Found (Word-level) 🆕 |
| Full phrase (typo) | ❌ Not found | ✅ Found (Phrase-level) |
| Multi-word (correct) | ✅ Found | ✅ Found (all strategies) |

### Database Compatibility

#### PostgreSQL (Production)
- ✅ Full fuzzy search support
- ✅ Word-level matching with `regexp_split_to_array()`
- ✅ Phrase-level matching with `levenshtein()`
- ✅ Results ordered by relevance

#### SQLite (Testing/Development)
- ⚠️  Automatic fallback to ILIKE
- ❌ No typo tolerance (Levenshtein not available)
- ✅ Exact and partial matches work
- ✅ No errors or failures

### Performance Considerations

1. **Case Insensitivity**: All comparisons use LOWER()
2. **Word Splitting**: `regexp_split_to_array()` is efficient for short strings
3. **Distance Threshold**: Kept low (≤3) for performance and relevance
4. **Hybrid Overhead**: Second query only runs if first returns nothing
5. **Index Optimization**: Consider GIN indexes on JSONB content

## Testing

### Unit Tests

```bash
go test ./pkg/routes/... -v -run TestFuzzySearch
```

### Test Scenarios Covered

1. **Single-word typos** (Word-level)
   - "ansibel" → "Ansible"
   - "kuberntes" → "Kubernetes"
   - "automaton" → "Automation"

2. **Multi-word phrase typos** (Phrase-level)
   - "Getting started with automation hb" → "...hub"
   - "Geting Startd" → "Getting Started"

3. **Partial words** (ILIKE fallback)
   - "ansible" → "Create your first Ansible Playbook"
   - "kubernetes" → "Deploy Java application on Kubernetes"

4. **Case insensitivity**
   - "ansible" → "Ansible"
   - "KUBERNETES" → "Kubernetes"

5. **Distance threshold enforcement**
- Completely unrelated terms return no results

6. **Tag filter integration**
   - Fuzzy search + bundle filters
   - Fuzzy search + application filters

## Troubleshooting

### Fuzzy Search Not Working

1. **Check Extension**:
   ```sql
   SELECT * FROM pg_extension WHERE extname = 'fuzzystrmatch';
   ```

2. **Check Threshold**: Ensure `FUZZY_SEARCH_DISTANCE_THRESHOLD` is set

3. **Review Logs**: Check for extension creation errors

### No Results Found

1. **Distance Too Large**: Search term may be too different
2. **Threshold Too Low**: Consider increasing threshold
3. **SQLite in Use**: Fuzzy search falls back to ILIKE in test mode

### Single-Word Typo Not Working

1. **Threshold**: Ensure distance is ≤ configured threshold
2. **Word Boundaries**: Search uses space-delimited words
3. **Database**: Ensure PostgreSQL is being used (not SQLite)

## Examples

```bash
# Word-level fuzzy matching
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=ansibel&fuzzy=true"
# Returns: Ansible-related quickstarts

curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=kuberntes&fuzzy=true"
# Returns: Kubernetes quickstarts

# Phrase-level fuzzy matching
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=Getting%20started%20with%20automation%20hb&fuzzy=true"
# Returns: "Getting started with automation hub"

# With tag filters
curl "http://localhost:8000/api/quickstarts/v1/quickstarts?display-name=ansibel&fuzzy=true&bundle=ansible"
# Returns: Ansible quickstarts in ansible bundle
```

## Future Enhancements

1. **Extended Search Scope**: Add fuzzy search to `spec.description` and `spec.tasks`
2. **Weighted Results**: Different weights for title vs description matches
3. **Synonym Support**: Map common synonyms (e.g., "k8s" → "Kubernetes")
4. **Search Analytics**: Track search terms to improve typo tolerance
5. **Multi-field Search**: Search across multiple fields simultaneously

## References

- [PostgreSQL fuzzystrmatch Documentation](https://www.postgresql.org/docs/current/fuzzystrmatch.html#FUZZYSTRMATCH-LEVENSHTEIN)
- [Levenshtein Distance Algorithm](https://en.wikipedia.org/wiki/Levenshtein_distance)
- [PostgreSQL String Functions](https://www.postgresql.org/docs/current/functions-string.html)
