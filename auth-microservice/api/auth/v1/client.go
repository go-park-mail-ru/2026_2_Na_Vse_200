package authv1

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial creates a connection to Auth on the application's internal network.
func Dial(target string, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	direct := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Auth's internal address must resolve inside the application network.
		grpc.WithNoProxy(),
		// Compose provides address records for Auth. Service settings are local.
		grpc.WithDisableServiceConfig(),
	}
	return grpc.NewClient(target, append(direct, options...)...)
}
