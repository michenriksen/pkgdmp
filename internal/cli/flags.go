package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/michenriksen/pkgdmp"
)

const flagEnvPrfx = "PKGDMP"

const (
	themesURL    = "https://xyproto.github.io/splash/docs/"
	defaultTheme = "swapoff"
)

const versionTmpl = `%s:
  Version:    %s
  Go version: %s
  Git commit: %s
  Released:   %s
`

var (
	// ErrNoDirs is returned by [ParseFlags] if args contain no directories.
	ErrNoDirs = errors.New("no directories in command line arguments")

	// ErrVersion is returned by [ParseFlags] if the -version flag is specified.
	ErrVersion = errors.New("version")
)

var flagSet *flag.FlagSet

// Config represents CLI configuration from flags.
type Config struct {
	Match        string
	Exclude      string
	Theme        string
	Dirs         []string `env:"skip"`
	NoFuncTypes  bool
	NoDoc        bool
	JSON         bool
	NoEnv        bool `env:"skip"`
	NoFuncs      bool
	NoHighlight  bool
	NoInterfaces bool
	NoStructs    bool
	FullDoc      bool
	Unexported   bool
	Version      bool `env:"skip"`
}

// ParseFlags parses command line arguments as flags and returns a CLI
// configuration together with exit code to use if error is also returned.
func ParseFlags(args []string, output io.Writer) (*Config, int, error) {
	cfg := &Config{}

	initFlagSet(cfg, output)

	if err := flagSet.Parse(args); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(output, "%v\n\n", err)
			flagSet.Usage()

			return nil, 1, err //nolint:wrapcheck // no need to wrap error.
		}

		return nil, 0, err //nolint:wrapcheck // no need to wrap error.
	}

	if cfg.Version {
		fmt.Fprintf(output, versionTmpl, AppName, Version(), BuildGoVersion(), BuildCommit(), BuildTime())
		return nil, 0, ErrVersion
	}

	if len(flagSet.Args()) == 0 {
		fmt.Fprintf(output, "no directories specified\n\n")
		flagSet.Usage()

		return nil, 1, ErrNoDirs
	}

	cfg.Dirs = flagSet.Args()

	envConfig(cfg)

	return cfg, 0, nil
}

// ParserOptsFromCfg constructs parser options from CLI configuration.
func ParserOptsFromCfg(cfg *Config) (pkgdmp.ParserOptions, error) {
	opts := pkgdmp.ParserOptions{
		ExcludeDocs:       cfg.NoDoc,
		ExcludeFuncTypes:  cfg.NoFuncTypes,
		ExcludeFuncs:      cfg.NoFuncs,
		ExcludeInterfaces: cfg.NoInterfaces,
		ExcludeStructs:    cfg.NoStructs,
		FullDocs:          cfg.FullDoc,
		Unexported:        cfg.Unexported,
	}

	if cfg.Match != "" {
		p, err := regexp.Compile(cfg.Match)
		if err != nil {
			return pkgdmp.ParserOptions{}, fmt.Errorf("parsing match regular expression: %w", err)
		}

		opts.OnlyRegexp = p
	}

	if cfg.Exclude != "" {
		p, err := regexp.Compile(cfg.Exclude)
		if err != nil {
			return pkgdmp.ParserOptions{}, fmt.Errorf("parsing exclude regular expression: %w", err)
		}

		opts.ExcludeRegexp = p
	}

	return opts, nil
}

