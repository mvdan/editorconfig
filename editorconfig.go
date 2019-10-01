// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// Pakage editorconfig allows parsing and using EditorConfig files, as defined
// in https://editorconfig.org/.
package editorconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/pattern"
)

const DefaultName = ".editorconfig"

// File is an EditorConfig file with a number of sections.
type File struct {
	Root     bool
	Sections []Section
}

// Section is a single EditorConfig section, which applies a number of
// properties to the filenames matching it.
type Section struct {
	// Name is the section's name. Usually, this will be a valid pattern
	// matching string.
	Name   string
	rxName *regexp.Regexp

	// Properties is the list of name-value properties contained by a
	// section. It is kept in increasing order, to allow binary searches.
	Properties []Property
}

// Property is a single property with a name and a value, which can be
// represented as a single line.
type Property struct {
	// Name is always lowercase and allows identifying a property.
	Name string
	// Value holds data for a property.
	Value string
}

// String turns a property into its INI format.
func (p Property) String() string { return fmt.Sprintf("%s=%s", p.Name, p.Value) }

// String turns a file into its INI format.
func (f File) String() string {
	var b strings.Builder
	if f.Root {
		fmt.Fprintf(&b, "root=true\n\n")
	}
	for i, section := range f.Sections {
		if i > 0 {
			fmt.Fprintln(&b)
		}
		fmt.Fprintf(&b, "[%s]\n", section.Name)
		for _, prop := range section.Properties {
			fmt.Fprintf(&b, "%s=%s\n", prop.Name, prop.Value)
		}
	}
	return b.String()
}

// Lookup finds a property by its name within a section and returns a pointer to
// it, or nil if no such property exists.
//
// Note that most of the time, Get should be used instead.
func (s Section) Lookup(name string) *Property {
	// TODO: binary search
	for i, prop := range s.Properties {
		if prop.Name == name {
			return &s.Properties[i]
		}
	}
	return nil
}

// Get returns the value of a property found by its name. If no such property
// exists, an empty string is returned.
func (s Section) Get(name string) string {
	if prop := s.Lookup(name); prop != nil {
		return prop.Value
	}
	return ""
}

// IndentSize is a shortcut for Get("indent_size") as an int.
func (s Section) IndentSize() int {
	n, _ := strconv.Atoi(s.Get("indent_size"))
	return n
}

// IndentSize is a shortcut for Get("trim_trailing_whitespace") as a bool.
func (s Section) TrimTrailingWhitespace() bool {
	return s.Get("trim_trailing_whitespace") == "true"
}

// IndentSize is a shortcut for Get("insert_final_newline") as a bool.
func (s Section) InsertFinalNewline() bool {
	return s.Get("insert_final_newline") == "true"
}

// IndentSize is similar to Get("indent_size"), but it handles the "tab" default
// and returns an int. When unset, it returns 0.
func (s Section) TabWidth() int {
	value := s.Get("indent_size")
	if value == "tab" {
		value = s.Get("tab_width")
	}
	n, _ := strconv.Atoi(value)
	return n
}

// Add introduces a number of properties to the section. Properties that were
// already part of the section are ignored.
func (s *Section) Add(properties ...Property) {
	for _, prop := range properties {
		if s.Lookup(prop.Name) == nil {
			s.Properties = append(s.Properties, prop)
		}
	}
}

// String turns a section into its INI format.
func (s Section) String() string {
	var b strings.Builder
	if s.Name != "" {
		fmt.Fprintf(&b, "[%s]\n", s.Name)
	}
	for _, prop := range s.Properties {
		fmt.Fprintf(&b, "%s=%s\n", prop.Name, prop.Value)
	}
	return b.String()
}

// Match returns whether a file name matches the section's pattern. The name
// should be a path relative to the directory holding the EditorConfig.
//
// The underyling regular expression is built on first use and cached for later
// use.
func (s *Section) Match(name string) bool {
	if s.rxName == nil {
		pat := s.Name
		if i := strings.IndexByte(pat, '/'); i == 0 {
			pat = pat[1:]
		} else if i < 0 {
			pat = "**/" + pat
		}
		rx, err := pattern.Regexp(pat, pattern.Filenames|pattern.Braces)
		if err != nil {
			panic(err)
		}
		s.rxName = regexp.MustCompile("^" + rx + "$")
	}
	return s.rxName.MatchString(name)
}

