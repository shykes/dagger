package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"dagger.io/dagger"
	"github.com/dagger/dagger/engine/client"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	queryFile          string
	queryVarsInput     []string
	queryVarsJSONInput string
)

var queryCmd = &cobra.Command{
	Use:                   "query [flags] [operation]",
	Aliases:               []string{"q"},
	DisableFlagsInUseLine: true,
	Long:                  "Send API queries to a dagger engine\n\nWhen no document file, read query from standard input.",
	Short:                 "Send API queries to a dagger engine",
	Example: `
dagger query <<EOF
{
  container {
    from(address:"hello-world") {
      exec(args:["/hello"]) {
        stdout
      }
    }
  }
}
EOF
`,
	Run:  Query,
	Args: cobra.MaximumNArgs(1), // operation can be specified
}

func Query(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	var operation string
	if len(args) > 0 {
		operation = args[0]
	}

	vars := make(map[string]interface{})
	if len(queryVarsJSONInput) > 0 {
		if err := json.Unmarshal([]byte(queryVarsJSONInput), &vars); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		vars = getKVInput(queryVarsInput)
	}

	// Use the provided query file if specified
	// Otherwise, if stdin is a pipe or other non-tty thing, read from it.
	// Finally, default to the operations returned by the loadExtension query
	var operations string
	if queryFile != "" {
		inBytes, err := os.ReadFile(queryFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		operations = string(inBytes)
	} else if !term.IsTerminal(int(os.Stdin.Fd())) {
		inBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		operations = string(inBytes)
	}

	result, err := doQuery(ctx, operations, operation, vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", result)
}

func doQuery(ctx context.Context, query, op string, vars map[string]interface{}) ([]byte, error) {
	res := make(map[string]interface{})
	err := withEngineAndTUI(ctx, client.Params{}, func(ctx context.Context, engineClient *client.Client) error {
		c, err := dagger.Connect(ctx, dagger.WithConn(EngineConn(engineClient)))
		if err != nil {
			return fmt.Errorf("connect to dagger: %w", err)
		}

		if environmentURI != "" && environmentURI != environmentURIDefault {
			fmt.Fprintf(os.Stderr, "Loading environment from %s\n", environmentURI)
			env, err := loadEnv(ctx, c)
			if err != nil {
				return err
			}
			// TODO: forces eval, ugly
			_, err = env.ID(ctx)
			if err != nil {
				return err
			}
		}

		err = engineClient.Do(ctx, query, op, vars, &res)
		return err
	})
	if err != nil {
		return nil, err
	}
	result, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getKVInput(kvs []string) map[string]interface{} {
	m := make(map[string]interface{})
	for _, kv := range kvs {
		split := strings.SplitN(kv, "=", 2)
		m[split[0]] = split[1]
	}
	return m
}

func init() {
	// changing usage template so examples are displayed below flags
	queryCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}
Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	queryCmd.Flags().StringVar(&queryFile, "doc", "", "document query file")
	queryCmd.Flags().StringSliceVar(&queryVarsInput, "var", nil, "query variable")
	queryCmd.Flags().StringVar(&queryVarsJSONInput, "var-json", "", "json query variables (overrides --var)")
}
