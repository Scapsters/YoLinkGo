package sensors

import (
	"com/src/connection"
	"com/src/requests"
	"com/src/util"
	"fmt"
)

const TOKEN_URL = "https://api.yosmart.com/open/yolink/token"
const API_URL = "https://api.yosmart.com/open/yolink/v2/api"

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

func (c YoLinkConnection) Open() error {
	currentTime := utils.Time()

	var tokenExpired = c.tokenExpirationTime != 0 && currentTime > c.tokenExpirationTime 
	var response map[string]any
	var err error
	if tokenExpired {
		response, err = requests.Post(
			API_URL,
			map[string]string{
				"grant_type":    "refresh_token",
				"client_id":     c.userId,
				"refresh_token": c.refreshToken,
			},
		)
		if err != nil {
			return fmt.Errorf("error refreshing token with refresh token %v: %w", c.refreshToken, err)
		}
	} else {
		response, err = requests.Post(
			API_URL,
			map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     c.userId,
				"client_secret": c.userKey,
			},
		)
		if err != nil {
			return fmt.Errorf("error refreshing token with refresh token %v: %w", c.refreshToken, err)
		}
	}

	for k, v := range response {
		fmt.Printf("%v: %v\n", k, v)
	}

	return nil
}
func (c YoLinkConnection) Close() error {
	// No need to close connection
	return nil
}
func (c YoLinkConnection) Status() (connection.PingResult, string) {
	// TODO: implement
	return connection.Good, ""
}

func (c YoLinkConnection) createTokens() error {
	// TODO: implement
	return nil
}
func (c YoLinkConnection) makeRequest() error {
	// TODO: implement
	return nil
}
