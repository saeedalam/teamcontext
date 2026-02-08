package search

import (
	"encoding/binary"
	"math"
	"sort"
	"strings"
	"unicode"
)

// MaxVocabularySize caps the vocabulary to prevent excessive memory usage.
const MaxVocabularySize = 5000

// TFIDFEngine provides TF-IDF based semantic search without requiring an LLM.
type TFIDFEngine struct {
	Vocabulary map[string]int `json:"vocabulary"` // term → index
	IDF        []float64      `json:"idf"`        // inverse document frequency per term
	DocCount   int            `json:"doc_count"`
}

// NewTFIDFEngine creates an empty TF-IDF engine.
func NewTFIDFEngine() *TFIDFEngine {
	return &TFIDFEngine{
		Vocabulary: make(map[string]int),
	}
}

// synonyms maps common programming abbreviations to their full forms.
var synonyms = map[string]string{
	"auth":    "authentication",
	"db":      "database",
	"api":     "endpoint",
	"err":     "error",
	"config":  "configuration",
	"env":     "environment",
	"repo":    "repository",
	"msg":     "message",
	"req":     "request",
	"res":     "response",
	"resp":    "response",
	"ctx":     "context",
	"fn":      "function",
	"func":    "function",
	"pkg":     "package",
	"cmd":     "command",
	"arg":     "argument",
	"args":    "arguments",
	"param":   "parameter",
	"params":  "parameters",
	"btn":     "button",
	"nav":     "navigation",
	"impl":    "implementation",
	"init":    "initialize",
	"util":    "utility",
	"utils":   "utilities",
	"lib":     "library",
	"libs":    "libraries",
	"dev":     "development",
	"prod":    "production",
	"dep":     "dependency",
	"deps":    "dependencies",
}

// suffixes to strip for simple stemming, ordered longest first.
var stemmingSuffixes = []string{
	"ation", "tion", "ment", "ness", "able", "ible",
	"ing", "ous", "ive", "ful", "less", "ist",
	"ed", "ly", "er", "al", "es",
}

// simpleStem strips common English suffixes from a word.
// Only applied to words >= 5 chars; result must be >= 3 chars.
func simpleStem(word string) string {
	if len(word) < 5 {
		return word
	}
	for _, suffix := range stemmingSuffixes {
		if strings.HasSuffix(word, suffix) {
			stem := word[:len(word)-len(suffix)]
			if len(stem) >= 3 {
				return stem
			}
		}
	}
	return word
}

// stopwords is a set of common English words to filter out.
var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "is": true, "it": true, "as": true,
	"be": true, "was": true, "are": true, "were": true, "been": true, "has": true,
	"have": true, "had": true, "do": true, "does": true, "did": true, "will": true,
	"would": true, "could": true, "should": true, "may": true, "might": true,
	"this": true, "that": true, "these": true, "those": true, "not": true,
	"no": true, "if": true, "then": true, "else": true, "when": true,
	"which": true, "who": true, "whom": true, "what": true, "where": true,
	"how": true, "all": true, "each": true, "every": true, "both": true,
	"few": true, "more": true, "most": true, "other": true, "some": true,
	"such": true, "only": true, "own": true, "same": true, "so": true,
	"than": true, "too": true, "very": true, "can": true, "just": true,
	"about": true, "into": true, "through": true, "during": true, "before": true,
	"after": true, "above": true, "below": true, "between": true, "up": true,
}

// Tokenize splits text into lowercase tokens, removing stopwords and short tokens.
// It also expands each token with its stemmed form and synonym (if any),
// then generates bigrams from the expanded list.
func (e *TFIDFEngine) Tokenize(text string) []string {
	text = strings.ToLower(text)

	// Split on non-alphanumeric characters
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})

	var expanded []string
	for _, t := range tokens {
		if len(t) < 2 {
			continue
		}
		if stopwords[t] {
			continue
		}
		// Add original token
		expanded = append(expanded, t)
		// Add stemmed form if different
		stemmed := simpleStem(t)
		if stemmed != t {
			expanded = append(expanded, stemmed)
		}
		// Add synonym if exists
		if syn, ok := synonyms[t]; ok {
			expanded = append(expanded, syn)
		}
	}

	// Generate bigrams from the expanded token list
	baseLen := len(expanded)
	for i := 0; i < baseLen-1; i++ {
		expanded = append(expanded, expanded[i]+"_"+expanded[i+1])
	}

	return expanded
}

