package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/palantir/go-githubapp/githubapp"
	"golang.org/x/oauth2"

	"github.com/containifyci/dunebot/cmd"
	appoauth "github.com/containifyci/dunebot/oauth2"
	"github.com/containifyci/dunebot/pkg/auth"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/queue"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"

	//SessionStore

	"github.com/alexedwards/scs"
	poauth2 "github.com/palantir/go-githubapp/oauth2"
)

type appCmdArgs struct {
	configFile *string
}

var appArgs = &appCmdArgs{}

type AppCommand struct {
	*config.Config
	githubapp.ClientCreator
}

// appCmd represents the app command
var appCmd = &cobra.Command{
	Use:   "app",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: execute,
}

func init() {
	cmd.RootCmd.AddCommand(appCmd)

	appArgs.configFile = appCmd.Flags().String("config", "", "The config file to use. If not set, the config will be loaded from the environment variables.")
}

func execute(cmd *cobra.Command, args []string) {
	logger := zerolog.New(os.Stdout).With().Caller().Stack().Timestamp().Logger()
	log.Logger = logger
	zerolog.DefaultContextLogger = &logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg, err := config.GetConfig(*appArgs.configFile)
	if err != nil {
		panic(err)
	}

	if cfg.AppConfig.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	mux := setupHttpServeMux(cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	logger.Info().Msgf("Starting server on %s...", addr)

	err = http.ListenAndServe(addr, mux)
	if err != nil {
		panic(err)
	}
}

func setupHttpServeMux(cfg *config.Config) *http.ServeMux {
	sessions := scs.NewCookieManager("session-key")
	sessions.Name("dunebot")
	sessions.Lifetime(24 * time.Hour)
	sessions.Persist(true)
	sessions.HttpOnly(true)
	sessions.Secure(true)

	metricsRegistry := metrics.DefaultRegistry

	cc, err := githubapp.NewDefaultCachingClientCreator(
		//TODO maybe find an better way then to copy the config
		cfg.Github.ToGithubAppConfig(),
		githubapp.WithClientUserAgent(cfg.AppConfig.UserAgent),
		githubapp.WithClientTimeout(cfg.AppConfig.ClientTimeOutDuration()),
		githubapp.WithClientCaching(false, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
		githubapp.WithClientMiddleware(
			githubapp.ClientMetrics(metricsRegistry),
		),
	)
	if err != nil {
		panic(err)
	}

	prQueue := queue.NewRepositoryQueue()

	authSrv := auth.NewSigningService(cfg.JWT.PrivateKey)
	if authSrv.IsEnabled() {
		log.Info().Msgf("JWT token authentication enabled: %s\n", authSrv.CreateToken(auth.ServiceClaims{ServiceName: "dunebot"})[0:12])
	} else {
		log.Info().Msgf("JWT token authentication disabled\n")
	}

	prHandler := &cmd.PRHandler{
		ClientCreator:   cc,
		RepositoryQueue: prQueue,
		Config:          cfg,
		AuthSrv:         authSrv,
	}

	appHandler := &cmd.AppHandler{
		ClientCreator: cc,
		Config:        cfg,
	}

	repositoryDispatchHandler := &cmd.RepositoryDispatchHandler{
		Handler: prHandler,
	}

	webhookHandler := githubapp.NewEventDispatcher(
		[]githubapp.EventHandler{
			prHandler, appHandler,
			repositoryDispatchHandler,
		},

		cfg.Github.App.WebhookSecret,
		// githubapp.WithScheduler(
		// 	githubapp.QueueAsyncScheduler(1024, 1, githubapp.WithSchedulingMetrics(metricsRegistry)),
		// ),
	)

	mux := http.NewServeMux()

	mux.Handle(githubapp.DefaultWebhookRoute, webhookHandler)

	mux.Handle("/api/metric/queue", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		entries := prQueue.Entries()
		log.Debug().Msgf("Entries %v+\n", entries)
		err := json.NewEncoder(w).Encode(entries)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to write queue response: %v\n", err)
		}
	}))

	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Error().Err(err).Msgf("Failed to write health response: %v\n", err)
		}
	}))

	handlerOpts := HandlerOptions{
		cfg:           cfg,
		clientOpts:    []github.Option{},
		ClientCreator: cc,
	}

	//return all repositories that DuneBot is enabled for
	mux.Handle("/api/installation/repositories", http.HandlerFunc(DuneBotRepositoriesHandler(handlerOpts)))

	mux.Handle(poauth2.DefaultRoute, poauth2.NewHandler(
		appoauth.GetConfig(cfg),
		poauth2.ForceTLS(true),
		poauth2.WithStore(&poauth2.SessionStateStore{
			Sessions: sessions,
		}),
		poauth2.OnLogin(Login(cfg.Github, sessions)),
	))

	gh, _ := cc.NewAppClient()
	// if err != nil {
	// 	panic(err)
	// }

	dashboardHandler := DashboardHandler(cfg, gh, &github.GithubClient{}, NewWebHandler(sessions))

	mux.Handle("/api/github/dashboard", RequireLogin(sessions, cfg.Server.BasePath)(dashboardHandler))

	registerSetup(cfg, gh, &github.GithubClient{}, mux)
	return mux
}

