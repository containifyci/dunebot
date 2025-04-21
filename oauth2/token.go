package oauth2

import (
	"context"
	"errors"

	"github.com/containifyci/oauth2-storage/pkg/proto"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OAuth2Config = oauth2.Config
type Endpoint = oauth2.Endpoint

type Config struct {
	Ctx            context.Context
	InstallationId string
	User           string
	Addr           string
	*OAuth2Config
	AuthInterceptor
}

/*
GRPC Authentication Interceptor

Inspired by the following tutorial https://dev.to/techschoolguru/use-grpc-interceptor-for-authorization-with-jwt-1c5h
*/
type AuthInterceptor struct {
	accessTokenFnc func() string
}

func (interceptor *AuthInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		log.Debug().Msgf("--> unary interceptor: %s", method)

		return invoker(interceptor.attachToken(ctx), method, req, reply, cc, opts...)
	}
}

func (interceptor *AuthInterceptor) Stream() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		log.Debug().Msgf("--> stream interceptor: %s", method)

		return streamer(interceptor.attachToken(ctx), desc, cc, method, opts...)
	}
}

func (interceptor *AuthInterceptor) attachToken(ctx context.Context) context.Context {
	accessToken := interceptor.accessTokenFnc()
	if accessToken != "" {
		return metadata.AppendToOutgoingContext(ctx, "authorization", accessToken)
	}
	return ctx
}

func NewAuthInterceptor(accessTokenFnc func() string) *AuthInterceptor {
	return &AuthInterceptor{accessTokenFnc: accessTokenFnc}
}

// END GRPC Authentication Interceptor

func NewClient(auth AuthInterceptor, addr string) (proto.TokenClient, func() error, error) {
	// Initialize a gRPC connection
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(auth.Unary()),
		grpc.WithStreamInterceptor(auth.Stream()))
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to gRPC server: %v\n", err)
		return nil, func() error { return nil }, err
	}

	// Initialize a gRPC client
	return proto.NewTokenClient(conn), func() error { return conn.Close() }, nil
}

func (c *Config) StoreToken(token *oauth2.Token) error {
	grpcClient, close, err := NewClient(c.AuthInterceptor, c.Addr)
	defer func () {
		err := close()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to close gRPC connection: %v\n", err)
		}
	}()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to gRPC server: %v\n", err)
		return err
	}
	tk := &proto.CustomToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       timestamppb.New(token.Expiry.UTC()),
		User:         c.User,
	}

	log.Debug().Msgf("Store Token for user: %s+\n", tk.User)
	log.Debug().Msgf("Token Expiry: %s\n", tk.Expiry.AsTime())

	_, err = grpcClient.StoreToken(c.Ctx, &proto.SingleToken{
		InstallationId: c.InstallationId,
		Token:          tk,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to store token %s\n", err)
		return err
	}
	return nil
}

func (c *Config) RetrieveToken() (*oauth2.Token, error) {
	grpcClient, close, err := NewClient(c.AuthInterceptor, c.Addr)

	defer func () {
		err := close()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to close gRPC connection: %v\n", err)
		}
	}()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to gRPC server: %v\n", err)
		return nil, err
	}

	tk, err := grpcClient.RetrieveToken(c.Ctx, &proto.SingleToken{
		InstallationId: c.InstallationId,
		Token: &proto.CustomToken{
			User: c.User,
		},
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to retrieve token from gRPC server: %v\n", err)
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  tk.Token.AccessToken,
		RefreshToken: tk.Token.RefreshToken,
		TokenType:    tk.Token.TokenType,
		Expiry:       tk.Token.Expiry.AsTime(),
	}, nil
}

func (c *Config) RevokeToken() error {
	grpcClient, close, err := NewClient(c.AuthInterceptor, c.Addr)
	defer func () {
		err := close()
		if err != nil {
			log.Error().Err(err).Msgf("Failed to close gRPC connection: %v\n", err)
		}
	}()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to gRPC server: %v\n", err)
		return err
	}

	message, err := grpcClient.RevokeToken(c.Ctx, &proto.SingleToken{
		InstallationId: c.InstallationId,
		Token: &proto.CustomToken{
			User: c.User,
		},
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to revoke token from gRPC server: %v\n", err)
		return err
	}

	if message.Revoked {
		log.Debug().Msgf("Successfully revoked token for %s\n", c.User)
		return nil
	}

	return errors.New(message.Error.Message)
}

func (c *Config) TokenSourceFrom(ctx context.Context) oauth2.TokenSource {
	t, err := c.RetrieveToken()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to retrieve token from gRPC server: %v\n", err)
		return nil
	}
	log.Debug().Msgf("Retrieved token: %v+\n", t)
	return c.TokenSource(ctx, t)
}

func (c *Config) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	rts := &DuneBotTokenSource{
		t:      t,
		source: c.OAuth2Config.TokenSource(ctx, t),
		config: c,
	}
	return oauth2.ReuseTokenSource(t, rts)
}

type DuneBotTokenSource struct {
	t      *oauth2.Token
	source oauth2.TokenSource
	config *Config
}

func (t *DuneBotTokenSource) Token() (*oauth2.Token, error) {
	token, err := t.source.Token()
	if err != nil {
		return nil, err
	}
	if token.RefreshToken != t.t.RefreshToken {
		if err := t.config.StoreToken(token); err != nil {
			return nil, err
		}
	}
	return token, nil
}
