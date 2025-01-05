package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/rs/zerolog/log"
)

type HandlerOptions struct {
	cfg        *config.Config
	clientOpts []github.Option
	githubapp.ClientCreator
}

func DuneBotRepositoriesHandler(opts HandlerOptions) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			log.Warn().Msgf("Unauthorized request: %v\n", r)
			http.Error(w, "Missing Authorization header bearer token", http.StatusUnauthorized)
			return
		}
		token := strings.Split(auth, " ")[1]

		options := append([]github.Option{github.WithConfig(github.NewStaticTokenConfig(token))}, opts.clientOpts...)
		ghu := github.NewClient(options...)

		user, _, err := ghu.Client.Users.Get(r.Context(), "")
		if err != nil {
			log.Error().Err(err).Msgf("Error getting user from token %v\n", err)
			http.Error(w, "Error getting user from token", http.StatusUnauthorized)
			return
		}

		log.Debug().Msgf("User: %s\n", *user.Login)

		orgs, _, err := ghu.Client.Organizations.List(r.Context(), "", nil)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to retrieve user orgs: %v\n", err)
			http.Error(w, "Failed to retrieve user orgs", http.StatusUnauthorized)
			return
		}

		orgMember := slices.ContainsFunc(orgs, func(org *github.Organization) bool {
			log.Debug().Msgf("Org: %+v\n", org)
			return *org.Login == "containifyci"
		})

		if !orgMember {
			log.Error().Err(err).Msgf("User is not a member of the organisation\n")
			http.Error(w, "User is not a member of the organisation", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		gh, err := opts.NewInstallationClient(opts.cfg.AppConfig.InstallationId)

		if err != nil {
			log.Error().Err(err).Msgf("Error creating installation client: %v\n", err)
			http.Error(w, "User is not a member of the organisation", http.StatusUnauthorized)
			return
		}

		ctx := context.Background()
		// Fetch all repositories that the app is enabled for
		var allRepos []string
		opts := &github.ListOptions{
			PerPage: 100,
		}
		for {
			repos, resp, err := gh.Apps.ListRepos(ctx, opts)
			if err != nil {
				log.Error().Err(err).Msgf("Error fetching repositories: %v", err)
				http.Error(w, "Error fetching repositories", http.StatusInternalServerError)
				return
			}

			for _, repo := range repos.Repositories {
				if repo.Archived == nil {
					repo.Archived = new(bool)
					*repo.Archived = false
				}
				if *repo.Archived {
					log.Debug().Msgf("Skipping archived repo %s\n", repo.GetFullName())
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

		err = json.NewEncoder(w).Encode(allRepos)
		if err != nil {
			log.Error().Err(err).Msgf("json response encoding failed: %v\n", err)
			http.Error(w, "json response encoding failed", http.StatusInternalServerError)
		}
	}
}
