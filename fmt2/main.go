package main 

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"unicode"
)

var flagDebug *bool = flag.Bool("debug", false, "debug messages")
var flagWidth *int = flag.Int("w", 70, "target max output columns")

/*

A tricky comment to format:

   *  Defines a total order over heap objects. This is required
   *  because the identity hash code is not unique across objects.
   *  Thus, we keep a hash map that specifies the order(s) for given
   *  collisions. We can do this because reference equality is well
   *  defined in Java. See:
   *
   *    http://gafter.blogspot.com/2007/03/compact-object-comparator.html
   *
   *  For more information.

*/

type revStringLengthSlice struct {
	sort.StringSlice
}

func (ssl revStringLengthSlice) Less(i, j int) bool {
	return len(ssl.StringSlice[j]) < len(ssl.StringSlice[i])
}

func splitPrefix(strs []string) (string, []string) {
	switch len(strs) {
	case 0:
		return "", strs
	case 1:
		l := 0
		for _, r := range strs[0] {
			if unicode.IsSpace(r) {
				l++
			} else {
				break
			}
		}
		prefix := strs[0][:l]
		strs[0] = strs[0][l:]
		return prefix, strs
	default:
	}

	sorted := make([]string, len(strs))
	copy(sorted, strs)
	sort.Sort(&revStringLengthSlice{sort.StringSlice(sorted)})

	strrs := make([][]rune, len(sorted))
	for i := range strs {
		strrs[i] = []rune(sorted[i])
	}

	i := 0
Outer:
	for ; i < len(strrs[0]); i++ {
		chr := strrs[0][i]
		for _, line := range strrs[1:] {
			if len(line) <= i {
				if unicode.IsSpace(chr) {
					continue Outer
				} else {
					break Outer
				}
			}

			if line[i] != chr {
				break Outer
			}
		}
	}

	prefix := string(strrs[0][:i])
	stripped := make([]string, len(strs))
	for i := range strs {
		if len(prefix) <= len(strs[i]) {
			stripped[i] = strs[i][len(prefix):]
		} else {
			stripped[i] = ""
		}
	}

	return prefix, stripped
}

// Group lines into paragraphs.
func splitParas(lines []string) [][]string {
	paras := make([][]string, 0)

	last := 0
	for i := range lines {
		if i == len(lines)-1 {
			if last != i || lines[i] != "" {
				paras = append(paras, lines[last:i+1])
			}

			return paras
		}

		if lines[i] != "" {
			continue
		}

		if i != last {
			paras = append(paras, lines[last:i])
		}

		last = i + 1 // Skip blank line.
	}

	panic("notreached")
}

func fmtlines(lines []string, width int) []string {
	text := ""
	for i, line := range lines {
		text += line
		if i != len(lines)-1 {
			text += " "
		}
	}

	words := strings.Split(text, " ") // TODO: any (consecutive) whitespace
	lines = make([]string, 1)
	l := 0

	for _, word := range words {
		if len(lines[l]) == 0 {
			lines[l] += word
		} else if len(lines[l])+len(word) < width-1 {
			lines[l] += " " + word
		} else {
			lines = append(lines, word)
			l++
		}
	}

	return lines
}

func main() {
	flag.Parse()

	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	text := string(bytes)
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		os.Exit(0)
	}

	// Strip a potentially lonely last line. TODO: needed?
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for i := range lines {
		lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
	}

	basePrefix, lines := splitPrefix(lines)

	paras := splitParas(lines)
	for i, para := range paras {
		if i != 0 {
			line := strings.TrimRightFunc(basePrefix, unicode.IsSpace)
			fmt.Println(line)
		}
		paraPrefix, para := splitPrefix(para)
		prefix := basePrefix + paraPrefix
		width := *flagWidth - len(prefix)
		if width < 0 {
			width = 0
		}
		para = fmtlines(para, width)
		for _, line := range para {
			line = strings.TrimRightFunc(basePrefix+paraPrefix+line, unicode.IsSpace)
			fmt.Println(line)
		}
	}
}
