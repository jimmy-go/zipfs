package zipf

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Zipf type.
type Zipf struct {
	path       string
	limit      int
	out        io.Writer
	symbols    bool
	words      map[string]int64
	counts     map[int64]string
	collection []Term
	sync.RWMutex
}

// New returns a Zipf analiser.
func New(dir string, limit int, symbols bool, output io.Writer) (*Zipf, error) {
	if dir == "" {
		return nil, errors.New("empty dir")
	}
	z := &Zipf{
		path:    dir,
		limit:   limit,
		out:     output,
		symbols: symbols,
		words:   make(map[string]int64),
		counts:  make(map[int64]string),
	}
	return z, nil
}

// Run executes the file path walk and report.
func (z *Zipf) Run() error {
	if err := z.Walk(z.path); err != nil {
		return err
	}
	if err := z.Report(); err != nil {
		return err
	}
	return nil
}

// Walk read all files in dir and populate the word's count.
func (z *Zipf) Walk(dir string) error {
	err := filepath.Walk(dir, func(name string, info os.FileInfo, err error) error {
		// skip directories
		if info != nil && info.IsDir() {
			return nil
		}

		// read file
		lines, err := readLines(name)
		if err != nil {
			return err
		}

		for i := range lines {
			line := lines[i]
			// skip empty lines
			if len(line) < 1 {
				continue
			}

			// Words
			words, err := SplitWord(line)
			if err != nil {
				continue
			}
			for _, w := range words {
				if err := z.Add(w); err != nil {
					return err
				}
			}

			if z.symbols {
				// Symbols
				ss, err := SplitSymbol(line)
				if err != nil {
					continue
				}
				for _, w := range ss {
					if err := z.Add(w); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	return err
}

// Add queue words to the map of words and sums 1 to existent words.
func (z *Zipf) Add(s string) error {
	z.RLock()
	defer z.RUnlock()

	if s == "" {
		return errors.New("empty word")
	}
	count, ok := z.words[s]
	if !ok {
		z.words[s] = 1
	}
	z.words[s] = count + 1
	return nil
}

// Report report words count without order.
func (z *Zipf) Report() error {
	z.RLock()
	defer z.RUnlock()

	var i int
	for k, c := range z.words {
		i++
		if i > z.limit {
			continue
		}
		z.collection = append(z.collection, Term{Word: k, Count: c})
	}

	sort.Sort(ByCountAsc(z.collection))

	for i := range z.collection {
		x := z.collection[i]
		fmt.Fprintf(z.out, "%s %d\n", x.Word, x.Count)
	}
	return nil
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var lines []string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Text()
		if len(line) < 1 {
			continue
		}
		lines = append(lines, line)
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return lines, scan.Err()
}

// Term struct contain final struct for terms/words
type Term struct {
	Word  string `json:"word"`
	Count int64  `json:"count"`
}
