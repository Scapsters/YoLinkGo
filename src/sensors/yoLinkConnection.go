package sensors

import (
	"com/connection"
	"com/requests"
	"com/util"
	"fmt"
)

const TOKEN_URL = "https://api.yosmart.com/open/yolink/token"
const API_URL = "https://api.yosmart.com/open/yolink/v2/api"

const TOKEN_REFRESH_BUFFER_MINUTES = 10

var _ connection.Connection = (*YoLinkConnection)(nil)

type YoLinkConnection struct {
	userId              string
	userKey             string
	accessToken         string
	refreshToken        string
	tokenExpirationTime int64
}

func NewYoLinkConnection(userId string, userKey string) (*YoLinkConnection, error) {
	c := &YoLinkConnection{
		userId:      userId,
		userKey:     userKey,
	}
	err := c.Open()
	if err != nil {
		return nil, fmt.Errorf("error while opening new YoLink connection: %w", err)
	}
	status, description := c.Status()
	if status != connection.Good {
		return nil, fmt.Errorf("error while checking status of new YoLink connection. Connection status: %v, connection description: %v", status, description)
	}
	return c, nil
}

// Ensure the conncetion to YoLink is active, with 3 main paths of execution:
// Token is active and far from expiring: no actions taken
// Token is active but close to expiring: token is refreshed using current token
// No token exists or token is expired: fetch new token
func (c *YoLinkConnection) Open() error {
	currentTime := utils.Time()

	var hasToken = c.tokenExpirationTime != 0
	var isTokenNearlyExpired = hasToken && currentTime > c.tokenExpirationTime - TOKEN_REFRESH_BUFFER_MINUTES * 60
	var isTokenExpired = hasToken && currentTime > c.tokenExpirationTime 
	var response *AuthenticationResponse
	var err error

	if hasToken && !isTokenNearlyExpired {
		return nil
	}

	if !hasToken || isTokenExpired {
		response, err = requests.PostForm[AuthenticationResponse](
			TOKEN_URL,
			map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     c.userId,
				"client_secret": c.userKey,
			},
		)
		if err != nil {
			return fmt.Errorf("error generating new access token %v: %w", c.refreshToken, err)
		}
	}
	if isTokenNearlyExpired {
		
	}
	c.accessToken = response.Access_token
	c.refreshToken = response.Refresh_token
	c.tokenExpirationTime = utils.TimeSeconds() + int64(response.Expires_in)
	return nil
}
func (c *YoLinkConnection) Close() error {
	// No need to close connection
	return nil
}
func (c *YoLinkConnection) Status() (connection.PingResult, string) {
	err := c.refreshCurrentToken()
	if err != nil {
		return connection.Bad, err.Error()
	}
	return connection.Good, "Successful ping via token refresh"
}
func (c *YoLinkConnection) makeRequest() error {
	// TODO: implement
	return nil
}
// Refresh the current token. Requires an existing token to exist.
func (c *YoLinkConnection) refreshCurrentToken() error {
	response, err := requests.PostForm[AuthenticationResponse](
		TOKEN_URL,
		map[string]string{
			"grant_type":    "refresh_token",
			"client_id":     c.userId,
			"refresh_token": c.refreshToken,
		},
	)
	if err != nil {
		return fmt.Errorf("error refreshing token with refresh token %v: %w", c.refreshToken, err)
	}
	c.accessToken = response.Access_token
	c.refreshToken = response.Refresh_token
	c.tokenExpirationTime = utils.TimeSeconds() + int64(response.Expires_in)
	return nil
}

type AuthenticationResponse struct {
	Access_token string
	Refresh_token string
	Expires_in int
}