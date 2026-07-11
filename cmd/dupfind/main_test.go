package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseFile parses a Go source file and returns the AST.
func parseFile(t *testing.T, path string) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return file
}

// findFuncDecl returns the first FuncDecl with the given name in the file.
func findFuncDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Block extraction
// ---------------------------------------------------------------------------

func TestExtractBlocksMinNodes(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		minNodes int
		minLines int
		want     int
	}{
		{name: "exact copy_a has 1 block >=20 nodes, >=5 lines", path: "testdata/exact/copy_a.go", minNodes: 20, minLines: 5, want: 1},
		{name: "exact copy_a has 0 blocks >=100 nodes", path: "testdata/exact/copy_a.go", minNodes: 100, minLines: 5, want: 0},
		{name: "exact copy_a has 0 blocks >=1 line", path: "testdata/exact/copy_a.go", minNodes: 20, minLines: 100, want: 0},
		{name: "falsepos has 2 blocks >=20 nodes", path: "testdata/falsepos/expected.go", minNodes: 20, minLines: 5, want: 2},
		{name: "falsepos has 0 blocks >=60 nodes", path: "testdata/falsepos/expected.go", minNodes: 60, minLines: 5, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files, err := collectGoFiles(".", false)
			if err != nil {
				t.Fatal(err)
			}
			var filtered []string
			for _, f := range files {
				if strings.Contains(f, tc.path) {
					filtered = append(filtered, f)
				}
			}
			if len(filtered) == 0 {
				t.Fatalf("no files matching %s", tc.path)
			}
			blocks := extractBlocks(filtered, tc.minNodes, tc.minLines)
			if len(blocks) != tc.want {
				t.Errorf("got %d blocks, want %d", len(blocks), tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Exact duplicate detection
// ---------------------------------------------------------------------------

func TestExactDuplicates(t *testing.T) {
	files, err := collectGoFiles("testdata/exact", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)
	groups := findExactDuplicates(blocks, 2)

	if len(groups) != 1 {
		t.Fatalf("got %d exact dup groups, want 1", len(groups))
	}
	if len(groups[0]) != 2 {
		t.Errorf("got %d copies in group, want 2", len(groups[0]))
	}
	if groups[0][0].NodeCount != groups[0][1].NodeCount {
		t.Errorf("exact dups should have same node count: %d vs %d",
			groups[0][0].NodeCount, groups[0][1].NodeCount)
	}
	if groups[0][0].Sig != groups[0][1].Sig {
		t.Errorf("exact dups should have identical signatures")
	}
}

func TestNoExactDuplicateInFalsePos(t *testing.T) {
	files, err := collectGoFiles("testdata/falsepos", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)
	groups := findExactDuplicates(blocks, 2)

	if len(groups) != 0 {
		t.Errorf("got %d exact dup groups in falsepos, want 0", len(groups))
	}
}

// ---------------------------------------------------------------------------
// Near duplicate detection
// ---------------------------------------------------------------------------

func TestNearDuplicates(t *testing.T) {
	files, err := collectGoFiles("testdata/near", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)
	pairs := findNearDuplicates(blocks, 0.75)

	if len(pairs) != 1 {
		t.Fatalf("got %d near-dup pairs, want 1", len(pairs))
	}
	if pairs[0].sim < 0.75 {
		t.Errorf("similarity %.2f below threshold 0.75", pairs[0].sim)
	}
}

func TestNoNearDuplicateInFalsePos(t *testing.T) {
	files, err := collectGoFiles("testdata/falsepos", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)
	pairs := findNearDuplicates(blocks, 0.75)

	if len(pairs) != 0 {
		t.Errorf("got %d near-dup pairs in falsepos, want 0", len(pairs))
		for _, p := range pairs {
			t.Errorf("  false positive: %s:%d vs %s:%d sim=%.2f",
				p.a.File, p.a.StartLine, p.b.File, p.b.StartLine, p.sim)
		}
	}
}

// ---------------------------------------------------------------------------
// Signature computation: identifier preservation
// ---------------------------------------------------------------------------

func TestSignaturePreservesIdentNames(t *testing.T) {
	file := parseFile(t, "testdata/exact/copy_a.go")
	fn := findFuncDecl(file, "processFile")
	if fn == nil {
		t.Fatal("processFile not found")
	}
	sig, _ := computeSignature(fn.Body)

	if !strings.Contains(sig, "id:data(") {
		t.Errorf("signature should preserve identifier 'data', got: %s", truncate(sig, 200))
	}
	if !strings.Contains(sig, "id:count(") {
		t.Errorf("signature should preserve identifier 'count', got: %s", truncate(sig, 200))
	}
	if !strings.Contains(sig, "id:ReadFile(") {
		t.Errorf("signature should preserve call target 'ReadFile', got: %s", truncate(sig, 200))
	}
}

func TestSignatureDoesNotNormalizeAllIdentsToID(t *testing.T) {
	file := parseFile(t, "testdata/falsepos/expected.go")
	fnA := findFuncDecl(file, "doSomething")
	fnB := findFuncDecl(file, "doSomethingElse")
	if fnA == nil || fnB == nil {
		t.Fatal("functions not found")
	}
	sigA, _ := computeSignature(fnA.Body)
	sigB, _ := computeSignature(fnB.Body)

	if sigA == sigB {
		t.Error("unrelated functions should not have identical signatures")
	}
	if !strings.Contains(sigA, "id:data(") {
		t.Errorf("doSomething sig should contain id:data, got: %s", truncate(sigA, 200))
	}
	if !strings.Contains(sigB, "id:value(") {
		t.Errorf("doSomethingElse sig should contain id:value, got: %s", truncate(sigB, 200))
	}
}

// ---------------------------------------------------------------------------
// Signature computation: literal kind preservation
// ---------------------------------------------------------------------------

func TestSignaturePreservesLiteralKinds(t *testing.T) {
	file := parseFile(t, "testdata/exact/copy_a.go")
	fn := findFuncDecl(file, "processFile")
	if fn == nil {
		t.Fatal("processFile not found")
	}
	sig, _ := computeSignature(fn.Body)

	if !strings.Contains(sig, "lit:STRING(") {
		t.Errorf("signature should preserve STRING literal kind, got: %s", truncate(sig, 200))
	}
	if !strings.Contains(sig, "lit:INT(") {
		t.Errorf("signature should preserve INT literal kind, got: %s", truncate(sig, 200))
	}
}

func TestSignatureDistinguishesLiteralKinds(t *testing.T) {
	src := `package main

func useString() int { return len("hello") }
func useInt() int { return 42 }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	fnStr := findFuncDecl(file, "useString")
	fnInt := findFuncDecl(file, "useInt")
	sigStr, _ := computeSignature(fnStr.Body)
	sigInt, _ := computeSignature(fnInt.Body)

	if strings.Contains(sigInt, "lit:STRING") {
		t.Errorf("useInt sig should not contain STRING literal, got: %s", sigInt)
	}
	if !strings.Contains(sigStr, "lit:STRING") {
		t.Errorf("useString sig should contain STRING literal, got: %s", sigStr)
	}
	if !strings.Contains(sigInt, "lit:INT") {
		t.Errorf("useInt sig should contain INT literal, got: %s", sigInt)
	}
}

// ---------------------------------------------------------------------------
// IDF weighting
// ---------------------------------------------------------------------------

func TestIDFWeightsBoilerplateLow(t *testing.T) {
	files, err := collectGoFiles("testdata/near", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)
	if len(blocks) < 3 {
		t.Fatalf("need at least 3 blocks for IDF test, got %d", len(blocks))
	}
	idf := computeIDF(blocks)

	var boilerplateShingle, rareShingle string
	for s := range blocks[0].Shingles {
		df := 0
		for _, b := range blocks {
			if b.Shingles[s] {
				df++
			}
		}
		if df == len(blocks) && boilerplateShingle == "" {
			boilerplateShingle = s
		}
		if df == 1 && rareShingle == "" {
			rareShingle = s
		}
	}
	if boilerplateShingle == "" {
		t.Fatal("could not find a boilerplate shingle present in all blocks")
	}
	if rareShingle == "" {
		t.Fatal("could not find a rare shingle present in only 1 block")
	}

	bpWeight := idf[boilerplateShingle]
	rareWeight := idf[rareShingle]
	if bpWeight >= rareWeight {
		t.Errorf("boilerplate weight %.2f should be less than rare weight %.2f",
			bpWeight, rareWeight)
	}
}

func TestWeightedJaccardSuppressesBoilerplate(t *testing.T) {
	files, err := collectGoFiles("testdata/falsepos", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)
	if len(blocks) < 2 {
		t.Fatal("need at least 2 blocks")
	}
	idf := computeIDF(blocks)

	sim := weightedJaccard(blocks[0].Shingles, blocks[1].Shingles, idf)
	if sim >= 0.75 {
		t.Errorf("falsepos pair should have weighted similarity < 0.75, got %.2f", sim)
	}
}

// ---------------------------------------------------------------------------
// Shingling
// ---------------------------------------------------------------------------

func TestShingleSet(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		k      int
		want   int
	}{
		{name: "k=4, 6 tokens → 3 shingles", tokens: []string{"a", "b", "c", "d", "e", "f"}, k: 4, want: 3},
		{name: "k=4, 4 tokens → 1 shingle", tokens: []string{"a", "b", "c", "d"}, k: 4, want: 1},
		{name: "k=4, 3 tokens → 1 shingle (short)", tokens: []string{"a", "b", "c"}, k: 4, want: 1},
		{name: "k=4, 0 tokens → 1 shingle (empty)", tokens: []string{}, k: 4, want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			set := shingleSet(tc.tokens, tc.k)
			if len(set) != tc.want {
				t.Errorf("got %d shingles, want %d", len(set), tc.want)
			}
		})
	}
}

func TestShingleSetUnique(t *testing.T) {
	tokens := []string{"x", "x", "x", "x", "x", "x", "x", "x"}
	set := shingleSet(tokens, 4)
	if len(set) != 1 {
		t.Errorf("repeated tokens should produce 1 unique shingle, got %d", len(set))
	}
}

// ---------------------------------------------------------------------------
// File collection
// ---------------------------------------------------------------------------

func TestCollectGoFilesExcludesTests(t *testing.T) {
	files, err := collectGoFiles("testdata", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			t.Errorf("test file should be excluded: %s", f)
		}
	}
}

func TestCollectGoFilesIncludesTests(t *testing.T) {
	files, err := collectGoFiles(".", true)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no _test.go files found with includeTests=true")
	}
}

// ---------------------------------------------------------------------------
// Node counting
// ---------------------------------------------------------------------------

func TestCountNodes(t *testing.T) {
	src := `package main

func simple() {
	x := 1
	_ = x
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	fn := findFuncDecl(file, "simple")
	if fn == nil {
		t.Fatal("simple not found")
	}
	n := countNodes(fn.Body)
	if n < 5 {
		t.Errorf("expected at least 5 nodes in function body, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// funcName
// ---------------------------------------------------------------------------

func TestFuncName(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "plain function",
			src:  `package p; func foo() {}`,
			want: "foo",
		},
		{
			name: "value receiver method",
			src:  `package p; type T struct{}; func (t T) bar() {}`,
			want: "T.bar",
		},
		{
			name: "pointer receiver method",
			src:  `package p; type T struct{}; func (t *T) baz() {}`,
			want: "*T.baz",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tc.src, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					got := funcName(fn)
					if got != tc.want {
						t.Errorf("got %q, want %q", got, tc.want)
					}
					return
				}
			}
			t.Fatal("no FuncDecl found")
		})
	}
}

// ---------------------------------------------------------------------------
// Sibling case filtering
// ---------------------------------------------------------------------------

func TestSiblingsNotReportedAsNearDups(t *testing.T) {
	files, err := collectGoFiles("testdata/siblings", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 5, 2)
	pairs := findNearDuplicates(blocks, 0.50)

	for _, p := range pairs {
		if siblings(p.a, p.b) {
			t.Errorf("sibling cases should not be reported: %s:%d vs %s:%d sim=%.2f",
				p.a.File, p.a.StartLine, p.b.File, p.b.StartLine, p.sim)
		}
	}
}

func TestSiblingsFunction(t *testing.T) {
	files, err := collectGoFiles("testdata/siblings", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 5, 2)

	var caseBlocks []Block
	for _, b := range blocks {
		if b.Name == "case" {
			caseBlocks = append(caseBlocks, b)
		}
	}
	if len(caseBlocks) < 2 {
		t.Fatalf("need at least 2 case blocks, got %d", len(caseBlocks))
	}

	if !siblings(caseBlocks[0], caseBlocks[1]) {
		t.Errorf("case clauses in same switch should be siblings")
	}
}

// ---------------------------------------------------------------------------
// Min-copies threshold
// ---------------------------------------------------------------------------

func TestMinCopiesFiltersTwoCopyGroups(t *testing.T) {
	files, err := collectGoFiles("testdata/exact", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)

	if got := len(findExactDuplicates(blocks, 2)); got != 1 {
		t.Errorf("min-copies=2: got %d groups, want 1", got)
	}
	if got := len(findExactDuplicates(blocks, 3)); got != 0 {
		t.Errorf("min-copies=3: got %d groups, want 0 (only 2 copies)", got)
	}
}

func TestMinCopiesDetectsThreeCopies(t *testing.T) {
	files, err := collectGoFiles("testdata/mincopies", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 20, 5)

	if got := len(findExactDuplicates(blocks, 3)); got != 1 {
		t.Fatalf("min-copies=3: got %d groups, want 1", got)
	}
	if len(findExactDuplicates(blocks, 3)[0]) != 3 {
		t.Errorf("group should have 3 copies")
	}
}

// ---------------------------------------------------------------------------
// Line-span filtering
// ---------------------------------------------------------------------------

func TestLineSpanFiltersShortBlocks(t *testing.T) {
	files, err := collectGoFiles("testdata/exact", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 1, 100)
	if len(blocks) != 0 {
		t.Errorf("min-lines=100 should filter all blocks in exact testdata, got %d", len(blocks))
	}
}

func TestLineSpanOnExtractedBlocks(t *testing.T) {
	files, err := collectGoFiles("testdata/exact", false)
	if err != nil {
		t.Fatal(err)
	}
	blocks := extractBlocks(files, 1, 1)
	if len(blocks) == 0 {
		t.Fatal("expected at least 1 block with min thresholds")
	}
	for _, b := range blocks {
		if b.LineSpan < 1 {
			t.Errorf("block %s:%d has LineSpan %d, want >= 1", b.File, b.StartLine, b.LineSpan)
		}
	}
}
