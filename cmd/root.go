package cmd

import (
	"fmt"
	"github.com/Benbentwo/utils/util"
	"log"
	"os"

	"github.com/Benbentwo/go-markdown2confluence/pkg"

	"github.com/spf13/cobra"
)

var m pkg.Markdown2Confluence

func init() {
	log.SetFlags(0)

	// Property Load order from lowest to highest should be
	// 		1. Default Config.yml
	// 		2. Passed config.yml
	// 		3. Env Vars
	// 		4. Override flags from CLI
	RootCmd.PersistentFlags().BoolVarP(&m.DryRun, "dry-run", "", false, "Print config but don't actually run anything first, Warning, prints PW to console")
	m.LoadFromConfig = RootCmd.PersistentFlags().StringArrayP("load-from", "l", nil, "Set the files to load configuration from, prioritizes first input over others")
	m.SourceEnvironmentVariables()

	RootCmd.PersistentFlags().StringVarP(&m.RunAllFiles, "all", "", "", "run all files matching a string name")
	RootCmd.Flags().SetInterspersed(false)
	RootCmd.PersistentFlags().StringVarP(&m.Space, "space", "s", "", "Space in which page should be created")
	RootCmd.PersistentFlags().StringVarP(&m.Username, "username", "u", "", "Confluence username. (Alternatively set CONFLUENCE_USERNAME environment variable)")
	RootCmd.PersistentFlags().StringVarP(&m.Password, "password", "p", "", "Confluence password. (Alternatively set CONFLUENCE_PASSWORD environment variable)")
	RootCmd.PersistentFlags().StringVarP(&m.Endpoint, "endpoint", "e", pkg.DefaultEndpoint, "Confluence endpoint. (Alternatively set CONFLUENCE_ENDPOINT environment variable)")
	RootCmd.PersistentFlags().StringVar(&m.Parent, "parent", "", "Optional parent page to next content under")
	RootCmd.PersistentFlags().BoolVarP(&m.Debug, "debug", "d", false, "Enable debug logging")
	RootCmd.PersistentFlags().IntVarP(&m.Since, "modified-since", "m", 0, "Only upload files that have modifed in the past n minutes")
	RootCmd.PersistentFlags().StringVarP(&m.Title, "title", "t", "", "Set the page title on upload (defaults to filename without extension)")

}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "markdown2confluence",
	Short: "Push markdown files to Confluence Cloud",
	Run: func(rootCmd *cobra.Command, args []string) {
		if m.RunAllFiles != "" {
			errs := m.RunAllConfigs()
			errorsFound := false
			for path, err := range errs {
				if err != nil {
					fmt.Printf(util.ColorError("ERROR: ")+"config %s failed, cause: %s \n", path, err)
					errorsFound = true
				}
			}
			if errorsFound {
				os.Exit(1)
			} else {
				return
			}
		}

		err := m.LoadConfig()
		if err != nil {
			log.Fatalf("loading config: %s", err)
		}

		if len(args) > 0 {
			m.SourceMarkdown = args
		}
		m.SourceEnvironmentVariables()
		// Validate the arguments
		err = m.Validate()
		if err != nil {
			log.Fatal(err)
		}

		if m.DryRun {
			m.PrintMe()
			return
		}
		errors := m.Run()
		for _, err := range errors {
			fmt.Println()
			fmt.Println(err)
		}
		if len(errors) > 0 {
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute(version string) {
	RootCmd.Version = version
	RootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "%s" .Version}}
`)
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
