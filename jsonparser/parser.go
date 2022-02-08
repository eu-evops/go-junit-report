package jsonparser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Result represents a test result.
type Result int

// Test result constants
const (
	PASS Result = iota
	FAIL
	SKIP
)

// Report is a collection of package tests.
type Report struct {
	Packages []*Package
}

// Package contains the test results of a single package.
type Package struct {
	Name        string
	Duration    time.Duration
	Tests       []*Test
	Benchmarks  []*Benchmark
	CoveragePct string

	// Time is deprecated, use Duration instead.
	Time int // in milliseconds
}

// Test contains the results of a single test.
type Test struct {
	Name     string
	Package  string
	Duration time.Duration
	Result   Result
	Output   []string

	SubtestIndent string

	// Time is deprecated, use Duration instead.
	Time int // in milliseconds
}

// Benchmark contains the results of a single benchmark.
type Benchmark struct {
	Name     string
	Duration time.Duration
	// number of B/op
	Bytes int
	// number of allocs/op
	Allocs int
}

type LineOutput struct {
	Time    time.Time
	Action  string
	Package string
	Test    string
	Output  string
	Elapsed float32
}

// Parse parses go test output from reader r and returns a report with the
// results. An optional pkgName can be given, which is used in case a package
// result line is missing.
func Parse(r io.Reader, pkgName string) (*Report, error) {
	reader := bufio.NewReader(r)

	report := &Report{make([]*Package, 0)}

	// keep track of tests we find
	var tests []*Test

	// keep track of benchmarks we find
	var benchmarks []*Benchmark

	// coverage percentage report for current package
	var coveragePct string

	// parse lines
	for {
		l, _, err := reader.ReadLine()

		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		var lineoutput LineOutput
		json.Unmarshal(l, &lineoutput)

		fmt.Fprintf(os.Stderr, "%s", lineoutput.Output)

		if lineoutput.Test == "" {
			var p *Package
			if p = findPackage(report.Packages, lineoutput.Package); p == nil {
				p = &Package{
					Name:        lineoutput.Package,
					Duration:    0,
					Tests:       make([]*Test, 0),
					Benchmarks:  benchmarks,
					CoveragePct: coveragePct,
				}
				report.Packages = append(report.Packages, p)
			}

			if lineoutput.Action == "pass" {
				p.Duration = time.Duration(lineoutput.Elapsed * float32(time.Second))
			}
		} else {
			var t *Test
			if t = findTest(tests, lineoutput.Test); t == nil {
				t = &Test{
					Name:    lineoutput.Test,
					Package: lineoutput.Package,
					Result:  FAIL,
					Output:  make([]string, 0),
				}
				tests = append(tests, t)
			}
			if lineoutput.Action == "output" {
				t.Output = append(t.Output, lineoutput.Output)
			}

			if lineoutput.Action == "pass" || lineoutput.Action == "fail" {
				if lineoutput.Action == "pass" {
					t.Result = PASS
				}
				t.Duration = time.Duration(lineoutput.Elapsed * float32(time.Second))
			}
		}
	}

	for _, t := range tests {
		var p *Package
		if p = findPackage(report.Packages, t.Package); p == nil {
			p = &Package{
				Name:        t.Package,
				Duration:    0,
				Tests:       make([]*Test, 0),
				Benchmarks:  benchmarks,
				CoveragePct: coveragePct,
			}
			report.Packages = append(report.Packages, p)
		}

		p.Tests = append(p.Tests, t)
	}

	return report, nil
}

func findTest(tests []*Test, name string) *Test {
	for i := len(tests) - 1; i >= 0; i-- {
		if tests[i].Name == name {
			return tests[i]
		}
	}
	return nil
}

func findPackage(packages []*Package, name string) *Package {
	for i := len(packages) - 1; i >= 0; i-- {
		if packages[i].Name == name {
			return packages[i]
		}
	}
	return nil
}

// Failures counts the number of failed tests in this report
func (r *Report) Failures() int {
	count := 0

	for _, p := range r.Packages {
		for _, t := range p.Tests {
			if t.Result == FAIL {
				count++
			}
		}
	}

	return count
}