// Filter returns a set of properties from a file that apply to a file name.
// Properties from later sections take precedence. The name should be a path
// relative to the directory holding the EditorConfig.
//
// Note that this function doesn't apply defaults; for that, see Find.
func (f *File) Filter(name string) Section {
	result := Section{}
	for i := len(f.Sections) - 1; i >= 0; i-- {
		section := f.Sections[i]
		if section.Match(name) {
			result.Add(section.Properties...)
		}
	}
	return result
}

// Find is equivalent to Query{}.Find.
func Find(name string) (Section, error) {
	return (&Query{}).Find(name)
}

// Query allows fine-grained control of how EditorConfig files are found and
// used.
type Query struct {
	// ConfigName specifies what EditorConfig file name to use when
	// searching for files on disk. If empty, it defaults to DefaultName.
	ConfigName string

	// Cache keeps track of which directories are known to contain an
	// EditorConfig.
	//
	// If nil, no caching takes place.
	Cache map[string]*File

	// Version specifies an EditorConfig version to use when applying its
	// spec. When empty, it defaults to the latest version. This field
	// should generally be left untouched.
	Version string
}

// Find figures out the properties that apply to a file name on disk, and
// returns them as a section. The name doesn't need to be an absolute path.
//
// Any relevant EditorConfig files are parsed and used as necessary. Parsing the
// files can be cached in Query.
//
// The defaults for supported properties are applied before returning.
func (q *Query) Find(name string) (Section, error) {
	name, err := filepath.Abs(name)
	if err != nil {
		return Section{}, err
	}
	configName := q.ConfigName
	if configName == "" {
		configName = DefaultName
	}

	result := Section{}
	dir := name
	for {
		if d := filepath.Dir(dir); d != dir {
			dir = d
		} else {
			break
		}
		file, e := q.Cache[dir]
		if !e {
			f, err := os.Open(filepath.Join(dir, configName))
			if os.IsNotExist(err) {
				// continue below, caching the nil file
			} else if err != nil {
				return Section{}, err
			} else {
				file_, err := Parse(f)
				f.Close()
				if err != nil {
					return Section{}, err
				}
				file = &file_
			}
			if q.Cache != nil {
				q.Cache[dir] = file
			}
		}
		if file == nil {
			continue
		}
		relative := name[len(dir)+1:]
		result.Add(file.Filter(relative).Properties...)
		if file.Root {
			break
		}
	}

	if result.Get("indent_style") == "tab" {
		if prop := result.Lookup("tab_width"); prop != nil {
			// When indent_style is "tab" and tab_width is set,
			// indent_size should default to tab_width.
			result.Add(Property{Name: "indent_size", Value: prop.Value})
		}
		if q.Version != "" && q.Version < "0.9.0" {
		} else if result.Lookup("indent_size") == nil {
			// When indent_style is "tab", indent_size defaults to
			// "tab". Only on 0.9.0 and later.
			result.Add(Property{Name: "indent_size", Value: "tab"})
		}
	} else if result.Lookup("tab_width") == nil {
		if prop := result.Lookup("indent_size"); prop != nil && prop.Value != "tab" {
			// tab_width defaults to the value of indent_size.
			result.Add(Property{Name: "tab_width", Value: prop.Value})
		}
	}
	return result, nil
}

func Parse(r io.Reader) (File, error) {
	f := File{}
	scanner := bufio.NewScanner(r)
	var section *Section
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.Index(line, " #"); i >= 0 {
			line = line[:i]
		} else if i := strings.Index(line, " ;"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)

		if len(line) > 2 && line[0] == '[' && line[len(line)-1] == ']' {
			name := line[1 : len(line)-1]
			if len(name) > 4096 {
				section = &Section{} // ignore
				continue
			}
			f.Sections = append(f.Sections, Section{Name: name})
			section = &f.Sections[len(f.Sections)-1]
			continue
		}
		i := strings.IndexAny(line, "=:")
		if i < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		value := strings.TrimSpace(line[i+1:])
		switch key {
		case "root", "indent_style", "indent_size", "tab_width", "end_of_line",
			"charset", "trim_trailing_whitespace", "insert_final_newline":
			value = strings.ToLower(value)
		}
		if len(key) > 50 || len(value) > 255 {
			continue
		}
		if section != nil {
			section.Add(Property{Name: key, Value: value})
		} else if key == "root" {
			f.Root = value == "true"
		}
	}
	return f, nil
}
