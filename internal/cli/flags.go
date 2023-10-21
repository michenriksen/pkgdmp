package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
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

var symbolTypeMap = map[string]pkgdmp.SymbolType{
	"arrayType": pkgdmp.SymbolArrayType,
	"chanType":  pkgdmp.SymbolChanType,
	"const":     pkgdmp.SymbolConst,
	"func":      pkgdmp.SymbolFunc,
	"funcType":  pkgdmp.SymbolFuncType,
	"identType": pkgdmp.SymbolIdentType,
	"interface": pkgdmp.SymbolInterfaceType,
	"mapType":   pkgdmp.SymbolMapType,
	"method":    pkgdmp.SymbolMethod,
	"struct":    pkgdmp.SymbolStructType,
}

var (
	// ErrNoDirs is returned by [ParseFlags] if args contain no directories.
	ErrNoDirs = errors.New("no directories in command line arguments")

	// ErrVersion is returned by [ParseFlags] if the -version flag is specified.
	ErrVersion = errors.New("version")
)

var flagSet *flag.FlagSet

// Config represents CLI configuration from flags.
type Config struct {
	onlyPackages    map[string]struct{}
	excludePackages map[string]struct{}
	ExcludePackages string
	Only            string
	ExcludeMatching string
	Theme           string
	Matching        string
	OnlyPackages    string
	Exclude         string
	Dirs            []string `env:"skip"`
	NoDocs          bool
	NoTags          bool
	NoHighlight     bool
	FullDocs        bool
	Unexported      bool
	Version         bool `env:"skip"`
	NoEnv           bool `env:"skip"`
	JSON            bool
}

// IncludePackage returns true if package with provided name should be included
// in the report according to configuration, or false otherwise.
func (c *Config) IncludePackage(name string) bool {
	if len(c.onlyPackages) != 0 {
		_, ok := c.onlyPackages[name]
		return ok
	}

	if len(c.excludePackages) != 0 {
		if _, ok := c.excludePackages[name]; ok {
			return false
		}
	}

	return true
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

	if cfg.OnlyPackages != "" {
		names := strings.Split(cfg.OnlyPackages, ",")
		cfg.onlyPackages = make(map[string]struct{}, len(names))

		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}

			cfg.onlyPackages[name] = struct{}{}
		}
	}

	if cfg.ExcludePackages != "" {
		names := strings.Split(cfg.ExcludePackages, ",")
		cfg.excludePackages = make(map[string]struct{}, len(names))

		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}

			cfg.excludePackages[name] = struct{}{}
		}
	}

	return cfg, 0, nil
}

// ParserOptsFromCfg constructs parser options from CLI configuration.
func ParserOptsFromCfg(cfg *Config) ([]pkgdmp.ParserOption, error) {
	var opts []pkgdmp.ParserOption

	if cfg.FullDocs {
		opts = append(opts, pkgdmp.WithFullDocs())
	}

	if cfg.NoDocs {
		opts = append(opts, pkgdmp.WithNoDocs())
	}

	if cfg.NoTags {
		opts = append(opts, pkgdmp.WithNoTags())
	}

	filters, err := filtersFromCfg(cfg)
	if err != nil {
		return nil, err
	}

	if len(filters) != 0 {
		opts = append(opts, pkgdmp.WithSymbolFilters(filters...))
	}

	return opts, nil
}

