// Package geo — локальный геокодер по дампу GeoNames (cities1000).
//
// Зачем: раньше каждый расчёт по названию города ходил в Nominatim (лимит 1 запрос/сек,
// без кэша) за координатами и в timeapi.io за часовым поясом. При лимите 429 у всех
// пользователей отваливались карты по городу, а при отказе tz-сервиса карта молча
// считалась по московскому времени. Здесь координаты и IANA-таймзона берутся из
// локального индекса: быстро, без сети и без единой внешней точки отказа.
//
// Формат файла (все числа little-endian):
//
//	магия "GEOIDX1\0" | keyCount u32 | placeCount u32 | placeSize u32 | keySize u32
//	places  : placeCount * placeSize  байт
//	keys    : keyCount   * keySize    байт (отсортированы по нормализованному имени)
//	nameBlob: u32 длина + нормализованные имена, на которые ссылаются ключи
//	strBlob : u32 длина + названия мест и таймзоны, на которые ссылаются place-записи
//
// Индекс строится командой cmd/geobuild и читается целиком в память (~30 МБ).
package geo

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
)

const (
	magic      = "GEOIDX1\x00"
	headerSize = len(magic) + 4*4
	placeSize  = 42
	keySize    = 10
)

// Place — населённый пункт.
type Place struct {
	ID         int32
	Name       string
	Lat        float64
	Lon        float64
	Population int64
	Timezone   string
	Country    string
}

// Index — открытый индекс GeoNames.
type Index struct {
	places []byte // секция place-записей
	keys   []byte // секция ключей
	names  []byte // нормализованные имена
	strs   []byte // строки (названия, таймзоны)

	placeCount int
	keyCount   int
}

// Open читает индекс из файла.
func Open(path string) (*Index, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < headerSize || string(b[:len(magic)]) != magic {
		return nil, fmt.Errorf("geo: %s — не файл индекса GeoNames", path)
	}
	keyCount := int(binary.LittleEndian.Uint32(b[len(magic):]))
	placeCount := int(binary.LittleEndian.Uint32(b[len(magic)+4:]))
	psize := int(binary.LittleEndian.Uint32(b[len(magic)+8:]))
	ksize := int(binary.LittleEndian.Uint32(b[len(magic)+12:]))
	if psize != placeSize || ksize != keySize {
		return nil, fmt.Errorf("geo: %s — несовместимый формат (place=%d key=%d)", path, psize, ksize)
	}

	off := headerSize
	if off+placeCount*placeSize > len(b) {
		return nil, fmt.Errorf("geo: %s — обрезанная секция мест", path)
	}
	places := b[off : off+placeCount*placeSize]
	off += placeCount * placeSize
	if off+keyCount*keySize > len(b) {
		return nil, fmt.Errorf("geo: %s — обрезанная секция ключей", path)
	}
	keys := b[off : off+keyCount*keySize]
	off += keyCount * keySize

	nameBlob, off, err := readBlob(b, off)
	if err != nil {
		return nil, fmt.Errorf("geo: %s — блок имён: %v", path, err)
	}
	strBlob, _, err := readBlob(b, off)
	if err != nil {
		return nil, fmt.Errorf("geo: %s — блок строк: %v", path, err)
	}

	return &Index{
		places: places, keys: keys, names: nameBlob, strs: strBlob,
		placeCount: placeCount, keyCount: keyCount,
	}, nil
}

func readBlob(b []byte, off int) (blob []byte, next int, err error) {
	if off+4 > len(b) {
		return nil, 0, fmt.Errorf("нет заголовка блока")
	}
	n := int(binary.LittleEndian.Uint32(b[off:]))
	off += 4
	if off+n > len(b) {
		return nil, 0, fmt.Errorf("обрезанный блок (%d байт)", n)
	}
	return b[off : off+n], off + n, nil
}

// Places возвращает количество населённых пунктов в индексе.
func (ix *Index) Places() int { return ix.placeCount }

// Keys возвращает количество поисковых имён.
func (ix *Index) Keys() int { return ix.keyCount }

func (ix *Index) place(idx int) Place {
	b := ix.places[idx*placeSize:]
	p := Place{
		ID:         int32(binary.LittleEndian.Uint32(b[0:])),
		Lat:        math.Float64frombits(binary.LittleEndian.Uint64(b[4:])),
		Lon:        math.Float64frombits(binary.LittleEndian.Uint64(b[12:])),
		Population: int64(binary.LittleEndian.Uint64(b[20:])),
		Country:    string(b[40:42]),
	}
	nameOff := int(binary.LittleEndian.Uint32(b[28:]))
	nameLen := int(binary.LittleEndian.Uint16(b[32:]))
	tzOff := int(binary.LittleEndian.Uint32(b[34:]))
	tzLen := int(binary.LittleEndian.Uint16(b[38:]))
	if nameOff+nameLen <= len(ix.strs) {
		p.Name = string(ix.strs[nameOff : nameOff+nameLen])
	}
	if tzOff+tzLen <= len(ix.strs) {
		p.Timezone = string(ix.strs[tzOff : tzOff+tzLen])
	}
	return p
}

func (ix *Index) key(i int) (name []byte, placeIdx int) {
	b := ix.keys[i*keySize:]
	nameOff := int(binary.LittleEndian.Uint32(b[0:]))
	nameLen := int(binary.LittleEndian.Uint16(b[4:]))
	placeIdx = int(binary.LittleEndian.Uint32(b[6:]))
	if nameOff+nameLen > len(ix.names) {
		return nil, placeIdx
	}
	return ix.names[nameOff : nameOff+nameLen], placeIdx
}

