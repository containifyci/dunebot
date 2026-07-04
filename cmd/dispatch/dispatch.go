package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	gogithub "github.com/google/go-github/v88/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/containifyci/dunebot/cmd"
	"github.com/containifyci/dunebot/pkg/backoff"
	"github.com/containifyci/dunebot/pkg/github"
)

type dispatchCmdArgs struct {
	repo         string
	dryRun       bool
	installation int64
}

var dispatchArgs = &dispatchCmdArgs{}

// dispatchCmd represents the dispatch command
var dispatchCmd = &cobra.Command{
	Use:   "dispatch",
	Short: "This sends a repository dispatch event to GitHub.",
	Long:  `This sends a repository dispatch event to GitHub.`,
	RunE:  execute,
}

func init() {
	cmd.RootCmd.AddCommand(dispatchCmd)

	dispatchCmd.Flags().StringVar(&dispatchArgs.repo, "repo", "", "Only send repository dispatch event for this for this Github repository in the form of (owner/repo for example containifyci/ad-service).")
	dispatchCmd.Flags().BoolVar(&dispatchArgs.dryRun, "dry", false, "If true no repository dispatch event is sent only log the event to stdout.")
	dispatchCmd.Flags().Int64Var(&dispatchArgs.installation, "installation", 0, "Only process this specific installation ID (0 = all installations). Only used in GitHub App mode.")
}

type Payload struct {
	PullRequest *github.PullRequest `json:"pull_request"`
	Owner       string              `json:"owner"`
	Repository  string              `json:"repository"`
}

func (p *Payload) String() string {
	return fmt.Sprintf("Owner: %s, Repository: %s, PRNumber: %d,", p.Owner, p.Repository, *p.PullRequest.Number)
}

func execute(cmd *cobra.Command, args []string) error {
	logger := zerolog.New(os.Stdout).With().Caller().Stack().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg, err := LoadConfig()
	if err != nil {
		log.Error().Err(err).Msgf("loading config: %v", err)
		return err
	}

	if cfg.IsAppMode() {
		return runAppMode(cfg)
	}

	gh, err := github.NewClient(github.WithConfig(github.NewStaticTokenConfig(cfg.GithubToken)))
	if err != nil {
		log.Error().Err(err).Msgf("initialse github Client: %v", err)
		return err
	}
	return run(cfg, gh)
}

// TODO: instead of using this function to list all the repositories that use Dunebot.
// Just implement two simple searches
//   - for org filter the repositories that have the dunebot customer property
//     curl -L   -H "Accept: application/vnd.github+json"   -H "Authorization: Bearer ${GH_TOKEN}"   -H "X-GitHub-Api-Version: 2022-11-28"   "https://api.github.com/search/repositories?q=props.dunebot:true+org:containifyci"
func listDuneBotRepositories(ctx context.Context, cfg *dispatchConfig) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.RepositoryEndpoint, nil)
	if err != nil {
		log.Error().Err(err).Msgf("creating request: %v", err)
		return nil, err
	}

	// Set the Bearer token in the Authorization header
	req.Header.Set("Authorization", "Bearer "+cfg.GithubToken)

	// Make the GET request using http.Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("Error making GET request: %v", err)
		return nil, err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Error().Err(err).Msgf("Error closing response body: %v", err)
		}
	}()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		log.Error().Err(err).Msgf("received status code %d", resp.StatusCode)
		return nil, fmt.Errorf("received status code %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("reading response body: %v", err)
		return nil, err
	}

	// Parse the JSON array into a slice of Repository structs
	var repositories []string
	err = json.Unmarshal(body, &repositories)
	if err != nil {
		log.Error().Err(err).Msgf("reading response body: %s", string(body))
		log.Error().Err(err).Msgf("unmarshalling JSON: %v", err)
		return nil, err
	}
	return repositories, nil
}

