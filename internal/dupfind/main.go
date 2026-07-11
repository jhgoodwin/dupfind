// dupfind: find duplicate and copied code blocks using go/ast tree signatures.
//
// Tree signatures are structural serializations of AST subtrees.
// Identifier names are preserved (id:name), literal kinds are preserved
// (lit:STRING, lit:INT, etc.), and operators are preserved.
// Two blocks with the same signature are structurally identical (exact dup).
// Near-duplicates use IDF-weighted Jaccard similarity on k-shingle token
// streams, so common Go boilerplate shingles contribute little weight.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const shingleK = 4

// Block is a code block with its tree signature and metadata.
type Block struct {
	File        string
	Name        string
	StartLine   int
	EndLine     int
	LineSpan    int
	NodeCount   int
	ParentStart int
	ParentEnd   int
	Sig         string
	Shingles    map[string]bool
}

type nearPair struct {
	a, b Block
	sim  float64
}

func main() {
	root := flag.String("root", ".", "root directory to scan")
	minNodes := flag.Int("min-nodes", 30, "minimum AST node count for a block")
	minLines := flag.Int("min-lines", 5, "minimum line span for a block")
	minCopies := flag.Int("min-copies", 3, "minimum copy count for exact duplicates")
	simThreshold := flag.Float64("sim", 0.75, "Jaccard similarity threshold for near-duplicates")
	includeTests := flag.Bool("tests", false, "include _test.go files")
	showSig := flag.Bool("sig", false, "show tree signatures in output")
	flag.Parse()

	files, err := collectGoFiles(*root, *includeTests)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	blocks := extractBlocks(files, *minNodes, *minLines)
	if len(blocks) == 0 {
		fmt.Println("no blocks found above minimum node/line count")
		return
	}

	fmt.Printf("scanned %d files, extracted %d blocks (min %d nodes, %d lines)\n\n",
		len(files), len(blocks), *minNodes, *minLines)

	exactGroups := findExactDuplicates(blocks, *minCopies)
	reportExactDuplicates(exactGroups, *showSig)

	nearPairs := findNearDuplicates(blocks, *simThreshold)
	reportNearDuplicates(nearPairs, *simThreshold, *showSig)
}

// ---------------------------------------------------------------------------
// File collection
// ---------------------------------------------------------------------------

func collectGoFiles(root string, includeTests bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", ".gocache", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// ---------------------------------------------------------------------------
// Block extraction
// ---------------------------------------------------------------------------

func extractBlocks(files []string, minNodes, minLines int) []Block {
	fset := token.NewFileSet()
	var blocks []Block

	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			continue
		}

		v := &blockVisitor{
			fset:     fset,
			path:     path,
			minNodes: minNodes,
			minLines: minLines,
			blocks:   &blocks,
		}
		ast.Walk(v, file)
	}
	return blocks
}

func addBlock(blocks *[]Block, file, name string, node ast.Node, fset *token.FileSet,
	minNodes, minLines, parentStart, parentEnd int) {
	b := makeBlock(file, name, node, fset, parentStart, parentEnd)
	if b.NodeCount >= minNodes && b.LineSpan >= minLines {
		*blocks = append(*blocks, b)
	}
}

type parentRange struct {
	start, end int
}

type blockVisitor struct {
	fset     *token.FileSet
	path     string
	minNodes int
	minLines int
	blocks   *[]Block
}

