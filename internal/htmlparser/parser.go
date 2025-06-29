package htmlparser

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// Document represents a parsed HTML document with CSS selector support
type Document struct {
	root *html.Node
}

// Selection represents a collection of HTML elements
type Selection struct {
	nodes []*html.Node
}

// NewDocument parses HTML from an io.Reader
func NewDocument(r io.Reader) (*Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return &Document{root: root}, nil
}

// Find returns elements matching the CSS selector (supports classes, IDs, attributes, descendants)
func (d *Document) Find(selector string) *Selection {
	nodes := d.findNodes(d.root, selector)
	return &Selection{nodes: nodes}
}

func (d *Document) findNodes(node *html.Node, selector string) []*html.Node {
	var results []*html.Node

	if strings.Contains(selector, ",") {
		selectors := strings.Split(selector, ",")
		for _, s := range selectors {
			s = strings.TrimSpace(s)
			if s != "" {
				parts := parseSelector(s)
				d.findMatchingNodes(node, parts, &results)
			}
		}
		return results
	}

	parts := parseSelector(selector)
	d.findMatchingNodes(node, parts, &results)
	return results
}

func (d *Document) findMatchingNodes(node *html.Node, parts []selectorPart, results *[]*html.Node) {
	if len(parts) == 0 {
		return
	}
	d.matchSelectorParts(node, parts, 0, results)
}

func (d *Document) matchSelectorParts(node *html.Node, parts []selectorPart, partIndex int, results *[]*html.Node) {
	if node.Type != html.ElementNode {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			d.matchSelectorParts(child, parts, partIndex, results)
		}
		return
	}

	if d.nodeMatchesPart(node, parts[partIndex]) {
		if partIndex == len(parts)-1 {
			*results = append(*results, node)
		} else {
			d.findDescendantMatches(node, parts, partIndex+1, results)
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		d.matchSelectorParts(child, parts, partIndex, results)
	}
}

func (d *Document) findDescendantMatches(node *html.Node, parts []selectorPart, partIndex int, results *[]*html.Node) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		d.matchSelectorParts(child, parts, partIndex, results)
	}
}

func (d *Document) nodeMatchesPart(node *html.Node, part selectorPart) bool {
	if part.tag != "" && part.tag != node.Data {
		return false
	}

	for _, className := range part.classes {
		if !d.hasClass(node, className) {
			return false
		}
	}

	if part.id != "" && d.getAttr(node, "id") != part.id {
		return false
	}

	for key, value := range part.attrs {
		nodeValue := d.getAttr(node, key)
		if value == "" {
			if nodeValue == "" {
				return false
			}
		} else {
			if nodeValue != value {
				return false
			}
		}
	}

	return true
}

func (d *Document) hasClass(node *html.Node, className string) bool {
	classAttr := d.getAttr(node, "class")
	if classAttr == "" {
		return false
	}

	classes := strings.Fields(classAttr)
	for _, class := range classes {
		if class == className {
			return true
		}
	}
	return false
}

func (d *Document) getAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// Each executes a function for each element in the selection
func (s *Selection) Each(fn func(int, *Element)) {
	for i, node := range s.nodes {
		element := &Element{node: node}
		fn(i, element)
	}
}

// First returns the first element or empty element if selection is empty
func (s *Selection) First() *Element {
	if len(s.nodes) == 0 {
		return &Element{node: nil}
	}
	return &Element{node: s.nodes[0]}
}

// Find searches within the current selection
func (s *Selection) Find(selector string) *Selection {
	var results []*html.Node

	for _, node := range s.nodes {
		doc := &Document{root: node}
		matches := doc.Find(selector).nodes
		results = append(results, matches...)
	}

	return &Selection{nodes: results}
}

// Len returns the number of elements in the selection
func (s *Selection) Len() int {
	return len(s.nodes)
}

// Get returns the element at index or empty element if out of bounds
func (s *Selection) Get(index int) *Element {
	if index < 0 || index >= len(s.nodes) {
		return &Element{node: nil}
	}
	return &Element{node: s.nodes[index]}
}

