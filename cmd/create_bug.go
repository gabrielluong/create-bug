package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielluong/create-bug/internal/bugzilla"
	"github.com/gabrielluong/create-bug/internal/client"
	"github.com/gabrielluong/create-bug/internal/config"
	"github.com/gabrielluong/create-bug/internal/history"

	"github.com/spf13/cobra"
)

func setupCreateBugCmd(cmd *cobra.Command) {
	var (
		product     string
		component   string
		summary     string
		version     string
		bugType     string
		description string
		priority    string
		severity    string
		platform    string
		opSys       string
		assignedTo  string
		cc          string
		blocks      string
		dependsOn   string
		alias       string
		status      string
		jsonFlag         bool
		historyFlag      bool
		clearHistoryFlag bool
	)

	cmd.Args = cobra.MaximumNArgs(1)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		if clearHistoryFlag {
			return clearHistory()
		}

		if historyFlag {
			return showHistory(cfg, jsonFlag)
		}

		// Resolve summary: positional arg > --summary flag > error
		resolvedSummary := summary
		if len(args) > 0 {
			resolvedSummary = args[0]
		}
		if resolvedSummary == "" {
			fmt.Fprintln(os.Stderr, "Error: required argument 'summary' not specified")
			return fmt.Errorf("required argument 'summary' not specified")
		}

		if cfg.BaseURL == "" {
			fmt.Fprintln(os.Stderr, "Error: BUGZILLA_URL or baseUrl in config is required")
			return fmt.Errorf("BUGZILLA_URL or baseUrl in config is required")
		}

		if cfg.APIKey == "" {
			fmt.Fprintln(os.Stderr, "Error: BUGZILLA_API_KEY is required to create bugs")
			return fmt.Errorf("BUGZILLA_API_KEY is required to create bugs")
		}

		d := cfg.Defaults

		// Merge CLI flags with config defaults
		resolvedProduct := mergeFlag(cmd, "product", product, d.Product)
		resolvedComponent := mergeFlag(cmd, "component", component, d.Component)
		resolvedVersion := mergeFlag(cmd, "bug-version", version, d.Version)
		resolvedType := mergeFlag(cmd, "type", bugType, d.Type)
		resolvedPlatform := mergeFlag(cmd, "platform", platform, d.Platform)
		resolvedOS := mergeFlag(cmd, "os", opSys, d.OS)
		resolvedPriority := mergeFlag(cmd, "priority", priority, d.Priority)
		resolvedSeverity := mergeFlag(cmd, "severity", severity, d.Severity)

		// Validate required fields
		for _, check := range []struct {
			flag string
			val  string
		}{
			{"--product", resolvedProduct},
			{"--component", resolvedComponent},
			{"--bug-version", resolvedVersion},
			{"--type", resolvedType},
		} {
			if check.val == "" {
				msg := fmt.Sprintf("required option '%s <value>' not specified and no default set in config", check.flag)
				fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
				return fmt.Errorf("%s", msg)
			}
		}

		resolvedComponent, err := bugzilla.ResolveComponent(resolvedProduct, resolvedComponent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return err
		}

		params := client.CreateBugParams{
			Product:   resolvedProduct,
			Component: resolvedComponent,
			Summary:   resolvedSummary,
			Version:   resolvedVersion,
			Type:      resolvedType,
			Platform:  resolvedPlatform,
			OpSys:     resolvedOS,
			Priority:  resolvedPriority,
			Severity:  resolvedSeverity,
		}

		if cmd.Flags().Changed("description") {
			params.Description = description
		}
		if cmd.Flags().Changed("assigned-to") {
			params.AssignedTo = assignedTo
		}
		if cmd.Flags().Changed("alias") {
			params.Alias = alias
		}
		if cmd.Flags().Changed("status") {
			params.Status = status
		}
		if cmd.Flags().Changed("cc") {
			params.CC = splitTrimmed(cc)
		}
		if cmd.Flags().Changed("blocks") {
			ids, err := splitInts(blocks)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid --blocks value: %s\n", err)
				return err
			}
			params.Blocks = ids
		}
		if cmd.Flags().Changed("depends-on") {
			ids, err := splitInts(dependsOn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid --depends-on value: %s\n", err)
				return err
			}
			params.DependsOn = ids
		}

		result, err := bugzilla.CreateBug(cfg, params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return err
		}

		bugURL := fmt.Sprintf("%s/show_bug.cgi?id=%d", cfg.BaseURL, result.ID)
		_ = history.Append(history.Entry{
			ID:        result.ID,
			Summary:   resolvedSummary,
			Product:   resolvedProduct,
			Component: resolvedComponent,
			URL:       bugURL,
			CreatedAt: time.Now(),
		}, cfg.HistorySize)

		if jsonFlag {
			out, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}
			fmt.Fprintln(os.Stdout, string(out))
			return nil
		}

		fmt.Fprintf(os.Stdout, "Created Bug %d\n", result.ID)
		fmt.Fprintln(os.Stdout, bugURL)
		return nil
	}

	cmd.Flags().StringVarP(&product, "product", "p", "", "product the bug is filed against")
	cmd.Flags().StringVarP(&component, "component", "c", "", "component within the product")
	cmd.Flags().StringVar(&summary, "summary", "", "brief description of the bug")
	cmd.Flags().StringVar(&version, "bug-version", "", "product version the bug was found in")
	cmd.Flags().StringVar(&bugType, "type", "", "bug type (defect, enhancement, task)")
	cmd.Flags().StringVar(&description, "description", "", "initial comment / bug description")
	cmd.Flags().StringVar(&priority, "priority", "", "priority (P1–P5)")
	cmd.Flags().StringVar(&severity, "severity", "", "severity (S1–S4, enhancement, normal)")
	cmd.Flags().StringVar(&platform, "platform", "", "hardware platform (All, x86, x86_64, ...)")
	cmd.Flags().StringVar(&opSys, "os", "", "operating system (All, Linux, Windows, macOS, ...)")
	cmd.Flags().StringVar(&assignedTo, "assigned-to", "", "assign the bug to this user")
	cmd.Flags().StringVar(&cc, "cc", "", "comma-separated list of CC emails")
	cmd.Flags().StringVarP(&blocks, "blocks", "b", "", "comma-separated bug IDs that this bug blocks")
	cmd.Flags().StringVarP(&dependsOn, "depends-on", "d", "", "comma-separated bug IDs that this bug depends on")
	cmd.Flags().StringVar(&alias, "alias", "", "short alias for the bug")
	cmd.Flags().StringVar(&status, "status", "", "initial status (default: UNCONFIRMED or NEW)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "output raw JSON (for Claude integration)")
	cmd.Flags().BoolVarP(&historyFlag, "history", "H", false, "show recently filed bugs")
	cmd.Flags().BoolVar(&clearHistoryFlag, "clear-history", false, "clear the history of filed bugs")

	cmd.RegisterFlagCompletionFunc("component", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		product, _ := cmd.Flags().GetString("product")
		if product == "" {
			product = config.Load().Defaults.Product
		}
		if components, ok := bugzilla.ProductComponents[product]; ok {
			return components, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
}

func mergeFlag(cmd *cobra.Command, name, flagVal, defaultVal string) string {
	if cmd.Flags().Changed(name) {
		return flagVal
	}
	return defaultVal
}

func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func clearHistory() error {
	if err := history.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to clear history: %s\n", err)
		return err
	}
	fmt.Fprintln(os.Stdout, "History cleared.")
	return nil
}

func showHistory(cfg *config.Config, jsonOutput bool) error {
	entries, err := history.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load history: %s\n", err)
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stdout, "No bugs filed yet.")
		return nil
	}

	if jsonOutput {
		out, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return err
		}
		fmt.Fprintln(os.Stdout, string(out))
		return nil
	}

	for _, e := range entries {
		fmt.Fprintf(os.Stdout, "Bug %d  %s  [%s :: %s]\n", e.ID, e.Summary, e.Product, e.Component)
		fmt.Fprintf(os.Stdout, "  %s  (%s)\n", e.URL, e.CreatedAt.Format("2006-01-02 15:04"))
	}
	return nil
}

func splitInts(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid bug ID", trimmed)
		}
		result = append(result, n)
	}
	return result, nil
}
