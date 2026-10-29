package geo

import (
	"strings"
	"unicode"
)

// Normalize приводит название к виду для поиска: нижний регистр, ё→е,
// латинские диакритики снимаются, пробелы схлопываются.
//
// Нужно потому, что в дампе GeoNames названия лежат вперемешку: «Москва»,
// «Moscow», «Moskva», «São Paulo» и «Sao Paulo» — это разные строки.
func Normalize(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == 'ё' {
			r = 'е'
		}
		if fold, ok := foldRune(r); ok {
			b.WriteString(fold)
			continue
		}
		if unicode.IsSpace(r) {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	// Схлопываем пробелы и обрезаем края.
	return strings.Join(strings.Fields(b.String()), " ")
}

// foldRune снимает диакритику с латинских букв (без внешних зависимостей).
func foldRune(r rune) (string, bool) {
	switch r {
	// a
	case 'à', 'á', 'â', 'ã', 'ä', 'å', 'ā', 'ă', 'ą':
		return "a", true
	// c
	case 'ç', 'ć', 'č', 'ĉ', 'ċ':
		return "c", true
	// d
	case 'ď', 'đ':
		return "d", true
	// e
	case 'è', 'é', 'ê', 'ë', 'ē', 'ė', 'ę', 'ě':
		return "e", true
	// g
	case 'ĝ', 'ğ', 'ġ', 'ģ':
		return "g", true
	// i
	case 'ì', 'í', 'î', 'ï', 'ī', 'į', 'ı':
		return "i", true
	// l
	case 'ł', 'ľ', 'ĺ':
		return "l", true
	// n
	case 'ñ', 'ń', 'ň':
		return "n", true
	// o
	case 'ò', 'ó', 'ô', 'õ', 'ö', 'ø', 'ō', 'ő':
		return "o", true
	// r
	case 'ř', 'ŕ':
		return "r", true
	// s
	case 'ś', 'š', 'ş', 'ș':
		return "s", true
	// t
	case 'ť', 'ţ', 'ț':
		return "t", true
	// u
	case 'ù', 'ú', 'û', 'ü', 'ū', 'ů', 'ű', 'ų':
		return "u", true
	// y
	case 'ý', 'ÿ':
		return "y", true
	// z
	case 'ź', 'ž', 'ż':
		return "z", true
	// лигатуры и особые
	case 'æ':
		return "ae", true
	case 'œ':
		return "oe", true
	case 'ß':
		return "ss", true
	}
	return "", false
}
