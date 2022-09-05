package client

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

/*
Note: Intercept all gRPC requests and attach an access token to them
(if necessary) before invoking the server.
*/

// AuthInterceptor struct.
// Fields: auth client object that will be used to login user,
// a map to tell us which method needs authentication, and
// the latest acquired access token.
type AuthInterceptor struct {
	authClient  *AuthClient
	authMethods map[string]bool
	accessToken string
}

func NewAuthInterceptor(
	authClient *AuthClient,
	authMethods map[string]bool,
	refreshDuration time.Duration, // tell us how often we should call the login API to get a new token.
) (*AuthInterceptor, error) {
	authInterceptor := &AuthInterceptor{
		authClient:  authClient,
		authMethods: authMethods,
	}

	// scheduleRefreshToken() to schedule refreshing access token and
	// pass in the refresh duration.
	err := authInterceptor.scheduleRefreshToken(refreshDuration)

	if err != nil {
		return nil, err
	}

	return authInterceptor, nil

}

// refreshToken refreshes  access token with no scheduling.
func (interceptor *AuthInterceptor) refreshToken() error {
	// use the auth client to login user.
	accessToken, err := interceptor.authClient.Login()
	if err != nil {
		return err
	}

	// Once the token is returned, we simply store it in the interceptor.accessToken field.
	interceptor.accessToken = accessToken
	log.Printf("token refreshed: %v", accessToken)

	return nil
}

// scheduleRefreshToken function refreshes access token on schedule
// Note: we launch a separate go routine to periodically call login API
// to get a new access token before the current token expired.
func (interceptor *AuthInterceptor) scheduleRefreshToken(refreshDuration time.Duration) error {
	// call refreshToken() successfully for the first time so that
	// a valid access token is always available to be used.
	err := interceptor.refreshToken()
	// check for errors
	if err != nil {
		return err
	}

	// Then, we launch a new go routine.
	go func() {
		// use a wait variable to store how much time we need to wait before refreshing the token.
		wait := refreshDuration

		// Then we launch an infinite loop,
		for {
			// call time.Sleep() to wait.
			time.Sleep(wait)

			// after that amount of waiting time, call interceptor.refreshToken().
			err := interceptor.refreshToken()
			if err != nil {
				// If an error occurs, we should only wait a short period of time,
				// before retrying it.
				wait = time.Second
			} else {
				// If there’s no error, then we definitely should wait for refreshDuration.
				wait = refreshDuration
			}
		}
	}()

	return nil
}

// Unary() function adds Unary interceptors to attach the token to the request context.
// returns a gRPC unary client interceptor.
func (interceptor *AuthInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// logs the calling method name.
		log.Printf("--> unary interceptor: %s", method)

		// check if this method needs authentication or not.
		// If the method require authentication,
		if interceptor.authMethods[method] {
			// attach the access token to the context before invoking the actual RPC.
			return invoker(interceptor.attachToken(ctx), method, req, reply, cc, opts...)
		}

		// If the method doesn’t require authentication, then nothing to be done,
		// we simply invoke the RPC with the original context.
		return invoker(ctx, method, req, reply, cc, opts...)

	}
}

// Stream() function adds stream interceptors to attach the token to the request context.
// returns a gRPC stream client interceptor.
func (interceptor *AuthInterceptor) Stream() grpc.StreamClientInterceptor {
    return func(
        ctx context.Context,
        desc *grpc.StreamDesc,
        cc *grpc.ClientConn,
        method string,
        ss grpc.Streamer, // ss -> server stream
        opts ...grpc.CallOption,
    ) (grpc.ClientStream, error) {
        log.Printf("--> stream interceptor: %s", method)

        if interceptor.authMethods[method] {
            return ss(interceptor.attachToken(ctx), desc, cc, method, opts...)
        }

        return ss(ctx, desc, cc, method, opts...)
    }
}


// attachToken() function attaches a token to the input context and returns the result.
func (interceptor *AuthInterceptor) attachToken(ctx context.Context) context.Context {
	// Note: authorization key string must match with the one used on the server side.
	return metadata.AppendToOutgoingContext(ctx, "authorization", interceptor.accessToken)
}