// Result — лучший найденный вариант и число совпадений.
type Result struct {
	Place      Place
	Matches    int     // сколько разных мест носят это имя
	Alternates []Place // до четырёх следующих по населению
}

// Lookup ищет населённый пункт по названию: точное совпадение нормализованного
// имени, при нескольких — самое населённое (Москва в России, а не Moscow, Idaho).
func (ix *Index) Lookup(city string) (Result, bool) {
	q := Normalize(city)
	if q == "" {
		return Result{}, false
	}
	key := []byte(q)
	lo := sort.Search(ix.keyCount, func(i int) bool {
		name, _ := ix.key(i)
		return bytesCompare(name, key) >= 0
	})

	var found []int
	for i := lo; i < ix.keyCount && len(found) < 64; i++ {
		name, pi := ix.key(i)
		if bytesCompare(name, key) != 0 {
			break
		}
		found = append(found, pi)
	}
	if len(found) == 0 {
		return Result{}, false
	}

	places := make([]Place, 0, len(found))
	seen := make(map[int32]bool, len(found))
	for _, pi := range found {
		if pi < 0 || pi >= ix.placeCount {
			continue
		}
		p := ix.place(pi)
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		places = append(places, p)
	}
	if len(places) == 0 {
		return Result{}, false
	}
	sort.SliceStable(places, func(i, j int) bool { return places[i].Population > places[j].Population })

	res := Result{Place: places[0], Matches: len(places)}
	if len(places) > 1 {
		n := len(places) - 1
		if n > 4 {
			n = 4
		}
		res.Alternates = places[1 : 1+n]
	}
	return res, true
}

// bytesCompare сравнивает байтовые строки лексикографически.
func bytesCompare(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// ─── сборка индекса ──────────────────────────────────────────────────────────

// SourcePlace — населённый пункт из дампа GeoNames.
type SourcePlace struct {
	ID       int32
	Name     string
	Lat, Lon float64
	Pop      int64
	Timezone string
	Country  string
	Names    []string // все варианты названия: name, asciiname, alternatenames
}

// Build собирает файл индекса из набора мест.
func Build(places []SourcePlace, outPath string) error {
	// Пул строк для названий мест и таймзон.
	strBlob := make([]byte, 0, 1<<20)
	strOff := make(map[string]int, len(places))
	addStr := func(s string) (int, int) {
		if off, ok := strOff[s]; ok {
			return off, len(s)
		}
		off := len(strBlob)
		strBlob = append(strBlob, s...)
		strOff[s] = off
		return off, len(s)
	}

	byID := make(map[int32]int, len(places))
	for i, p := range places {
		byID[p.ID] = i
	}

	placesSec := make([]byte, 0, len(places)*placeSize)
	for _, p := range places {
		nameOff, nameLen := addStr(p.Name)
		tzOff, tzLen := addStr(p.Timezone)
		cc := p.Country + "  "
		placesSec = appendU32(placesSec, uint32(p.ID))
		placesSec = appendU64(placesSec, math.Float64bits(p.Lat))
		placesSec = appendU64(placesSec, math.Float64bits(p.Lon))
		placesSec = appendU64(placesSec, uint64(p.Pop))
		placesSec = appendU32(placesSec, uint32(nameOff))
		placesSec = appendU16(placesSec, uint16(nameLen))
		placesSec = appendU32(placesSec, uint32(tzOff))
		placesSec = appendU16(placesSec, uint16(tzLen))
		placesSec = append(placesSec, cc[0], cc[1])
	}

	type keyRec struct {
		name string
		pid  int32
	}
	keys := make([]keyRec, 0, len(places)*2)
	for _, p := range places {
		seen := make(map[string]bool, len(p.Names))
		for _, n := range p.Names {
			norm := Normalize(n)
			if norm == "" || seen[norm] {
				continue
			}
			seen[norm] = true
			keys = append(keys, keyRec{name: norm, pid: p.ID})
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].name != keys[j].name {
			return keys[i].name < keys[j].name
		}
		return keys[i].pid < keys[j].pid
	})

	nameBlob := make([]byte, 0, 1<<20)
	nameOff := make(map[string]int, len(keys)/2)
	keysSec := make([]byte, 0, len(keys)*keySize)
	for _, k := range keys {
		off, ok := nameOff[k.name]
		if !ok {
			off = len(nameBlob)
			nameBlob = append(nameBlob, k.name...)
			nameOff[k.name] = off
		}
		idx, ok := byID[k.pid]
		if !ok {
			continue
		}
		keysSec = appendU32(keysSec, uint32(off))
		keysSec = appendU16(keysSec, uint16(len(k.name)))
		keysSec = appendU32(keysSec, uint32(idx))
	}
	// Число ключей в заголовке должно совпадать с фактически записанными.
	written := len(keysSec) / keySize

	buf := make([]byte, 0, headerSize+len(placesSec)+len(keysSec)+len(nameBlob)+len(strBlob)+8)
	buf = append(buf, magic...)
	buf = appendU32(buf, uint32(written))
	buf = appendU32(buf, uint32(len(places)))
	buf = appendU32(buf, placeSize)
	buf = appendU32(buf, keySize)
	buf = append(buf, placesSec...)
	buf = append(buf, keysSec...)
	buf = appendU32(buf, uint32(len(nameBlob)))
	buf = append(buf, nameBlob...)
	buf = appendU32(buf, uint32(len(strBlob)))
	buf = append(buf, strBlob...)

	return os.WriteFile(outPath, buf, 0o644)
}

func appendU16(b []byte, v uint16) []byte { return append(b, byte(v), byte(v>>8)) }

func appendU32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func appendU64(b []byte, v uint64) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}