func run(cfg *dispatchConfig, gh *github.GithubClient) error {
	ctx := context.Background()

	var repos []string
	if dispatchArgs.repo != "" {
		repos = []string{dispatchArgs.repo}
	} else if cfg.RepositoryEndpoint == "" {
		repositories, _, err := gh.Client.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{
			Visibility: "all",
		})
		if err != nil {
			log.Error().Err(err).Msgf("fetching repositories: %v", err)
			return err
		}
		for _, repo := range repositories {
			repos = append(repos, *repo.FullName)
		}
	} else {
		var err error
		repos, err = listDuneBotRepositories(ctx, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("fetching repositories: %v", err)
			return err
		}
	}

	return dispatchForRepos(ctx, gh, repos)
}

// runAppMode discovers repositories via GitHub App installations and dispatches events.
func runAppMode(cfg *dispatchConfig) error {
	ctx := context.Background()

	appClient, err := newAppClient(cfg)
	if err != nil {
		log.Error().Err(err).Msg("creating GitHub App client")
		return err
	}

	installations, err := listAllInstallations(ctx, appClient)
	if err != nil {
		log.Error().Err(err).Msg("listing installations")
		return err
	}
	log.Info().Msgf("Found %d installation(s)", len(installations))

	for _, install := range installations {
		if dispatchArgs.installation != 0 && install.GetID() != dispatchArgs.installation {
			log.Debug().Msgf("Skipping installation %d (filter: %d)", install.GetID(), dispatchArgs.installation)
			continue
		}

		log.Info().Msgf("Processing installation %d (%s)", install.GetID(), install.GetAccount().GetLogin())

		installClient, err := newInstallationClient(cfg, install.GetID())
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create installation client for installation %d", install.GetID())
			continue
		}

		ghClient, err := github.NewClient(github.WithGithubClient(installClient))
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create GithubClient for installation %d", install.GetID())
			continue
		}

		installRepos, err := listInstallationRepositories(ctx, installClient)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to list repositories for installation %d", install.GetID())
			continue
		}

		// Filter by --repo flag if set
		if dispatchArgs.repo != "" {
			found := false
			for _, r := range installRepos {
				if r == dispatchArgs.repo {
					installRepos = []string{r}
					found = true
					break
				}
			}
			if !found {
				log.Debug().Msgf("Repository %s not found in installation %d", dispatchArgs.repo, install.GetID())
				continue
			}
		}

		err = dispatchForRepos(ctx, ghClient, installRepos)
		if err != nil {
			log.Error().Err(err).Msgf("Error dispatching for installation %d", install.GetID())
		}
	}
	return nil
}

