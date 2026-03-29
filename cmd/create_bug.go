package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gabrielluong/create-bug/internal/bugzilla"
	"github.com/gabrielluong/create-bug/internal/client"
	"github.com/gabrielluong/create-bug/internal/config"

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
		jsonFlag    bool
	)

	cmd.Args = cobra.MaximumNArgs(1)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

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
		resolvedVersion := mergeFlag(cmd, "version", version, d.Version)
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
			{"--version", resolvedVersion},
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
		fmt.Fprintf(os.Stdout, "%s/show_bug.cgi?id=%d\n", cfg.BaseURL, result.ID)
		return nil
	}

	cmd.Flags().StringVar(&product, "product", "", "product the bug is filed against")
	cmd.Flags().StringVar(&component, "component", "", "component within the product")
	cmd.Flags().StringVar(&summary, "summary", "", "brief description of the bug")
	cmd.Flags().StringVar(&version, "version", "", "product version the bug was found in")
	cmd.Flags().StringVar(&bugType, "type", "", "bug type (defect, enhancement, task)")
	cmd.Flags().StringVar(&description, "description", "", "initial comment / bug description")
	cmd.Flags().StringVar(&priority, "priority", "", "priority (P1–P5)")
	cmd.Flags().StringVar(&severity, "severity", "", "severity (S1–S4, enhancement, normal)")
	cmd.Flags().StringVar(&platform, "platform", "", "hardware platform (All, x86, x86_64, ...)")
	cmd.Flags().StringVar(&opSys, "os", "", "operating system (All, Linux, Windows, macOS, ...)")
	cmd.Flags().StringVar(&assignedTo, "assigned-to", "", "assign the bug to this user")
	cmd.Flags().StringVar(&cc, "cc", "", "comma-separated list of CC emails")
	cmd.Flags().StringVar(&blocks, "blocks", "", "comma-separated bug IDs that this bug blocks")
	cmd.Flags().StringVar(&dependsOn, "depends-on", "", "comma-separated bug IDs that this bug depends on")
	cmd.Flags().StringVar(&alias, "alias", "", "short alias for the bug")
	cmd.Flags().StringVar(&status, "status", "", "initial status (default: UNCONFIRMED or NEW)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "output raw JSON (for Claude integration)")

	cmd.RegisterFlagCompletionFunc("component", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		product, _ := cmd.Flags().GetString("product")
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
