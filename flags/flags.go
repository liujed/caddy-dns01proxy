package flags

import (
	"fmt"
	"reflect"

	"github.com/liujed/goutil/optionals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Defines a command-line Flag.
type Flag[T any] struct {
	// Whether this flag is inherited by subcommands.
	Persistent bool

	Name         string
	ShortName    optionals.Optional[rune]
	DefaultValue T
	UsageMsg     string

	Required       bool
	Hidden         bool
	DeprecationMsg optionals.Optional[string]

	// If given, then command-line completions will be restricted to filenames
	// having any of the given extensions.
	FilenameExts optionals.Optional[[]string]

	// Whether command-line completions should be restricted to directory names.
	DirNames bool
}

// Adds the given boolean-valued flag to the given command.
func AddBoolFlag(
	cmd *cobra.Command,
	f Flag[bool],
) *bool {
	flags := f.getFlagSet(cmd)
	return addFlag(flags, flags.BoolP, f)
}

// Adds the given string-valued flag to the given command.
func AddStringFlag(
	cmd *cobra.Command,
	f Flag[string],
) *string {
	flags := f.getFlagSet(cmd)
	return addFlag(flags, flags.StringP, f)
}

// Adds the given string-slice-valued flag to the given command.
func AddStringSliceFlag(
	cmd *cobra.Command,
	f Flag[[]string],
) *[]string {
	flags := f.getFlagSet(cmd)
	return addFlag(flags, flags.StringSliceP, f)
}

// Returns the FlagSet corresponding to this flag.
func (f Flag[T]) getFlagSet(cmd *cobra.Command) *pflag.FlagSet {
	if f.Persistent {
		return cmd.PersistentFlags()
	}
	return cmd.Flags()
}

// Adds the given flag to the given command.
func addFlag[T any](
	flagSet *pflag.FlagSet,
	defineFlag func(name string, shorthand string, value T, usage string) *T,
	f Flag[T],
) *T {
	if reflect.TypeFor[T]().Kind() == reflect.Slice {
		f.UsageMsg = fmt.Sprintf("%s. Can be specified multiple times", f.UsageMsg)
	}

	shortName := ""
	if r, exists := f.ShortName.Get(); exists {
		shortName = string(r)
	}
	result := defineFlag(f.Name, shortName, f.DefaultValue, f.UsageMsg)

	if f.Required {
		cobra.MarkFlagRequired(flagSet, f.Name)
	}
	if f.Hidden {
		flagSet.MarkHidden(f.Name)
	}
	if msg, deprecated := f.DeprecationMsg.Get(); deprecated {
		flagSet.MarkDeprecated(f.Name, msg)
	}

	if exts, exists := f.FilenameExts.Get(); exists {
		cobra.MarkFlagFilename(flagSet, f.Name, exts...)
	}
	if f.DirNames {
		cobra.MarkFlagDirname(flagSet, f.Name)
	}

	return result
}