// dispatchForRepos iterates over repos, fetches open PRs, and dispatches events.
// Errors for individual repos/PRs are logged but do not stop processing of others.
func dispatchForRepos(ctx context.Context, gh *github.GithubClient, repos []string) error {
	var firstErr error
	prCount := 0
	for _, repo := range repos {
		log.Debug().Msgf("Repository: %s", repo)
		sp := strings.Split(repo, "/")
		if len(sp) != 2 {
			log.Error().Msgf("invalid repository format: %s, expected owner/repo", repo)
			continue
		}
		owner := sp[0]
		name := sp[1]

		if dispatchArgs.repo != "" && dispatchArgs.repo != repo {
			continue
		}

		// Fetch open pull requests for each repository
		prOpts := &github.PullRequestListOptions{
			State:       "open",
			ListOptions: github.ListOptions{PerPage: 50},
		}
		for {
			pullRequests, resp, err := gh.Client.PullRequests.List(ctx, owner, name, prOpts)
			if err != nil {
				log.Error().Err(err).Msgf("fetching pull requests for repository %s", repo)
				if firstErr == nil {
					firstErr = err
				}
				break
			}

			for _, pr := range pullRequests {
				log.Debug().Msgf("\tOpen PR: #%d %s", pr.GetNumber(), pr.GetTitle())
				//Only setting the required Pull Request fields like Number, State, User.Login, Head.Ref because repository_dispacth event has a strict payload size limit
				//https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#repository_dispatch
				miniPR := &github.PullRequest{
					Number: pr.Number,
					Title:  pr.Title,
					State:  pr.State,
					Head: &github.PullRequestBranch{
						Ref: pr.Head.Ref,
					},
					User: &github.User{
						Login: pr.User.Login,
					},
				}
				payLoad := Payload{
					PullRequest: miniPR,
					Owner:       owner,
					Repository:  name,
				}
				if dispatchArgs.dryRun {
					log.Info().Msgf("Dry run: Would dispatch event for %s", &payLoad)
					continue
				}
				b, err := json.Marshal(payLoad)
				if err != nil {
					log.Error().Err(err).Msgf("marshaling event: %v", err)
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				msg := json.RawMessage(b)
				_, _, err = gh.Client.Repositories.Dispatch(ctx, owner, name, github.DispatchRequestOptions{
					EventType:     "pull_request",
					ClientPayload: &msg,
				})
				if err != nil {
					log.Error().Err(err).Msgf("dispatching event for PR #%d", pr.GetNumber())
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				prCount++
				backoff.New(prCount).Wait()
				log.Info().Msgf("Dispatched event sent %s", string(msg))
			}

			if resp.NextPage == 0 {
				break
			}
			prOpts.Page = resp.NextPage
		}
	}
	return firstErr
}

// newAppClient creates a GitHub client authenticated as the App (JWT-based).
func newAppClient(cfg *dispatchConfig) (*gogithub.Client, error) {
	pk, err := cfg.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("getting private key: %w", err)
	}

	tr := http.DefaultTransport
	itr, err := ghinstallation.NewAppsTransport(tr, cfg.Github.App.IntegrationID, []byte(pk))
	if err != nil {
		return nil, fmt.Errorf("creating app transport: %w", err)
	}

	client, err := gogithub.NewClient(gogithub.WithHTTPClient(&http.Client{Transport: itr}))
	if err != nil {
		return nil, fmt.Errorf("creating github app client: %w", err)
	}
	return client, nil
}

// newInstallationClient creates a GitHub client authenticated for a specific installation.
func newInstallationClient(cfg *dispatchConfig, installationID int64) (*gogithub.Client, error) {
	pk, err := cfg.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("getting private key: %w", err)
	}

	tr := http.DefaultTransport
	itr, err := ghinstallation.New(tr, cfg.Github.App.IntegrationID, installationID, []byte(pk))
	if err != nil {
		return nil, fmt.Errorf("creating installation transport: %w", err)
	}

	client, err := gogithub.NewClient(gogithub.WithHTTPClient(&http.Client{Transport: itr}))
	if err != nil {
		return nil, fmt.Errorf("creating github installation client: %w", err)
	}
	return client, nil
}

// listAllInstallations lists all installations of the GitHub App.
func listAllInstallations(ctx context.Context, appClient *gogithub.Client) ([]*gogithub.Installation, error) {
	var allInstallations []*gogithub.Installation
	opts := &gogithub.ListOptions{PerPage: 100}
	for {
		installations, resp, err := appClient.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("listing app installations: %w", err)
		}
		allInstallations = append(allInstallations, installations...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allInstallations, nil
}

// listInstallationRepositories lists all repositories accessible to a specific installation.
func listInstallationRepositories(ctx context.Context, installClient *gogithub.Client) ([]string, error) {
	var allRepos []string
	opts := &gogithub.ListOptions{PerPage: 100}
	for {
		repos, resp, err := installClient.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("listing installation repositories: %w", err)
		}
		for _, repo := range repos.Repositories {
			if repo.Archived != nil && *repo.Archived {
				log.Debug().Msgf("Skipping archived repo %s", repo.GetFullName())
				continue
			}
			allRepos = append(allRepos, repo.GetFullName())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	sort.Strings(allRepos)
	return allRepos, nil
}