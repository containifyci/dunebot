package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/containifyci/dunebot/cmd"
	"github.com/containifyci/dunebot/pkg/backoff"
	"github.com/containifyci/dunebot/pkg/github"
)

type dispatchCmdArgs struct {
	repo   string
	dryRun bool
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

	if cfg.GithubToken == "" {
		log.Error().Msg("GITHUB_TOKEN is required")
		return fmt.Errorf("GITHUB_TOKEN is required")
	}

	gh := github.NewClient(github.WithConfig(github.NewStaticTokenConfig(cfg.GithubToken)))
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
	defer resp.Body.Close()

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

func run(cfg *dispatchConfig, gh github.GithubClient) error {
	// Create a new request

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
		repositories, err := listDuneBotRepositories(ctx, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("fetching repositories: %v", err)
			return err
		}
		repos = repositories
	}

	prCount := 0
	// Iterate over all repositories and fetch open pull requests
	for _, repo := range repos {
		log.Debug().Msgf("Repository: %s\n", repo)
		sp := strings.Split(repo, "/")
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
				log.Error().Err(err).Msgf("fetching pull requests for repository %s: %v", repo, err)
				return err
			}

			for _, pr := range pullRequests {
				log.Debug().Msgf("\tOpen PR: #%d %s\n", pr.GetNumber(), pr.GetTitle())
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
					return err
				}
				msg := json.RawMessage(b)
				_, _, err = gh.Client.Repositories.Dispatch(ctx, owner, name, github.DispatchRequestOptions{
					EventType:     "pull_request",
					ClientPayload: &msg,
				})
				if err != nil {
					log.Error().Err(err).Msgf("dispatching event for PR #%d: %v", pr.GetNumber(), err)
					return err
				}
				prCount++
				backoff.New(prCount).Wait()
				log.Info().Msgf("Dispatched event sent %s\n", string(msg))
			}

			if resp.NextPage == 0 {
				break
			}
			prOpts.Page = resp.NextPage
		}
	}
	return nil
}
