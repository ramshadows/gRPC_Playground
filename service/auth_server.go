package service

import (
	"context"
	pb "gRPC-Playground/ecommerce"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	userStore  UserStore
	jwtManager *JWTManager
}

// NewAuthServer builds and returns a new AuthServer object
func NewAuthServer(userStore UserStore, jwtManager *JWTManager) *AuthServer {
	return &AuthServer{
		pb.UnimplementedAuthServiceServer{},
		userStore,
		jwtManager,
	}
}

func (server *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// First we call userStore.Find() to find the user by username
	// using the req.GetUsername().
	user, err := server.userStore.Find(req.GetUsername())

	// If there’s an error, just return it with an Internal error code.
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot find user: %v", err)
	}

	// Else, if the user is not found or if the password is incorrect
	// then we return status code NotFound with a message saying
	// the username or password is incorrect.
	if user == nil || !user.IsCorrectPassword(req.GetPassword()) {
		return nil, status.Errorf(codes.NotFound, "incorrect username/password")
	}

	// If the user is found and the password is correct, we call jwtManager.Generate()
	// to generate a new access token.
	token, err := server.jwtManager.Generate(user)

	// If an error occurs, we return it with Internal status code.
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate access token")
	}

	// Otherwise, we create a new login response object with the generated access token, and return it to the client.
	res := &pb.LoginResponse{
		AccessToken: token,
	}
	return res, nil

}
