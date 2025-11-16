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
	c.accessToken = response.AccessToken
	c.refreshToken = response.RefreshToken
	c.tokenExpirationTime = utils.TimeSeconds() + int64(response.ExpiresIn)
	return nil
}
func (c *YoLinkConnection) Close() error {
	c.accessToken = ""
	c.refreshToken = ""
	c.tokenExpirationTime = 0
	return nil
}
func (c *YoLinkConnection) Status() (connection.PingResult, string) {
	err := c.refreshCurrentToken()
	if err != nil {
		return connection.Bad, err.Error()
	}
	return connection.Good, "Successful ping via token refresh"
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
	c.accessToken = response.AccessToken
	c.refreshToken = response.RefreshToken
	c.tokenExpirationTime = utils.TimeSeconds() + int64(response.ExpiresIn)
	return nil
}

func MakeYoLinkRequest[T any](c *YoLinkConnection, simpleBDDP SimpleBDDP) (*T, error) {
	BDDPMap, err := utils.ToMap(simpleBDDP)
	if err != nil {
		return nil, fmt.Errorf("error converting body %v to map: %w", simpleBDDP, err)
	}
	BDDPMap["time"] = fmt.Sprint(utils.TimeSeconds())

	c.Open() // Ensure tokens are up to date
	headers := map[string]string{
		"Content-Type": "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", c.accessToken),
	}
	response, err := requests.PostJson[T](API_URL, headers, BDDPMap)
	if err != nil {
		return nil, fmt.Errorf("error making request with body %v and headers %v: %w", BDDPMap, headers, err)
	}
	return response, nil
} 

type AuthenticationResponse struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn int `json:"expires_in"`
}

type YoLinkMethod string
const (
	HomeGetDeviceList YoLinkMethod = "Home.getDeviceList"
	THSensorGetState YoLinkMethod = "THSensor.getState"
)


// General request types from https://doc.yosmart.com/docs/protocol/datapacket
// Basic Download Data Packet (request)
type BDDP struct {
    Time         int64                   `json:"time"`                   // Current timestamp, necessary
    Method       YoLinkMethod            `json:"method"`                 // Method to invoke, necessary
    MsgID        *string                 `json:"msgid,omitempty"`        // Optional, defaults to timestamp
    TargetDevice *string                 `json:"targetDevice,omitempty"` // Optional, needed if sending to a device
    Token        *string                 `json:"token,omitempty"`        // Optional, needed if sending to a device
    Params       *map[string]any         `json:"params,omitempty"`       // Optional, special methods require
}

// BDDP from YoLink with timestamp made optional. External usages of BDDP shouldn't need to worry about the timestamp
type SimpleBDDP struct {
    Time         *int64                  `json:"time"`                   // Current timestamp, neccesary
    Method       YoLinkMethod            `json:"method"`                 // Method to invoke, necessary
    MsgID        *string                 `json:"msgid,omitempty"`        // Optional, defaults to timestamp
    TargetDevice *string                 `json:"targetDevice,omitempty"` // Optional, needed if sending to a device
    Token        *string                 `json:"token,omitempty"`        // Optional, needed if sending to a device
    Params       *map[string]any         `json:"params,omitempty"`       // Optional, special methods require
}

// Basic Uplink Data Packet (response)
type BUDP struct {
    Time   int64                   `json:"time"`          // Current timestamp
    Method YoLinkMethod            `json:"method"`          // Method invoked
    MsgID  int                     `json:"msgid"`          // Same as request
    Code   string                  `json:"code"`          // Status code, '000000' = success
    Desc   *string                 `json:"desc,omitempty"`  // Optional description of status code
    Data   *map[string]any         `json:"data,omitempty"`  // Optional result data
}

type TypedBUDP[T any] struct {
    Time   int64                   `json:"time"`            // Current timestamp
    Method YoLinkMethod            `json:"method"`          // Method invoked
    MsgID  int                     `json:"msgid"`           // Same as request
    Code   string                  `json:"code"`            // Status code, '000000' = success
    Desc   *string                 `json:"desc,omitempty"`  // Optional description of status code
    Data   *T                      `json:"data,omitempty"`  // Optional result data
}

type YoLinkDevice struct {
    DeviceID   string
    DeviceUUID string
    Token      string
    Name       string 
    Kind       string `json:"type"`
}

type YoLinkDeviceList struct {
	Devices []YoLinkDevice
}