func initFlagSet(cfg *Config, output io.Writer) {
	flagSet = nil // Avoid flag redefinition error.
	flagSet = flag.NewFlagSet("pkgdmp", flag.ContinueOnError)

	flagSet.SetOutput(output)
	flagSet.Usage = usage

	flagSet.StringVar(&cfg.Match, "match", "",
		flagDescf("Match", "only include entities with names matching regular expression"),
	)
	flagSet.StringVar(&cfg.Exclude, "exclude", "",
		flagDescf("Exclude", "exclude entities with names matching regular expression"),
	)
	flagSet.StringVar(&cfg.Theme, "theme", defaultTheme,
		flagDescf("Theme", "syntax highlighting theme to use - see %s", themesURL),
	)
	flagSet.BoolVar(&cfg.NoFuncTypes, "no-func-types", false,
		flagDescf("NoFuncTypes", "exclude function types"),
	)
	flagSet.BoolVar(&cfg.NoDoc, "no-doc", false,
		flagDescf("NoDoc", "exclude doc comments"),
	)
	flagSet.BoolVar(&cfg.JSON, "json", false,
		flagDescf("JSON", "output as JSON"),
	)
	flagSet.BoolVar(&cfg.NoEnv, "no-env", false,
		fmt.Sprintf("skip loading of configuration from '%s_*' environment variables", flagEnvPrfx),
	)
	flagSet.BoolVar(&cfg.NoFuncs, "no-funcs", false,
		flagDescf("NoFuncs", "exclude functions"),
	)
	flagSet.BoolVar(&cfg.NoHighlight, "no-highlight", false,
		flagDescf("NoHighlight", "skip source code highlighting"),
	)
	flagSet.BoolVar(&cfg.NoInterfaces, "no-interfaces", false,
		flagDescf("NoInterfaces", "exclude interfaces"),
	)
	flagSet.BoolVar(&cfg.NoStructs, "no-structs", false,
		flagDescf("NoStructs", "exclude structs"),
	)
	flagSet.BoolVar(&cfg.FullDoc, "full-doc", false,
		flagDescf("FullDoc", "include full doc comments instead of synopsis"),
	)
	flagSet.BoolVar(&cfg.Unexported, "unexported", false,
		flagDescf("Unexported", "include unexported entities"),
	)
	flagSet.BoolVar(&cfg.Version, "version", false, "print version information and exit")
}

func envConfig(cfg *Config) {
	if cfg.NoEnv {
		return
	}

	cfgVal := reflect.ValueOf(cfg).Elem()
	cfgTyp := reflect.TypeOf(*cfg)

	for i := 0; i < cfgVal.NumField(); i++ {
		field := cfgVal.Field(i)
		fieldTyp := cfgTyp.Field(i)
		fieldName := fieldTyp.Name

		if !fieldTyp.IsExported() {
			continue
		}

		if fieldTyp.Tag.Get("env") == "skip" {
			continue
		}

		val, ok := os.LookupEnv(cfgEnvKey(fieldName))
		if !ok {
			continue
		}

		switch field.Kind() {
		case reflect.Bool:
			field.SetBool(isTruthy(val))
		case reflect.String:
			field.SetString(val)
		}
	}

	if envNoColor() {
		cfg.NoHighlight = true
	}
}

func envNoColor() bool {
	// See https://no-color.org/
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return true
	}

	// Check PKGDMP_NO_COLOR.
	if _, ok := os.LookupEnv(cfgEnvKey("NO_COLOR")); ok {
		return true
	}

	// See https://bixense.com/clicolors/
	if _, ok := os.LookupEnv("CLICOLOR_FORCE"); ok {
		return false
	}

	// $TERM is often set to `dumb` to indicate that the terminal is very basic
	// and sometimes if the current command output is redirected to a file or
	// piped to another command.
	if os.Getenv("TERM") == "dumb" {
		return true
	}

	return false
}

func isTruthy(val string) bool {
	val = strings.ToLower(val)
	truthies := []string{"1", "true", "t", "yes"}

	for _, t := range truthies {
		if val == t {
			return true
		}
	}

	return false
}

func flagDescf(field, format string, args ...any) string {
	desc := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s [$%s]", desc, cfgEnvKey(field))
}

func cfgEnvKey(field string) string {
	field = strings.ToUpper(strings.Join(splitCamelCase(field), "_"))

	return fmt.Sprintf("%s_%s", flagEnvPrfx, field)
}

func splitCamelCase(s string) []string {
	if strings.ToUpper(s) == s {
		return []string{s}
	}

	var words []string

	wordStart := 0

	for i, char := range s {
		if unicode.IsUpper(char) {
			if i > wordStart {
				words = append(words, s[wordStart:i])
			}

			wordStart = i
		}
	}

	if len(s) > wordStart {
		words = append(words, s[wordStart:])
	}

	return words
}

func usage() {
	fmt.Fprintf(flagSet.Output(), "%s v%s\n\nUSAGE:\n\n  %s [FLAGS] DIRECTORY [DIRECTORY2] ...\n\nFLAGS:\n\n",
		AppName, Version(), AppName,
	)
	flagSet.PrintDefaults()
}
