package commands

import (
	"fmt"
	"strings"

	authpkg "github.com/sreeram/gurl/internal/auth"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

func authConfigFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "auth",
			Usage: "Authentication type (basic|bearer|apikey|oauth1|oauth2|awsv4|digest|ntlm|none)",
		},
		&cli.StringSliceFlag{
			Name:  "auth-param",
			Usage: "Authentication parameter (format: KEY=VALUE, can repeat)",
		},
	}
}

func parseAuthParamFlags(values []string) (map[string]string, error) {
	params := make(map[string]string, len(values))
	for _, value := range values {
		key, paramValue, err := parseAuthParamFlag(value)
		if err != nil {
			return nil, err
		}
		params[key] = paramValue
	}
	return params, nil
}

func parseAuthParamFlag(value string) (string, string, error) {
	key, paramValue, ok := strings.Cut(value, "=")
	if !ok {
		return "", "", fmt.Errorf("auth-param must be KEY=VALUE")
	}

	key = strings.TrimSpace(key)
	paramValue = strings.TrimSpace(paramValue)
	if key == "" || paramValue == "" {
		return "", "", fmt.Errorf("auth-param must be KEY=VALUE")
	}
	return key, paramValue, nil
}

func buildAuthConfig(authType string, params map[string]string) (*types.AuthConfig, bool, error) {
	authType = strings.ToLower(strings.TrimSpace(authType))
	if authType == "" {
		if len(params) > 0 {
			return nil, false, fmt.Errorf("--auth is required when using --auth-param")
		}
		return nil, false, nil
	}

	if authType == "none" {
		if len(params) > 0 {
			return nil, false, fmt.Errorf("--auth-param cannot be used with --auth none")
		}
		return nil, true, nil
	}

	if authpkg.BuiltinRegistry().Get(authType) == nil {
		return nil, false, fmt.Errorf("unknown auth type %q", authType)
	}

	return &types.AuthConfig{
		Type:   authType,
		Params: params,
	}, true, nil
}

func parseAuthConfigFlags(c *cli.Command) (*types.AuthConfig, bool, error) {
	params, err := parseAuthParamFlags(c.StringSlice("auth-param"))
	if err != nil {
		return nil, false, err
	}
	return buildAuthConfig(c.String("auth"), params)
}
