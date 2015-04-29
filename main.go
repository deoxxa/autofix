package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var LINTER_REGEXP = regexp.MustCompile(`^(.+?): line (\d+), col (\d+), (?:Warning|Error) - (.+?) \((.+?)\)$`)

type compactEntry struct {
	file        string
	line, col   int
	err, module string
}

func main() {
	var fd io.Reader

	if len(os.Args) >= 2 {
		if f, err := os.Open(os.Args[1]); err != nil {
			panic(err)
		} else {
			fd = f
		}
	} else {
		fd = os.Stdin
	}

	r := bufio.NewReader(fd)

	errors := make(map[string][]compactEntry)

	for {
		l, err := r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		m := LINTER_REGEXP.FindStringSubmatch(strings.TrimRight(l, "\n"))

		if m == nil {
			continue
		}

		line, err := strconv.ParseInt(m[2], 10, 64)
		if err != nil {
			continue
		}

		col, err := strconv.ParseInt(m[3], 10, 64)
		if err != nil {
			continue
		}

		errors[m[1]] = append(errors[m[1]], compactEntry{
			file:   m[1],
			line:   int(line),
			col:    int(col),
			err:    strings.TrimRight(m[4], "."),
			module: m[5],
		})
	}

	for f, l := range errors {
		data, err := ioutil.ReadFile(f)
		if err != nil {
			fmt.Printf("couldn't open %q\n", f)
			continue
		}

		lines := strings.Split(string(data), "\n")

		fmt.Printf("[%s]\n", f)

		for _, e := range l {
			switch {
			case e.module == "comma-dangle" && e.err == "Unexpected trailing comma":
				fmt.Printf("\naction: remove character at %d, %d\n\n", e.line, e.col)

				fmt.Printf("%s\n", lines[e.line-1])
				lines[e.line-1] = lines[e.line-1][0:e.col] + lines[e.line-1][e.col+1:]
				fmt.Printf("%s\n", lines[e.line-1])

				for i := range l {
					if l[i].line == e.line && l[i].col > e.col {
						l[i].col -= 1
					}
				}
			case e.module == "comma-dangle" && e.err == "Missing trailing comma":
				fmt.Printf("\naction: add comma at %d, %d\n\n", e.line, e.col)

				fmt.Printf("%s\n", lines[e.line-1])
				lines[e.line-1] = lines[e.line-1][0:e.col] + "," + lines[e.line-1][e.col:]
				fmt.Printf("%s\n", lines[e.line-1])

				for i := range l {
					if l[i].line == e.line && l[i].col > e.col {
						l[i].col += 1
					}
				}
			case e.module == "comma-spacing" && strings.HasPrefix(e.err, "A space is required after"), e.module == "key-spacing" && strings.HasPrefix(e.err, "Missing space before value for key"):
				fmt.Printf("\naction: add space at %d, %d\n\n", e.line, e.col)

				fmt.Printf("%s\n", lines[e.line-1])
				lines[e.line-1] = lines[e.line-1][0:e.col] + " " + lines[e.line-1][e.col:]
				fmt.Printf("%s\n", lines[e.line-1])

				for i := range l {
					if l[i].line == e.line && l[i].col > e.col {
						l[i].col += 1
					}
				}
			case e.module == "no-multi-spaces" && strings.HasPrefix(e.err, "Multiple spaces found"):
				fmt.Printf("\naction: remove spaces before %d, %d\n\n", e.line, e.col)

				// start just before the linter position
				col := e.col - 1
				// search backwards until we find a non-space character
				for lines[e.line-1][col] == ' ' {
					col--
				}
				// skip forward, back into the whitespace
				col += 1

				fmt.Printf("%s\n", lines[e.line-1])
				lines[e.line-1] = lines[e.line-1][0:col] + " " + lines[e.line-1][e.col:]
				fmt.Printf("%s\n", lines[e.line-1])

				for i := range l {
					if l[i].line == e.line && l[i].col > e.col {
						l[i].col -= (e.col - col) - 1
					}
				}
			}
		}

		fmt.Printf("\n")

		ioutil.WriteFile(f, []byte(strings.Join(lines, "\n")), 0644)
	}
}
