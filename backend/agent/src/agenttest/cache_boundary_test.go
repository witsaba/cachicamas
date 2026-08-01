// Tests for AI-11.3 — the advisory contract: a translator that ignores
// every cache-boundary marker still translates a request fully and
// unchanged, and the usage-side surface this milestone adds nothing to
// stays untouched.
//
// Lives here rather than in ai_test, for the same reason
// agenttest/request_test.go gives AI-10.5: the consumer a marker exists for
// is an adapter in a vendor package, and this package is where cross-package
// readability is already proven for AI-06 and AI-10.
package agenttest_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// buildTranslatorFixture constructs a request exercising every region a
// translator reads, optionally marking one carrier per region. mark=false
// and mark=true build otherwise identical requests differing only in
// marker state — the pair AI-11.3's tests compare.
func buildTranslatorFixture(t *testing.T, mark bool) ai.Request {
	t.Helper()

	seg, err := ai.NewSegment("You are a travel planning assistant.")
	if err != nil {
		t.Fatalf("ai.NewSegment returned %v, want no failure", err)
	}
	if mark {
		seg = seg.MarkCacheBoundary()
	}
	system, err := ai.NewSystemInstruction(seg)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}

	tool, err := ai.NewTool("search_flights", "search for flights between two airports", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}
	if mark {
		tool = tool.MarkCacheBoundary()
	}
	toolSet, err := ai.NewToolSet(tool)
	if err != nil {
		t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
	}

	choice, err := ai.NewNamedToolChoice("search_flights")
	if err != nil {
		t.Fatalf("ai.NewNamedToolChoice returned %v, want no failure", err)
	}

	text, err := ai.NewText("plan a trip to Paris")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, text)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	if mark {
		msg = msg.MarkCacheBoundary()
	}

	request, err := ai.NewRequest(
		"cachicamas-neutral-model-1", []ai.Message{msg},
		ai.WithSystemInstruction(system),
		ai.WithTools(toolSet),
		ai.WithToolChoice(choice),
		ai.WithTemperature(0.4),
		ai.WithStopSequences("</plan>"),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest(mark=%t) returned %v, want no failure", mark, err)
	}
	return request
}

// translateBlind renders a request through the exported surface alone,
// touching model, every system segment's text, every tool declaration's
// name/description/schema, the tool choice, every message's role and
// content, and every generation option — and never calls IsCacheBoundary or
// CacheBoundaries anywhere (R-ACB-008). It is the illustrative translator
// for a provider that caches automatically: fully conformant while ignoring
// markers wholesale.
func translateBlind(t *testing.T, request ai.Request) string {
	t.Helper()

	var b strings.Builder
	b.WriteString("model=")
	b.WriteString(request.Model())

	if system, ok := request.SystemInstruction(); ok {
		b.WriteString("|system=")
		for _, seg := range system.Segments() {
			b.WriteString(seg.Text())
			b.WriteByte(';')
		}
	}

	if tools, ok := request.Tools(); ok {
		b.WriteString("|tools=")
		for _, tool := range tools.Tools() {
			b.WriteString(tool.Name())
			b.WriteByte(':')
			b.WriteString(tool.Description())
			b.WriteByte(':')
			b.Write(tool.Schema())
			b.WriteByte(';')
		}
	}

	if choice, ok := request.ToolChoice(); ok {
		b.WriteString("|toolChoice=")
		b.WriteString(choice.Mode().String())
		if name, namesOne := choice.Name(); namesOne {
			b.WriteByte(':')
			b.WriteString(name)
		}
	}

	b.WriteString("|messages=")
	for _, msg := range request.Messages() {
		b.WriteString(msg.Role().String())
		b.WriteByte(':')
		for _, part := range msg.Content() {
			translatePartBlind(&b, part)
		}
		b.WriteByte(';')
	}

	if tokens, ok := request.MaxOutputTokens(); ok {
		b.WriteString("|maxOutputTokens=")
		b.WriteString(strconv.Itoa(tokens))
	}
	if temperature, ok := request.Temperature(); ok {
		b.WriteString("|temperature=")
		b.WriteString(strconv.FormatFloat(temperature, 'f', -1, 64))
	}
	if topP, ok := request.TopP(); ok {
		b.WriteString("|topP=")
		b.WriteString(strconv.FormatFloat(topP, 'f', -1, 64))
	}
	if stops, ok := request.StopSequences(); ok {
		b.WriteString("|stopSequences=")
		b.WriteString(strings.Join(stops, ","))
	}

	return b.String()
}

// translatePartBlind appends one content part's payload to b, dispatching on
// its kind through the exported accessors alone.
func translatePartBlind(b *strings.Builder, part ai.Part) {
	switch part.Kind() {
	case ai.PartKindText:
		text, _ := part.Text()
		b.WriteString("text(")
		b.WriteString(text)
		b.WriteByte(')')
	case ai.PartKindReasoning:
		reasoning, _ := part.Reasoning()
		b.WriteString("reasoning(")
		b.WriteString(reasoning.Text())
		b.WriteByte(')')
	case ai.PartKindToolCall:
		call, _ := part.ToolCall()
		b.WriteString("toolcall(")
		b.WriteString(call.Name())
		b.WriteByte(':')
		b.Write(call.Arguments())
		b.WriteByte(')')
	case ai.PartKindToolResult:
		result, _ := part.ToolResult()
		b.WriteString("toolresult(")
		b.WriteString(result.Content())
		b.WriteByte(')')
	}
}