// Last returns the last element or empty element if selection is empty
func (s *Selection) Last() *Element {
	if len(s.nodes) == 0 {
		return &Element{node: nil}
	}
	return &Element{node: s.nodes[len(s.nodes)-1]}
}

// Filter returns a new selection with elements matching the predicate
func (s *Selection) Filter(fn func(*Element) bool) *Selection {
	var filtered []*html.Node
	for _, node := range s.nodes {
		element := &Element{node: node}
		if fn(element) {
			filtered = append(filtered, node)
		}
	}
	return &Selection{nodes: filtered}
}

// Element wraps an HTML node with convenience methods
type Element struct {
	node *html.Node
}

// Find searches within this element
func (e *Element) Find(selector string) *Selection {
	if e.node == nil {
		return &Selection{nodes: []*html.Node{}}
	}

	doc := &Document{root: e.node}
	return doc.Find(selector)
}

// Attr returns the attribute value and whether it exists
func (e *Element) Attr(key string) (string, bool) {
	if e.node == nil {
		return "", false
	}

	for _, attr := range e.node.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

// Text returns the combined text content of the element
func (e *Element) Text() string {
	if e.node == nil {
		return ""
	}
	return e.extractText(e.node)
}

func (e *Element) extractText(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}

	var text strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		text.WriteString(e.extractText(child))
	}
	return text.String()
}

// HasClass checks if the element has the specified CSS class
func (e *Element) HasClass(className string) bool {
	if e.node == nil {
		return false
	}

	classAttr := ""
	for _, attr := range e.node.Attr {
		if attr.Key == "class" {
			classAttr = attr.Val
			break
		}
	}

	if classAttr == "" {
		return false
	}

	classes := strings.Fields(classAttr)
	for _, class := range classes {
		if class == className {
			return true
		}
	}
	return false
}

// TagName returns the HTML tag name
func (e *Element) TagName() string {
	if e.node == nil {
		return ""
	}
	return e.node.Data
}

// Exists returns true if the element is not nil
func (e *Element) Exists() bool {
	return e.node != nil
}

type selectorPart struct {
	tag     string
	classes []string
	id      string
	attrs   map[string]string
}

// parseSelector splits CSS selector into descendant parts
func parseSelector(selector string) []selectorPart {
	parts := strings.Fields(selector)
	var result []selectorPart

	for _, part := range parts {
		if part == "" {
			continue
		}
		result = append(result, parseSingleSelectorPart(part))
	}

	return result
}

// parseSingleSelectorPart parses one CSS selector part (tag.class#id[attr=value])
func parseSingleSelectorPart(part string) selectorPart {
	sp := selectorPart{
		attrs:   make(map[string]string),
		classes: []string{},
	}

	remaining := part

	// Parse attribute selectors [attr=value] or [attr]
	for strings.Contains(remaining, "[") {
		start := strings.Index(remaining, "[")
		end := strings.Index(remaining[start:], "]")
		if end == -1 {
			break
		}
		end += start

		attrPart := remaining[start+1 : end]
		if strings.Contains(attrPart, "=") {
			attrParts := strings.SplitN(attrPart, "=", 2)
			key := strings.TrimSpace(attrParts[0])
			value := strings.Trim(strings.TrimSpace(attrParts[1]), "\"'")
			sp.attrs[key] = value
		} else {
			key := strings.TrimSpace(attrPart)
			sp.attrs[key] = ""
		}

		remaining = remaining[:start] + remaining[end+1:]
	}

	// Parse ID selector #id
	if strings.Contains(remaining, "#") {
		parts := strings.SplitN(remaining, "#", 2)
		remaining = parts[0]
		if len(parts) > 1 {
			idPart := parts[1]
			var id strings.Builder
			for _, r := range idPart {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
					id.WriteRune(r)
				} else {
					break
				}
			}
			sp.id = id.String()
		}
	}

	// Parse class selectors .class1.class2
	if strings.Contains(remaining, ".") {
		parts := strings.Split(remaining, ".")
		remaining = parts[0]

		for i := 1; i < len(parts); i++ {
			className := parts[i]
			if className != "" {
				sp.classes = append(sp.classes, className)
			}
		}
	}

	// Remaining part is the tag name
	if remaining != "" {
		sp.tag = remaining
	}

	return sp
}
