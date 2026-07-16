package logger

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRegisteredCategoriesAreStable(t *testing.T) {
	want := []LogCategory{
		CategoryApplication,
		CategoryComposeRuntime,
		CategoryDatabaseEnt,
		CategoryDockerEvents,
		CategoryDockerStats,
		CategoryRuntimeCache,
		CategoryRuntimeMetrics,
		CategoryRuntimeStats,
		CategorySchedulerPoll,
	}
	if got := RegisteredCategories(); !reflect.DeepEqual(got, want) {
		t.Fatalf("registered categories = %v, want %v", got, want)
	}
}

func TestParseCategoryRulesRejectsInvalidInput(t *testing.T) {
	for _, raw := range []string{
		"docker.stats",
		"docker.stats=not-bool",
		"unknown.category=true",
		"docker.stats=true,docker.stats=false",
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseCategoryRules(raw); err == nil {
				t.Fatalf("ParseCategoryRules(%q) unexpectedly succeeded", raw)
			}
		})
	}
}

func TestCategoryRulesUseLongestPrefixAndTraceDefaultsOff(t *testing.T) {
	rules, err := ParseCategoryRules("docker=true,docker.stats=false,runtime=true")
	if err != nil {
		t.Fatalf("parse rules: %v", err)
	}
	if rules.allowed(CategoryDockerStats, zap.InfoLevel) {
		t.Fatal("specific false rule must suppress every level")
	}
	if !rules.allowed(CategoryDockerEvents, TraceLevel) {
		t.Fatal("parent true rule must enable descendant TRACE")
	}
	if (CategoryRules{}).allowed(CategoryRuntimeMetrics, TraceLevel) {
		t.Fatal("unconfigured category TRACE must remain disabled")
	}
	if !(CategoryRules{}).allowed(CategoryRuntimeMetrics, zap.DebugLevel) {
		t.Fatal("unconfigured category debug must retain normal log-level behavior")
	}
}

func TestDisabledCategoryDoesNotReachSink(t *testing.T) {
	core, observed := observer.New(TraceLevel)
	logger := zap.New(wrapCategoryCore(CategoryRules{CategoryDockerStats: false})(core))
	category := Category(logger, CategoryDockerStats)

	category.Trace("poll", zap.String("expensive", "value"))
	category.Info("still suppressed")
	if got := observed.All(); len(got) != 0 {
		t.Fatalf("disabled category reached sink: %#v", got)
	}
}

func TestTraceLazySkipsFieldBuilderWhenCategoryDisabled(t *testing.T) {
	core, observed := observer.New(TraceLevel)
	category := Category(zap.New(wrapCategoryCore(CategoryRules{})(core)), CategoryDockerStats)
	built := false

	category.TraceLazy("poll", func() []zap.Field {
		built = true
		return []zap.Field{zap.String("unexpected", "field")}
	})
	if built {
		t.Fatal("disabled TRACE invoked lazy builder")
	}
	if got := observed.All(); len(got) != 0 {
		t.Fatalf("disabled TRACE reached sink: %#v", got)
	}
}

func TestTraceWritesWhenLevelAndCategoryAreEnabled(t *testing.T) {
	core, observed := observer.New(TraceLevel)
	category := Category(zap.New(wrapCategoryCore(CategoryRules{CategoryDockerStats: true})(core)), CategoryDockerStats)
	category.Trace("poll")

	entries := observed.All()
	if len(entries) != 1 || entries[0].Level != TraceLevel {
		t.Fatalf("trace entries = %#v", entries)
	}
}
