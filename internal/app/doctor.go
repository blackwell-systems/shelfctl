package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	ghclient "github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// CheckResult holds the outcome of a single doctor check.
type CheckResult struct {
	Label   string
	Status  string // "pass", "fail", "warn", "skip"
	Message string
	Detail  string // optional remediation hint
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check shelfctl setup health",
		Long: `Run a series of health checks and report issues with remediation hints.

Checks run in order; later checks are skipped if an earlier prerequisite fails.

Exit code 0 if all checks pass or warn; non-zero if any check fails.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			allOK, err := runDoctor(os.Stdout)
			if err != nil {
				return err
			}
			if !allOK {
				os.Exit(1)
			}
			return nil
		},
	}
}

// runDoctor executes all health checks sequentially and writes formatted output to w.
// Returns (allOK bool, err error). allOK is false if any check has Status "fail".
func runDoctor(w io.Writer) (bool, error) {
	fmt.Fprintln(w, "shelfctl doctor")
	fmt.Fprintln(w)

	var results []CheckResult

	// Check 1: Config file
	cfgPath := config.DefaultPath()
	if envPath := os.Getenv("SHELFCTL_CONFIG"); envPath != "" {
		cfgPath = envPath
	}
	cfgResult, loadedCfg := checkConfig(cfgPath)
	results = append(results, cfgResult)

	// Check 2: GitHub token
	var tokenResult CheckResult
	if loadedCfg != nil {
		tokenResult = checkToken(loadedCfg)
	} else {
		tokenResult = CheckResult{
			Label:   "GitHub token",
			Status:  "skip",
			Message: "skipped (config not loaded)",
		}
	}
	results = append(results, tokenResult)

	// Checks 3-5 require a working token
	var ghClient *ghclient.Client
	if loadedCfg != nil && loadedCfg.GitHub.Token != "" {
		ghClient = ghclient.New(loadedCfg.GitHub.Token, loadedCfg.GitHub.APIBase)
	}

	var connResult, scopeResult CheckResult
	if ghClient == nil {
		connResult = CheckResult{Label: "API connectivity", Status: "skip", Message: "skipped (no token)"}
		scopeResult = CheckResult{Label: "Token scopes", Status: "skip", Message: "skipped (no token)"}
	} else {
		apiBase := loadedCfg.GitHub.APIBase
		if apiBase == "" {
			apiBase = "https://api.github.com"
		}
		connResult, scopeResult = checkAPIAndScopes(ghClient, apiBase)
	}
	results = append(results, connResult, scopeResult)

	// Shelf accessibility checks
	if ghClient == nil || loadedCfg == nil {
		results = append(results, CheckResult{
			Label:   "Shelf accessibility",
			Status:  "skip",
			Message: "skipped (no token)",
		})
	} else if len(loadedCfg.Shelves) == 0 {
		results = append(results, CheckResult{
			Label:   "Shelf accessibility",
			Status:  "warn",
			Message: "no shelves configured",
			Detail:  "run: shelfctl init --repo shelf-<name> --name <name> --create-repo",
		})
	} else {
		for i := range loadedCfg.Shelves {
			shelf := &loadedCfg.Shelves[i]
			owner := shelf.EffectiveOwner(loadedCfg.GitHub.Owner)
			r := checkShelf(ghClient, shelf.Name, owner, shelf.Repo)
			results = append(results, r)
		}
	}

	// Cache integrity
	var cacheResult CheckResult
	if loadedCfg == nil {
		cacheResult = CheckResult{Label: "Cache integrity", Status: "skip", Message: "skipped (config not loaded)"}
	} else {
		cacheResult = checkCache(loadedCfg, ghClient)
	}
	results = append(results, cacheResult)

	// Print results and summary
	failCount := 0
	warnCount := 0
	for _, r := range results {
		printCheckResult(w, r)
		if r.Status == "fail" {
			failCount++
		} else if r.Status == "warn" {
			warnCount++
		}
	}
	fmt.Fprintln(w)
	if failCount > 0 {
		fmt.Fprintf(w, "%s\n", color.RedString("Doctor found %d failure(s). Fix issues above and re-run.", failCount))
		return false, nil
	}
	if warnCount > 0 {
		fmt.Fprintf(w, "%s\n", color.YellowString("All checks passed (%d warning(s)).", warnCount))
	} else {
		fmt.Fprintf(w, "%s\n", color.GreenString("All checks passed."))
	}
	return true, nil
}

// checkConfig verifies the config file exists and parses cleanly.
func checkConfig(cfgPath string) (CheckResult, *config.Config) {
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return CheckResult{
			Label:   "Config file",
			Status:  "fail",
			Message: cfgPath + " not found",
			Detail:  "run: shelfctl init --repo shelf-<name> --name <name> --create-repo",
		}, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return CheckResult{
			Label:   "Config file",
			Status:  "fail",
			Message: "parse error: " + err.Error(),
			Detail:  "check YAML syntax or delete and re-run: shelfctl init",
		}, nil
	}
	if cfg.GitHub.Owner == "" {
		return CheckResult{
			Label:   "Config file",
			Status:  "warn",
			Message: cfgPath + " — owner not set",
			Detail:  "add 'github.owner' to config or pass --owner to init",
		}, cfg
	}
	return CheckResult{
		Label:   "Config file",
		Status:  "pass",
		Message: cfgPath,
	}, cfg
}

// checkToken verifies the GitHub token env var is set and non-empty.
func checkToken(cfg *config.Config) CheckResult {
	tokenEnv := cfg.GitHub.TokenEnv
	if tokenEnv == "" {
		tokenEnv = "SHELFCTL_GITHUB_TOKEN"
	}
	if cfg.GitHub.Token == "" {
		return CheckResult{
			Label:   "GitHub token",
			Status:  "fail",
			Message: tokenEnv + " is not set",
			Detail:  fmt.Sprintf("run: export %s=<your-token>", tokenEnv),
		}
	}
	return CheckResult{
		Label:   "GitHub token",
		Status:  "pass",
		Message: tokenEnv + " is set",
	}
}

// checkAPIAndScopes checks connectivity to the GitHub API and inspects token scopes.
func checkAPIAndScopes(gh *ghclient.Client, apiBase string) (connResult, scopeResult CheckResult) {
	start := time.Now()
	login, scopes, err := gh.GetUser()
	elapsed := time.Since(start)

	if err != nil {
		var connMsg string
		if errors.Is(err, ghclient.ErrUnauthorized) {
			connMsg = "token rejected (401 Unauthorized)"
		} else if errors.Is(err, ghclient.ErrForbidden) {
			connMsg = "token forbidden (403)"
		} else {
			connMsg = "unreachable: " + err.Error()
		}
		connResult = CheckResult{
			Label:   "API connectivity",
			Status:  "fail",
			Message: apiBase + " — " + connMsg,
			Detail:  "check network, token, and api_base in config",
		}
		scopeResult = CheckResult{
			Label:   "Token scopes",
			Status:  "skip",
			Message: "skipped (API unreachable)",
		}
		return
	}

	connResult = CheckResult{
		Label:   "API connectivity",
		Status:  "pass",
		Message: fmt.Sprintf("%s reachable (%dms) — logged in as %s", apiBase, elapsed.Milliseconds(), login),
	}

	scopeList := strings.Split(scopes, ",")
	hasRepo := false
	for _, s := range scopeList {
		s = strings.TrimSpace(s)
		if s == "repo" || s == "public_repo" {
			hasRepo = true
			break
		}
	}
	if !hasRepo {
		scopeResult = CheckResult{
			Label:   "Token scopes",
			Status:  "warn",
			Message: fmt.Sprintf("repo scope missing (have: %s)", strings.TrimSpace(scopes)),
			Detail:  "generate a new token with 'repo' scope at https://github.com/settings/tokens",
		}
	} else {
		scopeResult = CheckResult{
			Label:   "Token scopes",
			Status:  "pass",
			Message: "repo \u2713",
		}
	}
	return
}

// checkShelf verifies that a shelf's GitHub repo is accessible.
func checkShelf(gh *ghclient.Client, shelfName, owner, repo string) CheckResult {
	label := "Shelf: " + shelfName
	r, err := gh.GetRepo(owner, repo)
	if err != nil {
		if errors.Is(err, ghclient.ErrNotFound) {
			return CheckResult{
				Label:   label,
				Status:  "fail",
				Message: fmt.Sprintf("%s/%s not found", owner, repo),
				Detail:  "check that the repo exists and the token has access",
			}
		}
		return CheckResult{
			Label:   label,
			Status:  "fail",
			Message: fmt.Sprintf("%s/%s — %v", owner, repo, err),
		}
	}
	return CheckResult{
		Label:   label,
		Status:  "pass",
		Message: r.FullName,
	}
}

// checkCache runs orphan detection and reports count/size.
func checkCache(cfg *config.Config, gh *ghclient.Client) CheckResult {
	mgr := cache.New(cfg.Defaults.CacheDir)

	var shelves []cache.ShelfCatalog
	if gh != nil {
		for i := range cfg.Shelves {
			shelf := &cfg.Shelves[i]
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			data, _, err := gh.GetFileContent(owner, shelf.Repo, shelf.EffectiveCatalogPath(), "")
			if err != nil {
				continue
			}
			books, err := catalog.Parse(data)
			if err != nil {
				continue
			}
			shelves = append(shelves, cache.ShelfCatalog{
				Owner: owner,
				Repo:  shelf.Repo,
				Books: books,
			})
		}
	}

	report, err := mgr.DetectOrphans(shelves)
	if err != nil {
		return CheckResult{
			Label:   "Cache integrity",
			Status:  "warn",
			Message: "orphan detection error: " + err.Error(),
		}
	}
	if report.TotalCount == 0 {
		return CheckResult{
			Label:   "Cache integrity",
			Status:  "pass",
			Message: "no orphaned files",
		}
	}
	return CheckResult{
		Label:   "Cache integrity",
		Status:  "warn",
		Message: fmt.Sprintf("%d orphaned file(s) (%s)", report.TotalCount, util.HumanBytes(report.TotalSize)),
		Detail:  "run: shelfctl cache clear --orphans",
	}
}

// printCheckResult writes one formatted check line to w.
func printCheckResult(w io.Writer, r CheckResult) {
	var icon string
	switch r.Status {
	case "pass":
		icon = color.GreenString("\u2713")
	case "fail":
		icon = color.RedString("\u2717")
	case "warn":
		icon = color.YellowString("\u26A0")
	default:
		icon = color.HiBlackString("-")
	}
	label := fmt.Sprintf("%-20s", r.Label)
	fmt.Fprintf(w, "  %s %s %s\n", icon, color.CyanString(label), r.Message)
	if r.Detail != "" {
		fmt.Fprintf(w, "    %s\n", color.YellowString("hint: "+r.Detail))
	}
}
