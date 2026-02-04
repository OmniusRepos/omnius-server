package utils

import (
	"regexp"
	"strings"
)

// NormalizeTitle normalizes a movie/series title for comparison
func NormalizeTitle(title string) string {
	title = strings.ToLower(title)

	// Remove special characters
	re := regexp.MustCompile(`[^a-z0-9\s]`)
	title = re.ReplaceAllString(title, "")

	// Remove common articles
	title = strings.TrimPrefix(title, "the ")
	title = strings.TrimPrefix(title, "a ")
	title = strings.TrimPrefix(title, "an ")

	// Collapse whitespace
	spaceRe := regexp.MustCompile(`\s+`)
	title = spaceRe.ReplaceAllString(title, " ")

	return strings.TrimSpace(title)
}

// ExtractYear extracts a year from a title string
func ExtractYear(title string) (string, int) {
	re := regexp.MustCompile(`\((\d{4})\)$`)
	match := re.FindStringSubmatch(title)
	if len(match) >= 2 {
		year := 0
		for _, c := range match[1] {
			year = year*10 + int(c-'0')
		}
		cleanTitle := strings.TrimSpace(re.ReplaceAllString(title, ""))
		return cleanTitle, year
	}
	return title, 0
}

// GenerateSlug creates a URL-friendly slug from a title
func GenerateSlug(title string) string {
	slug := strings.ToLower(title)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	re := regexp.MustCompile(`[^a-z0-9\-]`)
	slug = re.ReplaceAllString(slug, "")

	// Collapse multiple hyphens
	hyphenRe := regexp.MustCompile(`-+`)
	slug = hyphenRe.ReplaceAllString(slug, "-")

	return strings.Trim(slug, "-")
}

// IsPartOfFranchise checks if a title matches a franchise pattern
func IsPartOfFranchise(title string, franchiseName string) bool {
	normalized := NormalizeTitle(title)
	franchiseNorm := NormalizeTitle(franchiseName)

	return strings.HasPrefix(normalized, franchiseNorm)
}

// ExtractFranchiseNumber extracts sequel/part number from a title
func ExtractFranchiseNumber(title string) int {
	// Check for roman numerals
	romanRe := regexp.MustCompile(`\b(II|III|IV|V|VI|VII|VIII|IX|X)\b`)
	romanMatch := romanRe.FindString(strings.ToUpper(title))
	if romanMatch != "" {
		return romanToInt(romanMatch)
	}

	// Check for Arabic numerals
	numRe := regexp.MustCompile(`\b(\d+)\b`)
	numMatch := numRe.FindStringSubmatch(title)
	if len(numMatch) >= 2 {
		num := 0
		for _, c := range numMatch[1] {
			num = num*10 + int(c-'0')
		}
		if num >= 2 && num <= 20 {
			return num
		}
	}

	return 1
}

func romanToInt(s string) int {
	romanMap := map[string]int{
		"I": 1, "II": 2, "III": 3, "IV": 4, "V": 5,
		"VI": 6, "VII": 7, "VIII": 8, "IX": 9, "X": 10,
	}
	if val, ok := romanMap[s]; ok {
		return val
	}
	return 1
}

// TitleSimilarity calculates the similarity between two titles (0-1)
func TitleSimilarity(a, b string) float64 {
	aNorm := NormalizeTitle(a)
	bNorm := NormalizeTitle(b)

	if aNorm == bNorm {
		return 1.0
	}

	// Simple word overlap calculation
	aWords := strings.Fields(aNorm)
	bWords := strings.Fields(bNorm)

	if len(aWords) == 0 || len(bWords) == 0 {
		return 0.0
	}

	wordSet := make(map[string]bool)
	for _, w := range aWords {
		wordSet[w] = true
	}

	matches := 0
	for _, w := range bWords {
		if wordSet[w] {
			matches++
		}
	}

	maxLen := len(aWords)
	if len(bWords) > maxLen {
		maxLen = len(bWords)
	}

	return float64(matches) / float64(maxLen)
}
