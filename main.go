package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const (
	// TemplateFile is the location of the template file to be generated.
	TemplateFile = "./template/conversions.tmpl"
	// OutputFile is the location to put the generated go code.
	OutputFile = "./output/conversions.go"
)

type (
	// ConversionFailure is a type for marrying the two types in a conversion failure as reported
	// by the go compiler.
	ConversionFailure struct {
		From string
		To   string
	}

	// ConversionFailures is a helper type around a []ConversionFailure to allow easier searching
	// through a []ConversionFailure.
	ConversionFailures []ConversionFailure
)

var (
	// Primitives contains the list of all primitives in golang, as reported by the builtin package.
	// I suppose even this could be extracted from the builtin package itself via some code introspection,
	// but for now I hardcoded the list since the list of built-in primitives is unlikely to change
	// frequently, if at all.
	Primitives = []string{
		"bool",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"int8",
		"int16",
		"int32",
		"int64",
		"float32",
		"float64",
		"complex64",
		"complex128",
		"string",
		"int",
		"uint",
		"uintptr",
		"byte", // NOTE(justin): is also a type alias for uint8
		"rune", // NOTE(justin): is also a type alias for int32
	}
)

// main is the main function for this program, but it is only responsible
// for calling Main and some other boilerplate code setup.
func main() {
	defer func(start time.Time) {
		duration := time.Since(start)
		logrus.Infof("execution took %v", duration)
	}(time.Now())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logrus.Info("starting")

	err := Main(ctx)
	if err != nil {
		logrus.Fatal(errors.Wrap(err, filepath.Base(os.Args[0])))
		return
	}

	logrus.Info("done")
}

// Main is the main driver function for this application.
func Main(ctx context.Context) error {
	err := Generate(ctx)
	if err != nil {
		return errors.Wrap(err, "generating")
	}

	cfs, err := Compile(ctx)
	if err != nil {
		return errors.Wrap(err, "compiling")
	}

	err = Report(ctx, cfs)
	if err != nil {
		return errors.Wrap(err, "reporting results")
	}

	return nil
}

// Generate executes the template and generates the go code for it.
// The templated file is assumed to be at TemplateFile, and the generated
// go code will be sent to OutputFile.
func Generate(_ context.Context) error {
	t, err := template.ParseFiles(TemplateFile)
	if err != nil {
		return errors.Wrapf(err, "parsing template file %q", TemplateFile)
	}

	f, err := os.Create(OutputFile)
	if err != nil {
		return errors.Wrapf(err, "creating output file %q", OutputFile)
	}
	defer func() { _ = f.Close() }()

	type Data struct {
		Now        string
		App        string
		Primitives []string
	}
	var data Data
	data.Now = time.Now().Format(time.RFC3339)
	data.App = os.Args[0]
	data.Primitives = Primitives

	err = t.Execute(f, data)
	if err != nil {
		return errors.Wrap(err, "executing template")
	}

	return nil
}

// Compile compiles the generated go code located at OutputFile, expecting it
// to fail compilation and throw errors. It records these compilation errors
// into a ConversionFailures and returns them.
func Compile(_ context.Context) (ConversionFailures, error) {
	command := fmt.Sprintf("go build -gcflags=-e -o /dev/null %s", OutputFile)
	pieces := strings.Split(command, " ")
	program := pieces[0]
	args := pieces[1:]

	cmd := exec.Command(program, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && err.Error() != "exit status 2" {
		// NOTE(justin): We expect to get an exit status 2 since we expect the compiler to complain.
		// If we got some other error, this will be triggered.
		return nil, errors.Wrap(err, "unexpected error while running compilation command")
	}

	stderrContent := stderr.String()
	stderrLines := strings.Split(stderrContent, "\n")

	var conversionErrs []string
	for _, stderrLine := range stderrLines {
		if strings.Contains(stderrLine, "cannot convert") {
			conversionErrs = append(conversionErrs, stderrLine)
		}
	}

	var cfs ConversionFailures
	re := regexp.MustCompile(".+ p.(.+) \\(.+\\) to type (.+)")
	for _, conversionErr := range conversionErrs {
		matches := re.FindStringSubmatch(conversionErr)
		from := matches[1]
		to := matches[2]
		var conversionFailure ConversionFailure
		conversionFailure.From = from
		conversionFailure.To = to
		cfs = append(cfs, conversionFailure)
	}

	return cfs, nil
}

// Report iterates over every primitive type against every primitive type and
// reports if a ConversionFailure exists for that conversion or not.
func Report(_ context.Context, cfs ConversionFailures) error {
	for _, outerPrimitive := range Primitives {
		logrus.Infof("---------- converting %s values ----------\n", outerPrimitive)
		for _, innerPrimitive := range Primitives {
			var compatible string
			if !cfs.Contains(outerPrimitive, innerPrimitive) {
				compatible = "✅"
			} else {
				compatible = "❌"
			}
			logrus.Infof("%10s -> %-10s %s ", outerPrimitive, innerPrimitive, compatible)
		}
	}

	return nil
}

// Contains is a helper function for determining if cfs contains a ConversionFailure
// that has it's From set to from and To set to to.
func (cfs ConversionFailures) Contains(from, to string) bool {
	for _, conversionFailure := range cfs {
		if conversionFailure.From == from && conversionFailure.To == to {
			return true
		}
	}
	return false
}