func (v *blockVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	switch x := n.(type) {
	case *ast.FuncDecl:
		if x.Body != nil {
			addBlock(v.blocks, v.path, funcName(x), x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
	case *ast.FuncLit:
		if x.Body != nil {
			addBlock(v.blocks, v.path, "func-lit", x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
	case *ast.IfStmt:
		if x.Body != nil {
			addBlock(v.blocks, v.path, "if", x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
		if eb, ok := x.Else.(*ast.BlockStmt); ok {
			addBlock(v.blocks, v.path, "else", eb, v.fset, v.minNodes, v.minLines, 0, 0)
		}
	case *ast.ForStmt:
		if x.Body != nil {
			addBlock(v.blocks, v.path, "for", x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
	case *ast.RangeStmt:
		if x.Body != nil {
			addBlock(v.blocks, v.path, "range", x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
	case *ast.SwitchStmt:
		if x.Body != nil {
			addBlock(v.blocks, v.path, "switch", x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
		return &caseVisitor{bv: v, start: v.fset.Position(n.Pos()).Line, end: v.fset.Position(n.End()).Line}
	case *ast.TypeSwitchStmt:
		if x.Body != nil {
			addBlock(v.blocks, v.path, "type-switch", x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
		return &caseVisitor{bv: v, start: v.fset.Position(n.Pos()).Line, end: v.fset.Position(n.End()).Line}
	case *ast.SelectStmt:
		if x.Body != nil {
			addBlock(v.blocks, v.path, "select", x.Body, v.fset, v.minNodes, v.minLines, 0, 0)
		}
		return &caseVisitor{bv: v, start: v.fset.Position(n.Pos()).Line, end: v.fset.Position(n.End()).Line}
	case *ast.CaseClause:
		addBlock(v.blocks, v.path, "case", x, v.fset, v.minNodes, v.minLines, 0, 0)
	case *ast.CommClause:
		addBlock(v.blocks, v.path, "comm", x, v.fset, v.minNodes, v.minLines, 0, 0)
	}
	return v
}

// caseVisitor wraps blockVisitor for switch/select bodies, injecting parent
// range into case/comm clauses and restoring the original visitor afterward.
type caseVisitor struct {
	bv         *blockVisitor
	start, end int
}

func (cv *caseVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	switch x := n.(type) {
	case *ast.CaseClause:
		addBlock(cv.bv.blocks, cv.bv.path, "case", x, cv.bv.fset, cv.bv.minNodes, cv.bv.minLines, cv.start, cv.end)
		return cv.bv // walk case body with blockVisitor to extract nested blocks
	case *ast.CommClause:
		addBlock(cv.bv.blocks, cv.bv.path, "comm", x, cv.bv.fset, cv.bv.minNodes, cv.bv.minLines, cv.start, cv.end)
		return cv.bv
	}
	return cv
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := ""
		if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
			recv = ident.Name + "."
		} else if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
			if ident, ok := star.X.(*ast.Ident); ok {
				recv = "*" + ident.Name + "."
			}
		}
		return recv + fn.Name.Name
	}
	return fn.Name.Name
}

func makeBlock(file, name string, node ast.Node, fset *token.FileSet, parentStart, parentEnd int) Block {
	sig, tokens := computeSignature(node)
	startLine := fset.Position(node.Pos()).Line
	endLine := fset.Position(node.End()).Line
	return Block{
		File:        file,
		Name:        name,
		StartLine:   startLine,
		EndLine:     endLine,
		LineSpan:    endLine - startLine + 1,
		NodeCount:   countNodes(node),
		ParentStart: parentStart,
		ParentEnd:   parentEnd,
		Sig:         sig,
		Shingles:    shingleSet(tokens, shingleK),
	}
}

// ---------------------------------------------------------------------------
// Tree signature computation
// ---------------------------------------------------------------------------

// computeSignature walks the AST and produces a structural string plus a
// flat token sequence (used for k-shingling). Identifier names are preserved
// as id:name, literal kinds as lit:KIND, and operators are preserved.
func computeSignature(n ast.Node) (string, []string) {
	var sb strings.Builder
	var tokens []string
	v := &sigVisitor{sb: &sb, tokens: &tokens}
	ast.Walk(v, n)
	return sb.String(), tokens
}

type sigVisitor struct {
	sb     *strings.Builder
	tokens *[]string
}

func (v *sigVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		v.sb.WriteString(")")
		return nil
	}
	name := nodeName(n)
	v.sb.WriteString(name)
	v.sb.WriteString("(")
	*v.tokens = append(*v.tokens, name)
	return v
}

func nodeName(n ast.Node) string {
	switch x := n.(type) {
	case *ast.Ident:
		return "id:" + x.Name
	case *ast.BasicLit:
		return "lit:" + x.Kind.String()
	case *ast.BinaryExpr:
		return "bin" + x.Op.String()
	case *ast.UnaryExpr:
		return "un" + x.Op.String()
	case *ast.AssignStmt:
		return "asgn" + x.Tok.String()
	case *ast.IncDecStmt:
		return "incdec" + x.Tok.String()
	case *ast.BranchStmt:
		return "br" + x.Tok.String()
	case *ast.GenDecl:
		return "gen" + x.Tok.String()
	default:
		return strings.TrimPrefix(fmt.Sprintf("%T", n), "*ast.")
	}
}

func countNodes(n ast.Node) int {
	count := 0
	ast.Inspect(n, func(node ast.Node) bool {
		if node != nil {
			count++
		}
		return true
	})
	return count
}

// ---------------------------------------------------------------------------
// K-shingling for near-duplicate detection
// ---------------------------------------------------------------------------

func shingleSet(tokens []string, k int) map[string]bool {
	set := make(map[string]bool)
	if len(tokens) < k {
		set[strings.Join(tokens, ",")] = true
		return set
	}
	for i := 0; i <= len(tokens)-k; i++ {
		set[strings.Join(tokens[i:i+k], ",")] = true
	}
	return set
}

// ---------------------------------------------------------------------------
// Exact duplicates
// ---------------------------------------------------------------------------

func findExactDuplicates(blocks []Block, minCopies int) [][]Block {
	groups := make(map[string][]Block)
	for _, b := range blocks {
		groups[b.Sig] = append(groups[b.Sig], b)
	}
	var result [][]Block
	for _, g := range groups {
		if len(g) >= minCopies {
			sort.Slice(g, func(i, j int) bool {
				if g[i].File != g[j].File {
					return g[i].File < g[j].File
				}
				return g[i].StartLine < g[j].StartLine
			})
			result = append(result, g)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return len(result[i]) > len(result[j])
	})
	return result
}

func reportExactDuplicates(groups [][]Block, showSig bool) {
	fmt.Printf("=== Exact Duplicates (%d groups) ===\n\n", len(groups))
	for i, g := range groups {
		fmt.Printf("[%d] %d copies, %d nodes:\n", i+1, len(g), g[0].NodeCount)
		for _, b := range g {
			fmt.Printf("    %s:%d-%d  %s\n", b.File, b.StartLine, b.EndLine, b.Name)
		}
		if showSig {
			fmt.Printf("    sig: %s\n", truncate(g[0].Sig, 120))
		}
		fmt.Println()
	}
}

// ---------------------------------------------------------------------------
// Near duplicates (copied code)
// ---------------------------------------------------------------------------

func findNearDuplicates(blocks []Block, threshold float64) []nearPair {
	idf := computeIDF(blocks)

	// Inverted index: shingle → block indices
	index := make(map[string][]int)
	for i, b := range blocks {
		for s := range b.Shingles {
			index[s] = append(index[s], i)
		}
	}

	// Candidate pairs: blocks sharing at least one shingle
	candidates := make(map[[2]int]bool)
	for _, indices := range index {
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				a, b := indices[i], indices[j]
				if blocks[a].Sig == blocks[b].Sig {
					continue // exact duplicate, already reported
				}
				if contains(blocks[a], blocks[b]) || contains(blocks[b], blocks[a]) {
					continue // nested blocks within same region
				}
				if siblings(blocks[a], blocks[b]) {
					continue // sibling cases within same switch/select
				}
				if a > b {
					a, b = b, a
				}
				candidates[[2]int{a, b}] = true
			}
		}
	}

	var pairs []nearPair
	for pair := range candidates {
		a, b := pair[0], pair[1]
		sim := weightedJaccard(blocks[a].Shingles, blocks[b].Shingles, idf)
		if sim >= threshold {
			pairs = append(pairs, nearPair{a: blocks[a], b: blocks[b], sim: sim})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].sim > pairs[j].sim
	})
	return pairs
}

func contains(a, b Block) bool {
	return a.File == b.File && a.StartLine <= b.StartLine && a.EndLine >= b.EndLine
}

// siblings returns true when two case/comm clauses belong to the same
// switch or select statement. Sibling cases are structurally similar by
// design and should not be reported as near-duplicates.
func siblings(a, b Block) bool {
	if a.File != b.File {
		return false
	}
	if a.ParentStart == 0 || b.ParentStart == 0 {
		return false
	}
	return a.ParentStart == b.ParentStart && a.ParentEnd == b.ParentEnd
}

// computeIDF returns inverse document frequency for each shingle across
// all blocks. Shingles appearing in many blocks (boilerplate) get low
// weight; rare shingles get high weight.
func computeIDF(blocks []Block) map[string]float64 {
	n := float64(len(blocks))
	df := make(map[string]int)
	for _, b := range blocks {
		for s := range b.Shingles {
			df[s]++
		}
	}
	idf := make(map[string]float64)
	for s, count := range df {
		idf[s] = 1 + math.Log(n/float64(count))
	}
	return idf
}

// weightedJaccard computes IDF-weighted Jaccard similarity between two
// shingle sets. Boilerplate shingles that appear in most blocks contribute
// near-zero weight, so shared Go idioms don't inflate similarity.
func weightedJaccard(a, b map[string]bool, idf map[string]float64) float64 {
	var interWeight, unionWeight float64
	for s := range a {
		w := idf[s]
		if b[s] {
			interWeight += w
		}
		unionWeight += w
	}
	for s := range b {
		if !a[s] {
			unionWeight += idf[s]
		}
	}
	if unionWeight == 0 {
		return 0
	}
	return interWeight / unionWeight
}

func reportNearDuplicates(pairs []nearPair, threshold float64, showSig bool) {
	fmt.Printf("=== Near Duplicates (%d pairs, similarity >= %.2f) ===\n\n", len(pairs), threshold)
	for i, p := range pairs {
		fmt.Printf("[%d] similarity: %.2f\n", i+1, p.sim)
		fmt.Printf("    %s:%d-%d  %s (%d nodes)\n", p.a.File, p.a.StartLine, p.a.EndLine, p.a.Name, p.a.NodeCount)
		fmt.Printf("    %s:%d-%d  %s (%d nodes)\n", p.b.File, p.b.StartLine, p.b.EndLine, p.b.Name, p.b.NodeCount)
		if showSig {
			fmt.Printf("    sig-a: %s\n", truncate(p.a.Sig, 120))
			fmt.Printf("    sig-b: %s\n", truncate(p.b.Sig, 120))
		}
		fmt.Println()
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
