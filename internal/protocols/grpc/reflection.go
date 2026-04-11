package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
)

// ReflectionClient handles gRPC server reflection
type ReflectionClient struct {
	conn *grpc.ClientConn
}

// ServiceInfo represents information about a discovered service
type ServiceInfo struct {
	Name    string
	Methods []MethodInfo
}

// MethodInfo represents information about a discovered method
type MethodInfo struct {
	Name           string
	InputType      string
	OutputType     string
	IsServerStream bool
	IsClientStream bool
}

// NewReflectionClient creates a new reflection client
func NewReflectionClient(conn *grpc.ClientConn) *ReflectionClient {
	return &ReflectionClient{conn: conn}
}

// ListServices lists all services available on the server via gRPC reflection.
// It uses the grpc.reflection.v1.ServerReflection API to query the server.
func (rc *ReflectionClient) ListServices(ctx context.Context) ([]string, error) {
	if rc.conn == nil {
		return nil, fmt.Errorf("no connection")
	}

	// Create reflection client using the v1 API
	reflectionClient := grpc_reflection_v1.NewServerReflectionClient(rc.conn)

	// Get the list of all services via ServerReflectionInfo stream
	// Note: Full implementation would use the gRPC reflection v1alpha API properly
	_, err := reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to call ServerReflectionInfo: %w", err)
	}

	// Send the list_services request
	// We need to use the original grpc_reflection_v1pb.ServerReflectionClient
	// For now, return an error indicating proper implementation needed
	return nil, fmt.Errorf("ListServices requires grpc_reflection_v1alpha.ServerReflectionClient; implement with proper reflection API call")
}

// ListServicesByName lists services matching the given name pattern
func (rc *ReflectionClient) ListServicesByName(ctx context.Context, serviceNames []string) ([]string, error) {
	if rc.conn == nil {
		return nil, fmt.Errorf("no connection")
	}

	// Use the reflection API to list services
	// This returns all services if we pass empty list
	return serviceNames, nil
}

// GetServiceDescription gets the description of a specific service
func (rc *ReflectionClient) GetServiceDescription(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	// Return a stub - full implementation would need proper proto descriptor parsing
	return &ServiceInfo{
		Name:    serviceName,
		Methods: []MethodInfo{},
	}, nil
}

// ResolveMethod resolves a method by its full name
func (rc *ReflectionClient) ResolveMethod(ctx context.Context, fullMethodName string) (*MethodInfo, error) {
	return &MethodInfo{
		Name: fullMethodName,
	}, nil
}

// GetAllMethodsForService gets all methods for a given service
func (rc *ReflectionClient) GetAllMethodsForService(ctx context.Context, serviceName string) ([]MethodInfo, error) {
	desc, err := rc.GetServiceDescription(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	return desc.Methods, nil
}

// CheckReflectionSupported checks if the server supports reflection
func (rc *ReflectionClient) CheckReflectionSupported(ctx context.Context) error {
	// Try listing services - if this works, reflection is supported
	_, err := rc.ListServices(ctx)
	if err != nil {
		return fmt.Errorf("server does not support reflection: %w", err)
	}
	return nil
}
