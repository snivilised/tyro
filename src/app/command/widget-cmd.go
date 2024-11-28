package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/snivilised/cobrass/src/assistant"
	"github.com/snivilised/li18ngo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/snivilised/arcadia/src/app/domain"
	"github.com/snivilised/arcadia/src/locale"
)

// CLIENT-TODO: rename this widget command to something required
// by the application. Widget command is meant to just serve as
// an aid in creating custom commands and intended to either be
// replaced or renamed.

const widgetPsName = "widget-ps"

func (b *Bootstrap) buildWidgetCommand(container *assistant.CobraContainer) *cobra.Command {
	// to test: arcadia widget -d ./some-existing-file -p "P?<date>" -t 30
	//
	widgetCommand := &cobra.Command{
		Use:   "widget",
		Short: li18ngo.Text(locale.WidgetCmdShortDescTemplData{}),
		Long:  li18ngo.Text(locale.WidgetCmdLongDescTemplData{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			var appErr error

			ps := container.MustGetParamSet(widgetPsName).(domain.WidgetParamSetPtr) //nolint:errcheck // is Must call

			if err := ps.Validate(); err == nil {
				native := ps.Native

				// rebind enum into native member
				// (eventually, Format/OutputFormatEn will be combined into
				// a single entity), see https://github.com/snivilised/cobrass/issues/147
				//
				native.Format = native.FormatEn.Value()

				// optionally invoke cross field validation
				//
				if xv := ps.CrossValidate(func(ps *domain.WidgetParameterSet) error {
					condition := (ps.Format == domain.XMLFormatEn)
					if condition {
						return nil
					}
					return fmt.Errorf("format: '%v' is not currently supported", ps.Format)
				}); xv == nil {
					options := []string{}
					cmd.Flags().Visit(func(f *pflag.Flag) {
						options = append(options, fmt.Sprintf("--%v=%v", f.Name, f.Value))
					})
					fmt.Printf("%v %v Running widget, with options: '%v', args: '%v'\n",
						AppEmoji, ApplicationName, options, args,
					)

					appErr = domain.EnterWidget(native)
				} else {
					return xv
				}
			} else {
				return err
			}

			return appErr
		},
	}

	defaultDirectory := "/default-directory"
	paramSet := assistant.NewParamSet[domain.WidgetParameterSet](widgetCommand)
	paramSet.BindValidatedString(
		assistant.NewFlagInfo("directory", "d", defaultDirectory),
		&paramSet.Native.Directory,
		func(value string, _ *pflag.Flag) error {
			// ideally, we should check if the Flag has been explicitly set
			//
			if value == defaultDirectory {
				return nil
			}

			if _, err := os.Stat(value); err != nil {
				if os.IsNotExist(err) {
					return err
				}
			}

			return nil
		},
	)

	paramSet.Native.FormatEn = domain.OutputFormatEnumInfo.NewValue()

	paramSet.BindValidatedEnum(
		assistant.NewFlagInfo("format", "f", "xml"),
		&paramSet.Native.FormatEn.Source,
		func(value string, _ *pflag.Flag) error {
			if domain.OutputFormatEnumInfo.En(value) == domain.XMLFormatEn {
				return nil
			}

			return fmt.Errorf(
				"only xml format is currently supported, other formats available in future release",
			)
		},
	)

	paramSet.BindBool(
		assistant.NewFlagInfo("concise", "c", false),
		&paramSet.Native.Concise,
	)

	paramSet.BindValidatedString(
		assistant.NewFlagInfo("pattern", "p", ""),
		&paramSet.Native.Pattern,
		func(value string, _ *pflag.Flag) error {
			result := strings.Contains(value, "P?<date>") ||
				(strings.Contains(value, "P?<d>") && strings.Contains(value, "P?<m>") &&
					strings.Contains(value, "P?<m>"))

			if result {
				return nil
			}

			return fmt.Errorf(
				"pattern is invalid, missing mandatory capture groups ('date' or 'd', 'm', and 'y')",
			)
		},
	)

	_ = widgetCommand.MarkFlagRequired("pattern")

	const (
		Low  = uint(25)
		High = uint(50)
		Def  = uint(10)
	)

	paramSet.BindValidatedUintWithin(
		assistant.NewFlagInfo("threshold", "t", Def),
		&paramSet.Native.Threshold,
		Low, High,
	)

	// If you want to disable the widget command but keep it in the project for reference
	// purposes, then simply comment out the following 2 register calls:
	// (Warning, this may just create dead code and result in lint failure so tread
	// carefully.)
	//
	container.MustRegisterRootedCommand(widgetCommand)
	container.MustRegisterParamSet(widgetPsName, paramSet)

	return widgetCommand
}
