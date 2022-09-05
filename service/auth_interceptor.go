package service

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	jwtManager      *JWTManager
	accessibleRoles map[string][]string
}

// NewAuthInterceptor() function builds and returns a new AuthInterceptor object.
func NewAuthInterceptor(jwtManager *JWTManager, accessibleRoles map[string][]string) *AuthInterceptor {
	return &AuthInterceptor{
		jwtManager:      jwtManager,
		accessibleRoles: accessibleRoles,
	}

}

// Unary() method auths the interceptor object, which will create and return a
// gRPC unary server interceptor function.
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		log.Println("--> unary interceptor: ", info.FullMethod)

		// call interceptor.authorize() with the input context and info.FullMethod
		err1 := interceptor.authorize(ctx, info.FullMethod)

		if err1 != nil {
			return nil, err1
		}

		return handler(ctx, req)

	}
}

// Stream() method auths the interceptor object, which will create and return a
// gRPC stream server interceptor function.
func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Println("--> stream interceptor: ", info.FullMethod)

		// call interceptor.authorize() with the stream context and info.FullMethod,
		// and return right away if an error is returned.
		err := interceptor.authorize(ss.Context(), info.FullMethod)

		if err != nil {
			return err
		}

		return handler(srv, ss)
	}
}

// AccessibleRoles() function, builds a list of RPC methods and the roles that can access each of them.
/*
Note: To get the full RPC method name, run both client and server.
Then in the server logs, you will see the full method name of the CreateLaptop RPC:
*/
func AccessibleRoles() map[string][]string {
	// Note: all methods of LaptopService will starts with the same path,
	// so I define a constant for it here.
	const laptopServicePath = "/ecommerce.LaptopService/"

	// create and return a map
	return map[string][]string{
		// The first method is CreateLaptop, which only admin users can call.
		laptopServicePath + "CreateLaptop": {"admin"},
		// The UploadImage method is also accessible for admin only.
		laptopServicePath + "UploadImage": {"admin"},
		// The RateLaptop method can be called by both admin and user.
		laptopServicePath + "RateLaptop": {"admin", "user"},
		// let’s say the SearchLaptop API is accessible by everyone,
		// even for non-registered users. So the idea is: we don’t put
		// SearchLaptop or any other publicly accessible RPCs in this map.
	}
}

// Authorize() function, takes a context and method as input, and will
// return an error if the request is unauthorized.
func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) error {
	// First we get the list of roles that can access the target RPC method.
	accessibleRoles, ok := interceptor.accessibleRoles[method]

	// If it’s not in the map, then it means the RPC is publicly accessible,
	// so we simply return nil in this case.
	if !ok {
		return nil
	}

	// Else, we should get the access token from the context.
	// To do that, we use the grpc/metadata package.
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata not provided")

	}

	// Else we get the values from the authorization metadata key.
	value := md["authorization"]

	// If it’s empty, we return Unauthenticated code because the token is not provided.
	if len(value) == 0 {
		return status.Errorf(codes.Unauthenticated, "Authorization token not provided")
	}

	// Otherwise, the access token should be stored in the 1st element of the values.
	accessToken := value[0]

	// call jwtManager.Verify() to verify the token and get back the claims.
	claims, err := interceptor.jwtManager.Verify(accessToken)

	if err != nil {
		return status.Errorf(codes.Unauthenticated, "access token is invalid: %v", err)
	}

	// Else, we iterate through the accessible roles to check
	// if the user’s role can access this RPC or not.
	for _, role := range accessibleRoles {
		// If the user’s role is found in the list,
		if role == claims.Role {
			// we simply return nil.
			return nil

		}
	}

	//  If not, we return PermissionDenied status code, 
	// and a message saying user doesn’t have permission to access this RPC.
	return status.Errorf(codes.PermissionDenied, "no permission to access this RPC")

}
