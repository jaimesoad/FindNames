package main

import (
	"regexp"
	"sort"
	"strings"

	"github.com/xrash/smetrics"
)

type Company struct {
	ID   uint64
	Name string
}

type Candidate struct {
	Company          Company
	NormalizedName   string
	JaroScore        float64
	TokenScore       float64
	ContainmentScore float64
	FinalScore       float64
}

var (
	punctuationRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	spaceRegex       = regexp.MustCompile(`\s+`)
)

var suffixes = map[string]struct{}{
	"llc":         {},
	"inc":         {},
	"corp":        {},
	"corporation": {},
	"ltd":         {},
	"limited":     {},
	"group":       {},
	"co":          {},
	"company":     {},
	"plc":         {},
	"gmbh":        {},
	"sa":          {},
	"bv":          {},
	"srl":         {},
	"holdings":    {},
	"holding":     {},
}

var abbreviations = map[string]string{
	"hldgs": "holdings",
	"intl":  "international",
	"tech":  "technologies",
	"svc":   "services",
}

func Normalize(name string) string {
	name = strings.ToLower(name)

	// Remove punctuation
	name = punctuationRegex.ReplaceAllString(name, " ")

	// Collapse whitespace
	name = spaceRegex.ReplaceAllString(name, " ")

	tokens := strings.Fields(name)

	out := make([]string, 0, len(tokens))

	for _, token := range tokens {
		// Expand abbreviations
		if expanded, ok := abbreviations[token]; ok {
			token = expanded
		}

		// Remove legal suffixes
		if _, isSuffix := suffixes[token]; isSuffix {
			continue
		}

		out = append(out, token)
	}

	sort.Strings(out)

	return strings.Join(out, " ")
}

func MatchCompanies(input string, companies []Company, limit int) []Candidate {
	normalizedInput := Normalize(input)

	results := make([]Candidate, 0)

	for _, company := range companies {
		normalizedCompany := Normalize(company.Name)

		jaro := smetrics.JaroWinkler(
			normalizedInput,
			normalizedCompany,
			0.7,
			4,
		)

		token := tokenSimilarity(
			normalizedInput,
			normalizedCompany,
		)

		containment := containmentScore(
			normalizedInput,
			normalizedCompany,
		)

		final := weightedScore(
			jaro,
			token,
			containment,
		)

		results = append(results, Candidate{
			Company:          company,
			NormalizedName:   normalizedCompany,
			JaroScore:        jaro,
			TokenScore:       token,
			ContainmentScore: containment,
			FinalScore:       final,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

func weightedScore(jaro, token, containment float64) float64 {
	return (jaro * 0.5) +
		(token * 0.4) +
		(containment * 0.1)
}

func tokenSimilarity(a, b string) float64 {
	aSet := toTokenSet(a)
	bSet := toTokenSet(b)

	intersection := 0

	for token := range aSet {
		if _, ok := bSet[token]; ok {
			intersection++
		}
	}

	union := len(aSet) + len(bSet) - intersection

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func containmentScore(a, b string) float64 {
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 1.0
	}

	return 0.0
}

func toTokenSet(s string) map[string]struct{} {
	tokens := strings.Fields(s)

	out := make(map[string]struct{}, len(tokens))

	for _, token := range tokens {
		out[token] = struct{}{}
	}

	return out
}