// BuildVocabulary builds the vocabulary and IDF values from a corpus of documents.
func (e *TFIDFEngine) BuildVocabulary(documents []string) {
	e.DocCount = len(documents)
	if e.DocCount == 0 {
		return
	}

	// Count document frequency for each term
	df := make(map[string]int)
	for _, doc := range documents {
		seen := make(map[string]bool)
		tokens := e.Tokenize(doc)
		for _, t := range tokens {
			if !seen[t] {
				seen[t] = true
				df[t]++
			}
		}
	}

	// Filter hapax legomena (terms appearing in only 1 document) and compute IDF scores.
	// Then keep the top terms by IDF descending — rare-but-not-unique terms are most discriminative.
	type termIDF struct {
		term string
		idf  float64
	}
	n := float64(e.DocCount)
	allTerms := make([]termIDF, 0, len(df))
	for t, f := range df {
		if f < 2 {
			continue // remove hapax legomena
		}
		idf := math.Log(n/float64(f)) + 1.0
		allTerms = append(allTerms, termIDF{t, idf})
	}
	// Sort by IDF descending — rare-but-not-unique terms are most discriminative
	sort.Slice(allTerms, func(i, j int) bool {
		if allTerms[i].idf != allTerms[j].idf {
			return allTerms[i].idf > allTerms[j].idf
		}
		return allTerms[i].term < allTerms[j].term // stable tiebreak
	})
	if len(allTerms) > MaxVocabularySize {
		allTerms = allTerms[:MaxVocabularySize]
	}

	// Build vocabulary and IDF from pruned terms
	e.Vocabulary = make(map[string]int, len(allTerms))
	e.IDF = make([]float64, len(allTerms))
	for idx, td := range allTerms {
		e.Vocabulary[td.term] = idx
		e.IDF[idx] = td.idf
	}
}

// Vectorize computes the TF-IDF vector for a single document.
func (e *TFIDFEngine) Vectorize(text string) []float64 {
	if len(e.Vocabulary) == 0 {
		return nil
	}

	tokens := e.Tokenize(text)
	if len(tokens) == 0 {
		return make([]float64, len(e.Vocabulary))
	}

	// Compute term frequency
	tf := make(map[string]float64)
	for _, t := range tokens {
		tf[t]++
	}
	// Normalize TF by document length
	docLen := float64(len(tokens))
	for k := range tf {
		tf[k] /= docLen
	}

	// Build TF-IDF vector
	vec := make([]float64, len(e.Vocabulary))
	for term, i := range e.Vocabulary {
		if tfVal, ok := tf[term]; ok {
			vec[i] = tfVal * e.IDF[i]
		}
	}

	return vec
}

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}

	return dot / denom
}

// PackVector converts a float64 vector to compressed bytes.
// Only stores non-zero values as (index uint16, value float64) pairs.
func PackVector(v []float64) []byte {
	// Count non-zero entries
	nonZero := 0
	for _, val := range v {
		if val != 0 {
			nonZero++
		}
	}

	// Format: [total_length uint16][count uint16][pairs of (index uint16, value float64)]
	buf := make([]byte, 4+nonZero*10)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(len(v)))
	binary.LittleEndian.PutUint16(buf[2:4], uint16(nonZero))

	offset := 4
	for i, val := range v {
		if val != 0 {
			binary.LittleEndian.PutUint16(buf[offset:offset+2], uint16(i))
			binary.LittleEndian.PutUint64(buf[offset+2:offset+10], math.Float64bits(val))
			offset += 10
		}
	}

	return buf
}

// UnpackVector converts compressed bytes back to a float64 vector.
func UnpackVector(b []byte) []float64 {
	if len(b) < 4 {
		return nil
	}

	totalLen := int(binary.LittleEndian.Uint16(b[0:2]))
	count := int(binary.LittleEndian.Uint16(b[2:4]))

	vec := make([]float64, totalLen)

	offset := 4
	for i := 0; i < count && offset+10 <= len(b); i++ {
		idx := int(binary.LittleEndian.Uint16(b[offset : offset+2]))
		val := math.Float64frombits(binary.LittleEndian.Uint64(b[offset+2 : offset+10]))
		if idx < totalLen {
			vec[idx] = val
		}
		offset += 10
	}

	return vec
}

// SemanticResult represents a semantic search result.
type SemanticResult struct {
	ID         string  `json:"id"`
	DocType    string  `json:"doc_type"`
	Text       string  `json:"text"`
	Similarity float64 `json:"similarity"`
}