func oauth2DeviceFlow(cfg config.ConfigTransformer, errorHandler errorReporter) (*oauth2.DeviceAuthResponse, func(github.GithubClientCreator, *github.Installation), error) {
	config := cfg.ToOAuth2Config()

	ctx := context.Background()

	log.Debug().Msgf("Config: %+v\n", config)

	deviceCode, err := config.DeviceAuth(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("error getting device code: %v\n", err)
		return nil, nil, err
	}

	log.Debug().Msgf("Go to %v and enter code %v\n", deviceCode.VerificationURI, deviceCode.UserCode)

	return deviceCode, func(ghc github.GithubClientCreator, installation *github.Installation) {
		token, err := config.DeviceAccessToken(ctx, deviceCode)
		if err != nil {
			errorHandler.Errorf(err, "Error exchanging Device Code for for access token: %v\n", err)
			return
		}

		cli := ghc.NewClient(github.WithConfig(github.NewStaticTokenConfig(token.AccessToken)))

		user, _, err := cli.Client.Users.Get(ctx, "")
		if err != nil {
			errorHandler.Errorf(err, "Error getting user from token source: %v\n", err)
			return
		}
		log.Debug().Msgf("User: %v\n", user)

		authSrv := auth.NewSigningService(cfg.Config().JWT.PrivateKey)
		accesToken := authSrv.CreateTokenFnc(auth.ServiceClaims{ServiceName: "dunebot"})

		oauth2cfg := appoauth.Config{
			InstallationId:  fmt.Sprintf("%d", installation.GetID()),
			User:            user.GetLogin(),
			Ctx:             ctx,
			OAuth2Config:    config,
			AuthInterceptor: *appoauth.NewAuthInterceptor(accesToken),
			Addr:            cfg.Config().JWT.Address,
		}

		err = oauth2cfg.StoreToken(token)
		if err != nil {
			errorHandler.Errorf(err, "Error storing token: %v\n", err)
			return
		}

		// cli = ghc.NewClient(github.WithTokenSource(oauth2cfg.TokenSource(ctx, token)), github.WithContext(ctx), github.WithConfig(github.NewConfig()))
		installs, _, err := cli.Client.Apps.ListUserInstallations(ctx, nil)
		if err != nil {
			// log.Error().Err(err).Msgf("Failed to list installations for %s", oauth2cfg.User)
			errorHandler.Errorf(err, "Failed to list installations for %s", oauth2cfg.User)
			return
		}

		for _, install := range installs {
			log.Debug().Msgf("Retrieved following installations %v+\n", install)
		}

		log.Debug().Msgf("Token Source User: %v\n", user)
	}, nil
}

// Page represents the data to be rendered in the HTML template
type Page struct {
	Title            string
	Message          string
	VerificationLink string
	Code             string
}

func FindInstallation(ctx context.Context, gh *github.Client, user string) (*github.Installation, error) {
	installion, _, err := gh.Apps.FindOrganizationInstallation(ctx, "containifyci")
	if err != nil {
		log.Info().Err(err).Msgf("Failed to retrieve organisation installation try with user installation instead: %v\n", err)
		installion, _, err = gh.Apps.FindUserInstallation(ctx, user)
	}
	if err != nil {
		log.Error().Err(err).Msgf("Failed to retrieve user installation: %v\n", err)
		return nil, err
	}
	return installion, nil
}
