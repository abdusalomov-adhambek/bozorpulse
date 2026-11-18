// Package query maps the product words users type (Uzbek, Russian) to the
// English words the product catalog actually stores. The search layer is a
// plain case-insensitive LIKE, so without this mapping a perfectly reasonable
// question like "soat sotib olmoqchiman" finds nothing: the catalog calls that
// product "Smart Watch".
package query

import (
	"sort"
	"strings"
	"unicode"
)

// productTerms maps a local-language term (single word or short phrase) to an
// English token that occurs in a catalog product name. Keys are tokenized with
// the same rules as user input, so apostrophes and hyphens are harmless.
var productTerms = map[string]string{
	// watches
	"soat": "watch", "soatlar": "watch", "qo'l soati": "watch", "qol soati": "watch",
	"aqlli soat": "watch", "smart soat": "watch", "часы": "watch",
	"смарт часы": "watch", "наручные часы": "watch",
	// cement and building materials
	"sement": "cement", "tsement": "cement", "portland sement": "cement",
	"цемент": "cement", "портландцемент": "cement",
	"g'isht": "brick", "кирпич": "brick", "faner": "plywood", "фанера": "plywood",
	"armatura": "rebar", "арматура": "rebar", "plitka": "tile", "плитка": "tile",
	"qum": "sand", "песок": "sand",
	// electronics
	"naushnik": "headphone", "naushniklar": "headphone", "наушники": "headphone",
	"karnay": "speaker", "kolonka": "speaker", "колонка": "speaker", "динамик": "speaker",
	"lampa": "lamp", "chiroq": "lamp", "лампа": "lamp",
	"zaryadka": "charger", "зарядка": "charger", "powerbank": "power bank", "пауэрбанк": "power bank",
	// food
	"guruch": "rice", "рис": "rice", "basmati": "rice",
	"yog'": "oil", "yog": "oil", "moy": "oil", "масло": "oil",
	"un": "flour", "мука": "flour", "bug'doy": "wheat", "bugdoy": "wheat", "пшеница": "wheat",
	"shakar": "sugar", "сахар": "sugar", "choy": "tea", "чай": "tea",
	"olma": "apple", "olmalar": "apple", "яблоко": "apple", "яблоки": "apple",
	"kartoshka": "potato", "картошка": "potato", "картофель": "potato",
	"piyoz": "onion", "лук": "onion", "sabzi": "carrot", "морковь": "carrot",
	"pomidor": "tomato", "помидор": "tomato", "томаты": "tomato",
	"urug'": "seed", "семена": "seed", "kungaboqar urug'i": "seed",
	// home and textiles
	"futbolka": "t-shirt", "футболка": "t-shirt", "jinsi": "jean", "джинсы": "jean", "denim": "jean",
	"sviter": "sweater", "свитер": "sweater", "mata": "fabric", "gazlama": "fabric", "ткань": "fabric",
	"choyshab": "linen", "постельное белье": "linen", "постельное бельё": "linen",
	"sochiq": "towel", "полотенце": "towel", "idish-tovoq": "utensil", "посуда": "utensil",
	"chaynik": "kettle", "чайник": "kettle", "qozon": "pot", "кастрюля": "pot",
	"kesish taxtasi": "board", "quti": "box", "коробка": "box",
}

type entry struct {
	tokens    []string
	canonical string
}

var entries []entry

func init() {
	// English product words are canonical too: "cement olmoqchiman" must
	// reduce to "cement" just like "soat sotib olmoqchiman" reduces to "watch".
	canonicals := make([]string, 0, len(productTerms))
	for _, c := range productTerms {
		canonicals = append(canonicals, c)
	}
	for _, c := range canonicals {
		if _, ok := productTerms[c]; !ok {
			productTerms[c] = c
		}
	}
	keys := make([]string, 0, len(productTerms))
	for k := range productTerms {
		keys = append(keys, k)
	}
	// Longest phrases first so "qo'l soati" wins over "soat".
	sort.Slice(keys, func(i, j int) bool {
		a, b := tokenize(keys[i]), tokenize(keys[j])
		if len(a) != len(b) {
			return len(a) > len(b)
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		entries = append(entries, entry{tokens: tokenize(k), canonical: productTerms[k]})
	}
}

func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '’'
	})
}

// matchesToken also accepts inflections for single-word terms ("цемент" in
// "цемента", "soat" in "soatlar"). Short terms are excluded so "un" (flour)
// cannot swallow every word beginning with those letters.
func matchesToken(token, term string) bool {
	if token == term {
		return true
	}
	if len(term) < 4 || len(token) < 4 {
		return false
	}
	return strings.HasPrefix(token, term) || strings.HasPrefix(term, token)
}

func containsSequence(tokens, phrase []string) bool {
	if len(phrase) == 0 || len(phrase) > len(tokens) {
		return false
	}
	for i := 0; i+len(phrase) <= len(tokens); i++ {
		match := true
		for j, term := range phrase {
			ok := tokens[i+j] == term
			if !ok && len(phrase) == 1 {
				ok = matchesToken(tokens[i+j], term)
			}
			if !ok {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Normalize returns the catalog keyword for q. A known local term is replaced
// by its English equivalent; unrelated words are kept (lower-cased) so SKUs
// and already-English searches keep working. It is idempotent.
func Normalize(q string) string {
	tokens := tokenize(q)
	if len(tokens) == 0 {
		return ""
	}
	for _, e := range entries {
		if containsSequence(tokens, e.tokens) {
			return e.canonical
		}
	}
	// No dictionary hit: keep the original spelling so SKUs such as
	// "CN-PC-050" still match the catalog's SKU column.
	return strings.ToLower(strings.TrimSpace(q))
}
