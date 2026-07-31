package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"rest/internal/clients/sso/responses"
	"time"

	ssov1 "github.com/CheckEZ/protos/gen/go/sso"
	grpclog "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	api_auth   ssov1.AuthClient
	api_health ssov1.HealthClient
	log        *slog.Logger
	conn       *grpc.ClientConn
}

func New(
	ctx context.Context, log *slog.Logger, addr string, timeout time.Duration, retriesCount int,
) *Client {
	const op = "grpc.New"

	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.NotFound, codes.Aborted, codes.DeadlineExceeded),
		grpcretry.WithMax(uint(retriesCount)),
		grpcretry.WithPerRetryTimeout(timeout),
	}

	logOpts := []grpclog.Option{
		grpclog.WithLogOnEvents(grpclog.PayloadReceived, grpclog.PayloadSent),
	}

	cc, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			grpclog.UnaryClientInterceptor(InteceptorLogger(log), logOpts...),
			grpcretry.UnaryClientInterceptor(retryOpts...),
		),
	)
	if err != nil {
		panic(fmt.Errorf("%s: %w", op, err))
	}

	return &Client{
		api_auth:   ssov1.NewAuthClient(cc),
		api_health: ssov1.NewHealthClient(cc),
		log:        log,
		conn:       cc,
	}
}

func InteceptorLogger(l *slog.Logger) grpclog.Logger {
	return grpclog.LoggerFunc(func(ctx context.Context, lvl grpclog.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func (c *Client) Close() {
	if err := c.conn.Close(); err != nil {
		c.log.Error("failed to close sso connection", slog.Any("error", err))

		return
	}

	c.log.Info("sso connection stoped")
}

func (c *Client) Register(
	ctx context.Context, email string, wallet string, password1 string, password2 string,
) (int64, error) {
	const op = "grpc.auth.Register"

	log := c.log.With(slog.String("op", op))

	log.Info("attempting register")

	resp, err := c.api_auth.Register(ctx, &ssov1.RegisterRequest{
		Email:     email,
		Wallet:    wallet,
		Password1: password1,
		Password2: password2,
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("register successfully")

	return resp.GetUserId(), nil
}

func (c *Client) Login(
	ctx context.Context, email string, password string,
) (responses.TokenResponse, error) {
	const op = "grpc.auth.Login"

	log := c.log.With(slog.String("op", op))

	log.Info("attempting login")

	resp, err := c.api_auth.Login(ctx, &ssov1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return responses.TokenResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("login successfully")

	return responses.TokenResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (string, error) {
	const op = "grpc.auth.Refresh"

	log := c.log.With(slog.String("op", op))

	log.Info("attempting refresh token")

	resp, err := c.api_auth.Refresh(ctx, &ssov1.RefreshRequest{RefreshToken: refreshToken})
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("access token obtained successfully")

	return resp.AccessToken, nil
}

func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	const op = "grpc.auth.Logout"

	log := c.log.With(slog.String("op", op))

	log.Info("attempting refresh token")

	_, err := c.api_auth.Logout(ctx, &ssov1.LogoutRequest{RefreshToken: refreshToken})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("logout successfully")

	return nil
}

func (c *Client) Check(ctx context.Context) error {
	const op = "grpc.health.Check"

	log := c.log.With(slog.String("op", op))

	log.Info("attempting refresh token")

	_, err := c.api_health.Check(ctx, &ssov1.HealthCheckRequest{})
	if err != nil {
		return err
	}

	log.Info("check completed successfully")

	return nil
}