// AI-11.3 item 1, S-ACB-033 — a marker-blind translator renders a marked
// request and its unmarked twin identically: markers are advisory, and
// ignoring them wholesale is a conformant strategy (R-ACB-008).
func TestTranslateBlind_MarkedRequestAndUnmarkedTwin_RenderIdentically(t *testing.T) {
	t.Parallel()

	unmarked := buildTranslatorFixture(t, false)
	marked := buildTranslatorFixture(t, true)

	got, want := translateBlind(t, marked), translateBlind(t, unmarked)
	if got != want {
		t.Errorf("translateBlind(marked) = %q, want %q (translateBlind(unmarked)) — "+
			"a marker-blind translator must not observe a difference", got, want)
	}
}

// AI-11.3 item 1, S-ACB-035 — the blind rendering is complete relative to
// the unmarked twin: every region the fixture applies shows up in the
// rendering, so the identity above cannot be satisfied by two renderings
// that are merely both empty.
func TestTranslateBlind_Rendering_IsCompleteAcrossEveryAppliedRegion(t *testing.T) {
	t.Parallel()

	rendered := translateBlind(t, buildTranslatorFixture(t, true))

	for _, want := range []string{
		"model=cachicamas-neutral-model-1",
		"travel planning assistant",
		"search_flights",
		"toolChoice=specific:search_flights",
		"plan a trip to Paris",
		"temperature=0.4",
		"stopSequences=</plan>",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("translateBlind(request) = %q, want it to contain %q — no region may be silently dropped", rendered, want)
		}
	}
}

// translateAware is the same walk as translateBlind, plus one marker read.
// Its purpose is entirely to prove
// TestTranslateBlind_MarkedRequestAndUnmarkedTwin_RenderIdentically is not
// vacuous — without this control, a blind translator that rendered nothing
// at all would satisfy that test perfectly (design.md § 6, the AI-06
// testdata/handrolled + testdata/constructed lesson applied to a rendering
// rather than a build).
func translateAware(t *testing.T, request ai.Request) string {
	t.Helper()
	return translateBlind(t, request) + "|cacheBoundaries=" + strconv.Itoa(len(request.CacheBoundaries()))
}

// AI-11.3 item 3, S-ACB-034 — the control.
func TestTranslateAware_MarkedRequestAndUnmarkedTwin_RenderDifferently(t *testing.T) {
	t.Parallel()

	unmarked := buildTranslatorFixture(t, false)
	marked := buildTranslatorFixture(t, true)

	got, other := translateAware(t, marked), translateAware(t, unmarked)
	if got == other {
		t.Errorf("translateAware(marked) = translateAware(unmarked) = %q, want them to differ — "+
			"this control is what proves TestTranslateBlind_MarkedRequestAndUnmarkedTwin_RenderIdentically is not vacuous", got)
	}
}

// AI-11.3 item 2, S-ACB-036 *(pin)* — the usage record's cache-read and
// cache-write counts read and validate exactly as landed by AI-13.3, at the
// same position, and this milestone adds nothing there (R-ACB-009): AI-11
// adds request-side expression only.
func TestUsage_CacheReadAndCacheWrite_ReadAndValidateExactlyAsBeforeThisMilestone(t *testing.T) {
	t.Parallel()

	usage := ai.Usage{CacheRead: ai.Tokens(120), CacheWrite: ai.Tokens(40)}

	if got, ok := usage.CacheRead.Count(); !ok || got != 120 {
		t.Errorf("usage.CacheRead.Count() = (%d, %t), want (120, true)", got, ok)
	}
	if got, ok := usage.CacheWrite.Count(); !ok || got != 40 {
		t.Errorf("usage.CacheWrite.Count() = (%d, %t), want (40, true)", got, ok)
	}
	if err := usage.Validate(); err != nil {
		t.Errorf("usage.Validate() returned %v, want no failure", err)
	}

	negative := ai.Usage{CacheRead: ai.Tokens(-1)}
	err := negative.Validate()
	if err == nil {
		t.Fatalf("negative.Validate() returned no failure, want one — a negative cache-read count is out of range")
	}
	if !errors.Is(err, ai.ErrOutOfRange) {
		t.Errorf("errors.Is(err, ErrOutOfRange) = false, want true (err = %v)", err)
	}
	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
	}
	if got := violation.Path().String(); got != "cache_read" {
		t.Errorf("violation.Path() = %q, want %q", got, "cache_read")
	}
}

// AI-11.3 item 2, S-ACB-037 *(pin)* — the exported surface of this layer
// carries no hit-rate, cache-efficiency or cache-statistics accessor
// (V-REQ-25: "Layer 1 never measures a hit rate").
//
// Parses every non-test .go file's top-level declarations in src/ai with
// go/parser rather than checking a known file list, so a future accessor
// added anywhere in the package — not only in a file this milestone
// touches — still fails this pin.
func TestExportedSurface_CarriesNoHitRateOrCacheStatisticsAccessor(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../ai", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf(`parser.ParseDir("../ai") returned %v, want no failure`, err)
	}
	if len(pkgs) == 0 {
		t.Fatal(`parser.ParseDir("../ai") returned no packages; this pin would pass vacuously`)
	}

	forbidden := []string{"hitrate", "hitratio", "cacheefficiency", "cachestatistics"}
	checked := 0
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || !fn.Name.IsExported() {
					continue
				}
				checked++
				lower := strings.ToLower(fn.Name.Name)
				for _, name := range forbidden {
					if strings.Contains(lower, name) {
						t.Errorf("found exported declaration %q at %s, want no hit-rate/cache-efficiency/cache-statistics accessor",
							fn.Name.Name, fset.Position(fn.Pos()))
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no exported top-level declaration was inspected; this pin would pass vacuously")
	}
}
