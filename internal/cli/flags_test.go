package cli_test

import (
	"errors"
	"flag"
	"io"
	"reflect"
	"regexp"
	"testing"

	"github.com/michenriksen/pkgdmp/internal/cli"
)

func TestParseFlags(t *testing.T) {
	tt := []struct {
		name         string
		args         []string
		wantCfg      *cli.Config
		wantExitCode int
		wantErr      error
	}{
		{
			name:         "no args",
			wantExitCode: 1,
			wantErr:      cli.ErrNoDirs,
		},
		{
			name:         "help flag",
			args:         []string{"-help"},
			wantExitCode: 0,
			wantErr:      flag.ErrHelp,
		},
		{
			name:         "version flag",
			args:         []string{"-version"},
			wantExitCode: 0,
			wantErr:      cli.ErrVersion,
		},
		{
			name:         "flags but no directories",
			args:         []string{"-unexported", "-full-docs"},
			wantExitCode: 1,
			wantErr:      cli.ErrNoDirs,
		},
		{
			name: "flags and directories",
			args: []string{"-unexported", "-no-docs", "-exclude=interfaces", "directory1", "directory2"},
			wantCfg: &cli.Config{
				Unexported: true,
				NoDocs:     true,
				Exclude:    "interfaces",
				Dirs:       []string{"directory1", "directory2"},
				Theme:      "swapoff",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cfg, exitCode, err := cli.ParseFlags(tc.args, io.Discard)

			if !reflect.DeepEqual(cfg, tc.wantCfg) {
				t.Errorf("expected config:\n\n%#v\n\nbut got:\n\n%#v", tc.wantCfg, cfg)
			}

			if exitCode != tc.wantExitCode {
				t.Errorf("expected exit code %d, but got %d", tc.wantExitCode, exitCode)
			}

			if !errors.Is(err, tc.wantErr) {
				if tc.wantErr == nil {
					t.Errorf("expected no error, but got: %v", err)
				}

				t.Errorf("expected error %v, but got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestParserOptsFromCfg(t *testing.T) {
	tt := []struct {
		name          string
		cfg           *cli.Config
		wantOpts      []string
		wantErrRegexp *regexp.Regexp
	}{
		{
			name: "default config",
			cfg:  &cli.Config{},
			wantOpts: []string{
				"symbolFilters(filters=filterUnexported(action=Exclude))",
			},
		},
		{
			name: "full docs and exclude interfaces",
			cfg:  &cli.Config{FullDocs: true, Exclude: "interface"},
			wantOpts: []string{
				"fullDocs",
				"symbolFilters(filters=filterUnexported(action=Exclude),filterSymbolTypes(action=Exclude,symbolTypes=SymbolInterfaceType))",
			},
		},
		{
			name: "match and exclude patterns",
			cfg:  &cli.Config{Matching: `^FooBa(r|z)`, ExcludeMatching: `(Hello|Hi)World`},
			wantOpts: []string{
				"symbolFilters(filters=" +
					"filterUnexported(action=Exclude)," +
					"filterMatchingIdents(action=Include,pattern=^FooBa(r|z))," +
					"filterMatchingIdents(action=Exclude,pattern=(Hello|Hi)World))",
			},
		},
		{
			name:          "invalid match regexp",
			cfg:           &cli.Config{Matching: `a\x{2`},
			wantErrRegexp: regexp.MustCompile(`parsing matching regular expression:.*invalid escape sequence`),
		},
		{
			name:          "invalid exclude regexp",
			cfg:           &cli.Config{ExcludeMatching: `a\x{2`},
			wantErrRegexp: regexp.MustCompile(`parsing exclude matching regular expression:.*invalid escape sequence`),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := cli.ParserOptsFromCfg(tc.cfg)

			if wOptsLen := len(tc.wantOpts); wOptsLen != 0 {
				optsLen := len(opts)

				if optsLen != wOptsLen {
					t.Fatalf("expected option length to be %d, but got %d", wOptsLen, optsLen)
				}

				for i, opt := range opts {
					wantOpt := tc.wantOpts[i]
					actualOpt := opt.String()

					if actualOpt != wantOpt {
						t.Fatalf("expected option at index %d to be:\n\n%s\n\nbut is:\n\n%s\n\n",
							i, wantOpt, actualOpt,
						)
					}
				}
			}

			if tc.wantErrRegexp != nil {
				if err == nil {
					t.Fatalf("expected error matching regular expression `%s`, but got no error", tc.wantErrRegexp)
				}

				if !tc.wantErrRegexp.MatchString(err.Error()) {
					t.Errorf("expected error %q to match regular expression `%s`", err, tc.wantErrRegexp)
				}
			}
		})
	}
}
