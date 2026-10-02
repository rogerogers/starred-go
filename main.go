package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

const version = "4.3.0-go"

func printHelp() {
	fmt.Printf(`starred (Go rewrite) v%s

Usage:
  starred [options]

Examples:
  # Output to stdout (or redirect to file)
  starred --username maguowei --token=xxxxxxxx --sort > README.md

  # Directly commit to a GitHub repository
  starred --username maguowei --token=xxxxxxxx --repository awesome-stars

Options:
  -u, --username      GitHub username [env: USER or GITHUB_USER]
  -t, --token         GitHub personal access token [env: GITHUB_TOKEN]
      --sort          Sort categories alphabetically (default: order of star)
      --topic         Categorize by repository topics instead of languages
      --topic-limit   Topic stargazer count threshold (default: 500)
  -r, --repository    GitHub repository name to commit to
  -f, --filename      File name to commit or save (default: README.md)
  -m, --message       Commit message (default: "update awesome-stars, created by starred-go")
      --private       Include private repositories (default: false)
  -o, --out           Output directly to a local file path
  -v, --version       Show version information
  -h, --help          Show this help message
`, version)
}

func parseFlags() (*Config, error) {
	cfg := &Config{
		TopicLimit: 500,
		Filename:   "README.md",
		Message:    "update awesome-stars, created by starred-go",
	}

	// Environment variable defaults
	if u := os.Getenv("GITHUB_USER"); u != "" {
		cfg.Username = u
	} else if u := os.Getenv("USER"); u != "" {
		cfg.Username = u
	}
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		cfg.Token = tok
	}

	fs := flag.NewFlagSet("starred", flag.ContinueOnError)
	fs.Usage = printHelp

	fs.StringVar(&cfg.Username, "username", cfg.Username, "GitHub username")
	fs.StringVar(&cfg.Username, "u", cfg.Username, "GitHub username (shorthand)")

	fs.StringVar(&cfg.Token, "token", cfg.Token, "GitHub token")
	fs.StringVar(&cfg.Token, "t", cfg.Token, "GitHub token (shorthand)")

	fs.BoolVar(&cfg.Sort, "sort", false, "Sort by category name alphabetically")
	fs.BoolVar(&cfg.Topic, "topic", false, "Category by topic, default is by language")

	fs.IntVar(&cfg.TopicLimit, "topic-limit", cfg.TopicLimit, "Topic stargazer_count threshold")
	fs.IntVar(&cfg.TopicLimit, "topic_limit", cfg.TopicLimit, "Topic stargazer_count threshold (alias)")

	fs.StringVar(&cfg.Repository, "repository", "", "Repository name to commit to")
	fs.StringVar(&cfg.Repository, "r", "", "Repository name (shorthand)")

	fs.StringVar(&cfg.Filename, "filename", cfg.Filename, "Target filename")
	fs.StringVar(&cfg.Filename, "f", cfg.Filename, "Target filename (shorthand)")

	fs.StringVar(&cfg.Message, "message", cfg.Message, "Commit message")
	fs.StringVar(&cfg.Message, "m", cfg.Message, "Commit message (shorthand)")

	fs.BoolVar(&cfg.Private, "private", false, "Include private repos")
	fs.StringVar(&cfg.OutputFile, "out", "", "Output file path")
	fs.StringVar(&cfg.OutputFile, "o", "", "Output file path (shorthand)")

	fs.BoolVar(&cfg.ShowVersion, "version", false, "Show version")
	fs.BoolVar(&cfg.ShowVersion, "v", false, "Show version (shorthand)")

	// Normalize arguments (e.g. support `--topic_limit=100`)
	var normalizedArgs []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--topic_limit") {
			arg = strings.Replace(arg, "--topic_limit", "--topic-limit", 1)
		}
		normalizedArgs = append(normalizedArgs, arg)
	}

	if err := fs.Parse(normalizedArgs); err != nil {
		return nil, err
	}

	return cfg, nil
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.ShowVersion {
		fmt.Printf("starred version %s\n", version)
		os.Exit(0)
	}

	if cfg.Username == "" {
		fmt.Fprintln(os.Stderr, "Error: missing required argument --username (or $GITHUB_USER / $USER)")
		os.Exit(1)
	}
	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Error: missing required argument --token (or $GITHUB_TOKEN)")
		os.Exit(1)
	}

	ctx := context.Background()
	client := NewGitHubClient(cfg.Token)

	// Fetch starred repositories
	repos, err := client.FetchStarredRepos(ctx, cfg.Username, cfg.TopicLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching stars: %v\n", err)
		os.Exit(1)
	}

	// Generate markdown content
	markdown := GenerateMarkdown(cfg.Username, repos, cfg.Sort, cfg.Topic, cfg.Private)

	// Case 1: Push directly to GitHub repository
	if cfg.Repository != "" {
		fmt.Fprintf(os.Stderr, "Committing to %s/%s (%s)...\n", cfg.Username, cfg.Repository, cfg.Filename)
		repoURL, err := client.SyncToRepository(ctx, cfg.Username, cfg.Repository, cfg.Filename, cfg.Message, markdown)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error syncing to repository: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Successfully updated: %s\n", repoURL)
		_ = OpenInBrowser(repoURL)
		return
	}

	// Case 2: Output to file
	if cfg.OutputFile != "" {
		if err := os.WriteFile(cfg.OutputFile, []byte(markdown), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file '%s': %v\n", cfg.OutputFile, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Successfully written to %s\n", cfg.OutputFile)
		return
	}

	// Case 3: Output to stdout
	fmt.Print(markdown)
}
