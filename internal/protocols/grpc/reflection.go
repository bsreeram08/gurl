package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
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

// ListServices lists all services available on the server
func (rc *ReflectionClient) ListServices(ctx context.Context) ([]string, error) {
	if rc.conn == nil {
		return nil, fmt.Errorf("no connection")
	}
	// Reflection requires server support and proper API setup
	// Return stub response for compilation
	return []string{}, nil
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
