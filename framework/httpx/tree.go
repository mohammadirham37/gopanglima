package httpx

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type segKind uint8

const (
	segStatic segKind = iota
	segParam
	segCatchAll
)

type segment struct {
	kind  segKind
	value string // literal for static segments, parameter name otherwise
}

// parsePattern validates a full route pattern and splits it into segments.
func parsePattern(pattern string) ([]segment, []string, error) {
	if !strings.HasPrefix(pattern, "/") {
		return nil, nil, fmt.Errorf("pattern must start with \"/\"")
	}
	raw := strings.Split(pattern[1:], "/")
	segs := make([]segment, 0, len(raw))
	var names []string
	seen := make(map[string]bool)
	for i, s := range raw {
		last := i == len(raw)-1
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			name := s[1 : len(s)-1]
			kind := segParam
			if n, ok := strings.CutSuffix(name, "..."); ok {
				if !last {
					return nil, nil, fmt.Errorf("catch-all parameter %q must be the last segment", s)
				}
				name, kind = n, segCatchAll
			}
			if !validParamName(name) {
				return nil, nil, fmt.Errorf("invalid parameter name %q", name)
			}
			if seen[name] {
				return nil, nil, fmt.Errorf("duplicate parameter name %q", name)
			}
			seen[name] = true
			names = append(names, name)
			segs = append(segs, segment{kind: kind, value: name})
			continue
		}
		if strings.ContainsRune(s, '%') {
			return nil, nil, fmt.Errorf("segment %q: write static segments unescaped (no %%)", s)
		}
		if strings.ContainsAny(s, "{}") {
			return nil, nil, fmt.Errorf("segment %q: parameters must occupy a whole segment", s)
		}
		if s == "." || s == ".." {
			return nil, nil, fmt.Errorf("segment %q is not allowed", s)
		}
		if s == "" && !last {
			return nil, nil, fmt.Errorf("empty path segment (\"//\")")
		}
		segs = append(segs, segment{kind: segStatic, value: s})
	}
	return segs, names, nil
}

func validParamName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// node is a radix-like segment tree node. The tree is built once during
// Build and is read-only afterwards.
type node struct {
	static   map[string]*node
	param    *node
	routes   map[string]*Route // routes ending exactly at this node, by method
	catchAll map[string]*Route // catch-all routes rooted at this node, by method
}

func (n *node) insert(rt *Route) error {
	cur := n
	for _, s := range rt.segments {
		switch s.kind {
		case segStatic:
			if cur.static == nil {
				cur.static = make(map[string]*node)
			}
			next, ok := cur.static[s.value]
			if !ok {
				next = &node{}
				cur.static[s.value] = next
			}
			cur = next
		case segParam:
			if cur.param == nil {
				cur.param = &node{}
			}
			cur = cur.param
		case segCatchAll:
			if cur.catchAll == nil {
				cur.catchAll = make(map[string]*Route)
			}
			if prev, ok := cur.catchAll[rt.method]; ok {
				return conflict(rt, prev)
			}
			cur.catchAll[rt.method] = rt
			return nil
		}
	}
	if cur.routes == nil {
		cur.routes = make(map[string]*Route)
	}
	if prev, ok := cur.routes[rt.method]; ok {
		return conflict(rt, prev)
	}
	cur.routes[rt.method] = rt
	return nil
}

func conflict(rt, prev *Route) error {
	return fmt.Errorf("route %s %s conflicts with %s %s", rt.method, rt.pattern, prev.method, prev.pattern)
}

// lookup finds the highest-priority route for method and path segments.
// Matched parameter values are appended to *vals in path order.
func (n *node) lookup(method string, segs []string, vals *[]string) *Route {
	if len(segs) == 0 {
		return n.routes[method]
	}
	seg := segs[0]
	if child := n.static[seg]; child != nil {
		if rt := child.lookup(method, segs[1:], vals); rt != nil {
			return rt
		}
	}
	if n.param != nil && validParamValue(seg) {
		*vals = append(*vals, seg)
		if rt := n.param.lookup(method, segs[1:], vals); rt != nil {
			return rt
		}
		*vals = (*vals)[:len(*vals)-1]
	}
	if rt := n.catchAll[method]; rt != nil && validCatchAll(segs) {
		*vals = append(*vals, strings.Join(segs, "/"))
		return rt
	}
	return nil
}

// allowed collects every method that has a route matching the path.
func (n *node) allowed(segs []string, out map[string]bool) {
	if len(segs) == 0 {
		for m := range n.routes {
			out[m] = true
		}
		return
	}
	if child := n.static[segs[0]]; child != nil {
		child.allowed(segs[1:], out)
	}
	if n.param != nil && validParamValue(segs[0]) {
		n.param.allowed(segs[1:], out)
	}
	if validCatchAll(segs) {
		for m := range n.catchAll {
			out[m] = true
		}
	}
}

// allowHeader formats the Allow header value for a set of methods, adding
// the implicit HEAD (via GET) and OPTIONS methods.
func allowHeader(methods map[string]bool) string {
	if methods[http.MethodGet] {
		methods[http.MethodHead] = true
	}
	methods[http.MethodOptions] = true
	list := make([]string, 0, len(methods))
	for m := range methods {
		list = append(list, m)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

// validParamValue rejects empty values, values containing "/" (from an
// encoded %2F), and dot segments, so a single-segment parameter can never
// smuggle a path separator or "." / ".." into application code.
func validParamValue(s string) bool {
	return s != "" && s != "." && s != ".." && strings.IndexByte(s, '/') < 0
}

// validCatchAll rejects catch-all values that would contain a "." or ".."
// path element after decoding, including elements produced by an encoded
// %2F inside a segment.
func validCatchAll(segs []string) bool {
	for _, s := range segs {
		for _, part := range strings.Split(s, "/") {
			if part == "." || part == ".." {
				return false
			}
		}
	}
	return true
}