func filtersFromCfg(cfg *Config) ([]pkgdmp.SymbolFilter, error) {
	var filters []pkgdmp.SymbolFilter

	if !cfg.Unexported {
		filters = append(filters, pkgdmp.FilterUnexported(pkgdmp.Exclude))
	}

	if cfg.Exclude != "" {
		st, err := strToSymbolTypes(cfg.Exclude)
		if err != nil {
			return nil, fmt.Errorf("parsing symbol types: %w", err)
		}

		filters = append(filters, pkgdmp.FilterSymbolTypes(pkgdmp.Exclude, st...))
	}

	if cfg.Only != "" {
		st, err := strToSymbolTypes(cfg.Only)
		if err != nil {
			return nil, fmt.Errorf("parsing symbol types: %w", err)
		}

		filters = append(filters, pkgdmp.FilterSymbolTypes(pkgdmp.Include, st...))
	}

	if cfg.Matching != "" {
		p, err := regexp.Compile(cfg.Matching)
		if err != nil {
			return nil, fmt.Errorf("parsing matching regular expression: %w", err)
		}

		filters = append(filters, pkgdmp.FilterMatchingIdents(pkgdmp.Include, p))
	}

	if cfg.ExcludeMatching != "" {
		p, err := regexp.Compile(cfg.ExcludeMatching)
		if err != nil {
			return nil, fmt.Errorf("parsing exclude matching regular expression: %w", err)
		}

		filters = append(filters, pkgdmp.FilterMatchingIdents(pkgdmp.Exclude, p))
	}

	return filters, nil
}

func initFlagSet(cfg *Config, output io.Writer) {
	flagSet = nil // Avoid flag redefinition error.
	flagSet = flag.NewFlagSet("pkgdmp", flag.ContinueOnError)

	flagSet.SetOutput(output)
	flagSet.Usage = usage

	flagSet.StringVar(&cfg.Matching, "matching", "",
		flagDescf("Matching", "only include symbol with names matching regular expression"),
	)
	flagSet.StringVar(&cfg.ExcludeMatching, "exclude-matching", "",
		flagDescf("ExcludeMatching", "exclude symbols with names matching regular expression"),
	)
	flagSet.BoolVar(&cfg.Unexported, "unexported", false,
		flagDescf("Unexported", "include unexported entities"),
	)
	flagSet.StringVar(&cfg.Only, "only", "",
		flagDescf("Only", "comma-separated list of symbol types to include"),
	)
	flagSet.StringVar(&cfg.Exclude, "exclude", "",
		flagDescf("Exclude", "comma-separated list of symbol types to exclude"),
	)
	flagSet.StringVar(&cfg.ExcludePackages, "exclude-packages", "",
		flagDescf("ExcludePackages", "comma-separated list of package names to exclude"),
	)
	flagSet.StringVar(&cfg.OnlyPackages, "only-packages", "",
		flagDescf("OnlyPackages", "comma-separated list of package names to include"),
	)
	flagSet.BoolVar(&cfg.NoDocs, "no-docs", false,
		flagDescf("NoDocs", "exclude doc comments"),
	)
	flagSet.BoolVar(&cfg.NoTags, "no-tags", false,
		flagDescf("NoTags", "exclude struct field tags"),
	)
	flagSet.BoolVar(&cfg.FullDocs, "full-docs", false,
		flagDescf("FullDocs", "include full doc comments instead of synopsis"),
	)
	flagSet.StringVar(&cfg.Theme, "theme", defaultTheme,
		flagDescf("Theme", "syntax highlighting theme to use - see %s", themesURL),
	)
	flagSet.BoolVar(&cfg.JSON, "json", false,
		flagDescf("JSON", "output as JSON"),
	)
	flagSet.BoolVar(&cfg.NoEnv, "no-env", false,
		fmt.Sprintf("skip loading of configuration from '%s_*' environment variables", flagEnvPrfx),
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
	fmt.Fprintf(flagSet.Output(), "\nSYMBOL TYPES:\n\n  %s\n\n", strings.Join(supportedSymbolTypes(), ", "))
}

func strToSymbolTypes(list string) ([]pkgdmp.SymbolType, error) {
	ss := strings.Split(list, ",")
	res := make([]pkgdmp.SymbolType, 0, len(ss))

	for _, s := range ss {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}

		st, ok := symbolTypeMap[s]
		if !ok {
			return nil, fmt.Errorf("unsupported symbol type string: %q", s)
		}

		res = append(res, st)
	}

	return res, nil
}

func supportedSymbolTypes() []string {
	res := make([]string, 0, len(symbolTypeMap))

	for stStr := range symbolTypeMap {
		res = append(res, stStr)
	}

	sort.Strings(res)

	return res
}
