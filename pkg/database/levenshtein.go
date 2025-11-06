package database

// levenshtein calculates the Levenshtein distance between two strings
// This is a pure Go implementation for use with SQLite
func levenshtein(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	// Convert strings to rune slices for proper Unicode handling
	r1 := []rune(s1)
	r2 := []rune(s2)

	len1 := len(r1)
	len2 := len(r2)

	// Edge cases
	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create two rows for dynamic programming
	prevRow := make([]int, len2+1)
	currRow := make([]int, len2+1)

	// Initialize first row
	for j := 0; j <= len2; j++ {
		prevRow[j] = j
	}

	// Calculate distances
	for i := 1; i <= len1; i++ {
		currRow[0] = i

		for j := 1; j <= len2; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}

			currRow[j] = min3(
				currRow[j-1]+1,    // insertion
				prevRow[j]+1,      // deletion
				prevRow[j-1]+cost, // substitution
			)
		}

		// Swap rows
		prevRow, currRow = currRow, prevRow
	}

	return prevRow[len2]
}

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
