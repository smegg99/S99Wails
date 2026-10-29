// internal/cli/root.go

package cli

import (
	"errors"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/spf13/cobra"

	"github.com/smegg99/s99wails/v2/internal/cli/catalog"
)

// Root makes cmd a localized root command: --lang, translated help and usage errors that exit 2.
func (a *App) Root(cmd *cobra.Command) *cobra.Command {
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	cmd.SetOut(a.data)
	cmd.SetErr(a.out)
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return Usage(err) })
	// --lang fires during flag parsing, before cobra answers --help.
	cmd.PersistentFlags().Func("lang", "", func(value string) error {
		match, ok := catalog.Supported(value)
		if !ok {
			return errors.New(catalog.CliErrUnsupportedLanguage(a.Say(), catalog.CliErrUnsupportedLanguageParams{
				Lang: value, Supported: strings.Join(catalog.Locales, ", "),
			}))
		}
		a.localize(match)
		return nil
	})
	// Registered now, or Find takes --help for a flag with a value and swallows --lang.
	cmd.InitDefaultHelpFlag()
	a.Describe(func(l *i18n.Localizer) {
		cmd.Flags().Lookup("help").Usage = catalog.CliFlagHelp(l, catalog.CliFlagHelpParams{Command: cmd.Name()})
		cmd.PersistentFlags().Lookup("lang").Usage = catalog.CliFlagLang(l)
		cmd.SetUsageTemplate(usageTemplate(l))
	})
	a.localize(catalog.FromEnv(a.env))
	return cmd
}

// NoArgs answers a stray positional in the reader's language, as a usage error.
func (a *App) NoArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return Usage(errors.New(catalog.CliErrNoArgs(a.Say(), catalog.CliErrNoArgsParams{Command: cmd.CommandPath()})))
	}
	return nil
}

// cobraUsageTemplate is cobra v1.10.2's defaultUsageTemplate verbatim, so upgrades show a diff.
const cobraUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// usageTemplate replaces cobra's headings, which live in the template rather than in any Short field.
func usageTemplate(l *i18n.Localizer) string {
	// Ordered: "Flags:" is a substring of "Global Flags:".
	replacements := []struct{ from, to string }{
		{"Usage:", catalog.CliHeadingUsage(l)},
		{"Aliases:", catalog.CliHeadingAliases(l)},
		{"Examples:", catalog.CliHeadingExamples(l)},
		{"Available Commands:", catalog.CliHeadingCommands(l)},
		{"Additional Commands:", catalog.CliHeadingAdditionalCommands(l)},
		{"Global Flags:", catalog.CliHeadingGlobalFlags(l)},
		{"Flags:", catalog.CliHeadingFlags(l)},
		{"Additional help topics:", catalog.CliHeadingHelpTopics(l)},
		{
			`Use "{{.CommandPath}} [command] --help" for more information about a command.`,
			catalog.CliHeadingMoreInfo(l, catalog.CliHeadingMoreInfoParams{Command: "{{.CommandPath}}"}),
		},
	}
	out := cobraUsageTemplate
	for _, r := range replacements {
		out = strings.Replace(out, r.from, r.to, 1)
	}
	return out
}
