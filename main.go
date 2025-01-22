package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type pangram struct {
	key   string
	words []string
}

type LengthCount struct {
	Length int `json:"length"`
	Count  int `json:"count"`
}

type Solve struct {
	Key       string        `json:"key"`
	Pangrams  []string      `json:"pangrams"`
	Lengths   []LengthCount `json:"lengths"`
	Words     []string      `json:"words"`
	KeyLetter string        `json:"keyLetter"`
}

func main() {
	fn := flag.String("f", "words.txt", "filename")
	sfx := flag.String("s", "", "suffix")
	minWords := flag.Int("min", 12, "min words")
	maxWords := flag.Int("max", 1000, "max words per length")
	flag.Parse()

	start := time.Now()

	s, err := os.ReadFile(*fn)
	if err != nil {
		log.Fatal(err)
	}
	sp := strings.Split(string(s), "\r\n")

	filtered := filter(sp)

	mins := []string{}
	for _, w := range filtered {
		if len(w) > 3 {
			mins = append(mins, w)
		}
	}

	writeLines(fmt.Sprintf("mins%s.txt", *sfx), mins)

	pangrams := getPangrams(mins)
	writeLines(fmt.Sprintf("pangrams%s.txt", *sfx), pangrams)

	unique := getUnique(pangrams)
	plines := []string{}
	for _, u := range unique {
		csv := strings.Join(u.words, ", ")
		plines = append(plines, fmt.Sprintf("%s %s", u.key, csv))
	}
	writeLines(fmt.Sprintf("unique%s.txt", *sfx), plines)

	solves := getSolves(unique, mins)
	log.Println("total solves", len(solves))
	writeJson(fmt.Sprintf("solves%s.json", *sfx), solves)

	probables := getProbableSolves(solves, *minWords, *maxWords)
	log.Println("probable solves", len(probables))
	writeJson(fmt.Sprintf("probablesolves%s.json", *sfx), probables)

	log.Println("finished.", time.Since(start))
}

func filter(words []string) []string {
	filterreg := regexp.MustCompile("^[a-z]+$")
	filtered := []string{}
	for _, w := range words {
		lw := strings.ToLower(w)
		if filterreg.MatchString(lw) {
			filtered = append(filtered, lw)
		}
	}
	return filtered
}

func getPangrams(words []string) []string {
	pangrams := []string{}
	for _, w := range words {
		m := make(map[rune]int)
		for _, c := range w {
			m[c]++
		}
		if len(m) == 7 {
			pangrams = append(pangrams, w)
		}
	}
	return pangrams
}

func getKey(w string) string {
	ch := []string{}
	cm := make(map[string]bool)
	for _, c := range w {
		cm[string(c)] = true
	}
	for k := range cm {
		ch = append(ch, k)
	}
	sort.Strings(ch)

	key := strings.Join(ch, "")
	return key
}

func getUnique(pangrams []string) []pangram {
	m := make(map[string][]string)
	for _, w := range pangrams {
		key := getKey(w)
		m[key] = append(m[key], w)
	}

	plist := []pangram{}
	for k, v := range m {
		p := pangram{
			key:   k,
			words: v,
		}
		plist = append(plist, p)
	}
	return plist
}

func getSolves(pangrams []pangram, words []string) []Solve {
	solves := []Solve{}
	for _, p := range pangrams {
		pm := make(map[string]bool)
		for _, pg := range p.words {
			pm[pg] = true
		}

		cm := make(map[rune]bool)
		for _, r := range p.key {
			cm[r] = true
		}

		valid := []string{}
		for _, w := range words {
			isValid := true
			if _, ok := pm[w]; ok {
				continue
			}

			for _, c := range w {
				if _, ok := cm[c]; !ok {
					isValid = false
					break
				}
			}
			if isValid {
				valid = append(valid, w)
			}
		}
		if len(valid) > 0 {
			valid = sortWords(valid)
			solves = append(solves, Solve{
				Key:      p.key,
				Pangrams: p.words,
				Words:    valid,
				Lengths:  getLengthCounts(p.words),
			})
		}
	}
	return solves
}

func getProbableSolves(solves []Solve, minwords, maxwords int) []Solve {
	max := 0
	nsolves := []Solve{}
	for _, s := range solves {
		for _, kl := range s.Key {
			x := getWordsWithRune(kl, s.Words)
			x = sortWords(cleanWords(x, s.Key, maxwords))
			if len(x) < minwords {
				continue
			}
			if len(x) > max {
				max = len(x)
			}
			ns := Solve{
				Key:       s.Key,
				Pangrams:  s.Pangrams,
				Words:     x,
				Lengths:   getLengthCounts(x),
				KeyLetter: string(kl),
			}
			nsolves = append(nsolves, ns)
		}
	}
	return nsolves
}

func sortWords(words []string) []string {
	sort.Slice(words, func(i, j int) bool {
		cless := words[i] < words[j]
		eqlen := len(words[i]) == len(words[j])

		if eqlen {
			return cless
		}

		return len(words[i]) < len(words[j])
	})
	return words
}

func cleanWords(words []string, key string, maxwords int) []string {
	seen := make(map[string][]string)
	lens := make(map[int][]string)
	chars := make(map[rune]int)
	for _, c := range key {
		chars[c] = 0
	}

	wchars := make(map[string]map[rune]int)
	for _, w := range words {
		wchars[w] = make(map[rune]int)
		for _, c := range w {
			wchars[w][c]++
		}
	}

	sort.Slice(words, func(i, j int) bool {
		c1 := wchars[words[i]]
		c2 := wchars[words[j]]

		return len(c1) >= len(c2)
	})

	for _, w := range words {
		if lv, ok := lens[len(w)]; ok {
			if len(lv) > maxwords {
				continue
			}
		}

		lens[len(w)] = append(lens[len(w)], w)

		k := getKey(w)
		for _, c := range k {
			chars[c]++
		}

		if v, ok := seen[k]; ok {
			skip := false
			for _, x := range v {
				if len(x) == len(w) {
					skip = true
				}
			}
			if !skip {
				seen[k] = append(seen[k], w)
			}
		} else {
			seen[k] = append(seen[k], w)
		}
	}

	zeros := 0
	anyzero := false
	for _, v := range chars {
		if v == 0 {
			anyzero = true
			zeros++
		}
	}

	if anyzero {
		log.Println(key, len(words), zeros)
		return []string{}
	}

	list := []string{}
	for _, v := range seen {
		list = append(list, v...)
	}
	return list
}

func getWordsWithRune(r rune, words []string) []string {
	with := []string{}
	for _, w := range words {
		for _, c := range w {
			if c == r {
				with = append(with, w)
				break
			}
		}
	}
	return with
}

func getLengthCounts(words []string) []LengthCount {
	lc := []LengthCount{}
	m := make(map[int]int)
	for _, w := range words {
		m[len(w)]++
	}

	for k, v := range m {
		lc = append(lc, LengthCount{Length: k, Count: v})
	}
	sort.Slice(lc, func(i, j int) bool {
		return lc[i].Length < lc[j].Length
	})
	return lc
}

func writeLines(filename string, words []string) error {
	o, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer o.Close()
	for _, w := range words {
		fmt.Fprintln(o, w)
	}
	return nil
}

func writeJson(filename string, v interface{}) error {
	o, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer o.Close()

	enc := json.NewEncoder(o)
	enc.SetIndent(" ", " ")
	return enc.Encode(v)
}
