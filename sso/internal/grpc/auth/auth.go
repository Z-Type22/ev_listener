package auth

import (
	"context"
	"errors"
	"strings"

	"sso/internal/services/auth"

	ssov1 "github.com/CheckEZ/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Auth interface {
	Register(ctx context.Context, email string, wallet string, password1 string, password2 string) (userID int64, err error)
	Login(ctx context.Context, email string, password string) (tokens auth.TokenResponse, err error)
	Refresh(ctx context.Context, refreshToken string) (accessToken string, err error)
	Logout(ctx context.Context, refreshToken string) (err error)
}

type ServerAPI struct {
	ssov1.UnimplementedAuthServer
	auth Auth
}

func Register(gRPC *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPC, &ServerAPI{auth: auth})
}

func (s *ServerAPI) Register(ctx context.Context, req *ssov1.RegisterRequest) (*ssov1.RegisterResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if req.GetWallet() == "" || !strings.HasPrefix(req.GetWallet(), "0x") {
		return nil, status.Error(codes.InvalidArgument, "invalid wallet or wallet is empty")
	}

	if req.GetPassword1() == "" {
		return nil, status.Error(codes.InvalidArgument, "password1 is required")
	}

	if req.GetPassword2() == "" {
		return nil, status.Error(codes.InvalidArgument, "password2 is required")
	}

	if req.GetPassword1() != req.GetPassword2() {
		return nil, status.Error(codes.InvalidArgument, "passwords don't match")
	}

	userID, err := s.auth.Register(ctx, req.GetEmail(), req.GetWallet(), req.GetPassword1(), req.GetPassword2())
	if err != nil {
		if errors.Is(err, auth.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user with email already exists")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.RegisterResponse{UserId: userID}, nil
}

func (s *ServerAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	tokens, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid email or password")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *ServerAPI) Refresh(ctx context.Context, req *ssov1.RefreshRequest) (*ssov1.RefreshResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	accessToken, err := s.auth.Refresh(ctx, req.GetRefreshToken())
	if err != nil {
		if errors.Is(err, auth.ErrRefreshTokenRevoked) {
			return nil, status.Error(codes.InvalidArgument, "token is revoked")
		}

		if errors.Is(err, auth.ErrRefreshTokenExpired) {
			return nil, status.Error(codes.InvalidArgument, "token is expired")
		}

		return nil, status.Error(codes.InvalidArgument, "invalid format token")
	}

	return &ssov1.RefreshResponse{AccessToken: accessToken}, nil
}

func (s *ServerAPI) Logout(ctx context.Context, req *ssov1.LogoutRequest) (*ssov1.LogoutResponse, error) {
	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	err := s.auth.Logout(ctx, req.GetRefreshToken())
	if err != nil {
		if errors.Is(err, auth.ErrRefreshTokenRevoked) {
			return nil, status.Error(codes.InvalidArgument, "token already revoked")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.LogoutResponse{}, nil
}
