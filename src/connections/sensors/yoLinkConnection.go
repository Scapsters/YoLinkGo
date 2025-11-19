package sensors

import (
	"com/connections"
	"com/connections/db"
	"com/data"
	"com/utils"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const TOKEN_URL = "https://api.yosmart.com/open/yolink/token"
const API_URL = "https://api.yosmart.com/open/yolink/v2/api"

const TOKEN_REFRESH_BUFFER_MINUTES = 30

const YOLINK_BRAND_NAME = "yolink"

var _ SensorConnection = (*YoLinkConnection)(nil)

type YoLinkConnection struct {
	userId              string
	userKey             string
	accessToken         string
	refreshToken        string
	tokenExpirationTime int64
}

func NewYoLinkConnection(userId string, userKey string) (*YoLinkConnection, error) {
	c := &YoLinkConnection{
		userId:  userId,
		userKey: userKey,
	}
	err := c.Open()
	if err != nil {
		return nil, fmt.Errorf("error while opening new YoLink connection: %w", err)
	}
	status, description := c.Status()
	if status != connections.Good {
		return nil, fmt.Errorf("error while checking status of new YoLink connection. Connection status: %v, connection description: %v", status, description)
	}
	return c, nil
}

// Ensure the connection to YoLink is active, with 3 main paths of execution:
// Token is active and far from expiring: no actions taken.
// Token is active but close to expiring: token is refreshed using current token.
// No token exists or token is expired: fetch new token.
func (c *YoLinkConnection) Open() error {
	currentTime := utils.TimeSeconds()

	var hasToken = c.tokenExpirationTime != 0
	var isTokenNearlyExpired = hasToken && currentTime > c.tokenExpirationTime-TOKEN_REFRESH_BUFFER_MINUTES*60
	var isTokenExpired = hasToken && currentTime > c.tokenExpirationTime
	var response *AuthenticationResponse
	var err error

	if hasToken && !isTokenNearlyExpired {
		return nil
	}

	if !hasToken || isTokenExpired {
		utils.FDefaultSafeLog("getting access key through creds: %v", c)
		response, err = utils.PostForm[AuthenticationResponse](
			TOKEN_URL,
			map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     c.userId,
				"client_secret": c.userKey,
			},
		)
		if err != nil {
			return fmt.Errorf("error generating new access token with user id %v and client secret %v: %w", c.userId, c.userKey, err)
		}
		c.accessToken = response.AccessToken
		c.refreshToken = response.RefreshToken
		c.tokenExpirationTime = utils.TimeSeconds() + int64(response.ExpiresIn)
	}
	if isTokenNearlyExpired {
		utils.FDefaultSafeLog("refreshing token")
		err = c.refreshCurrentToken()
		if err != nil {
			return fmt.Errorf("error refreshing current tokens with connection %v: %w", c, err)
		}
	}

	return nil
}
func (c *YoLinkConnection) Close() error {
	c.accessToken = ""
	c.refreshToken = ""
	c.tokenExpirationTime = 0
	return nil
}
func (c *YoLinkConnection) Status() (connections.PingResult, string) {
	err := c.refreshCurrentToken()
	if err != nil {
		return connections.Bad, err.Error()
	}
	return connections.Good, "Successful ping via token refresh"
}
func (c *YoLinkConnection) GetDeviceState(device *data.StoreDevice) ([]data.Event, error) {
	// Verify device brand
	if device.Brand != YOLINK_BRAND_NAME {
		return nil, fmt.Errorf("GetDeviceState called on YoLinkConnection but given device is of brand %v", device.Brand)
	}
	// Make request
	deviceState, err := MakeYoLinkRequest[BUDP](c, SimpleBDDP{Method: YoLinkMethod(device.Kind + ".getState"), TargetDevice: &device.BrandID, Token: &device.Token})
	if deviceState.Code != "000000" {
		return nil, fmt.Errorf("code was non-zero: %v for device %v (name: %v) in connection %v at time %v", deviceState.Code, device.BrandID, device.Name, c, utils.TimeSeconds())
	}
	if err != nil {
		return nil, fmt.Errorf("error while quering device: %w", err)
	}
	// Process response
	dataMap, err := utils.ToMap[any](deviceState.Data)
	if err != nil {
		return nil, fmt.Errorf("error converting data %v: %w", deviceState.Data, err)
	}
	pairs := utils.FlattenMap(dataMap, []utils.KVPair{}, "")

	// Ensure necessary keys exist
	var hasReportAt bool
	for k := range dataMap {
		if k == "reportAt" {
			hasReportAt = true
		}
	}
	if !hasReportAt {
		return nil, fmt.Errorf("reportAt missing for sensor %v (name %v) at time %v", device.ID, device.Name, time.Now())
	}
	reportAtString, ok := dataMap["reportAt"].(string)
	if !ok {
		return nil, fmt.Errorf("error converting reportAt %v to string", dataMap["reportAt"])
	}
	eventTimestamp, err := time.Parse(time.RFC3339Nano, reportAtString)
	if err != nil {
		return nil, fmt.Errorf("error converting time %v to epoch seconds: %w", dataMap["reportAt"], err)
	}
	// Create events
	events := []data.Event{}
	for _, pair := range pairs {
		events = append(events, data.Event{
			EventSourceDeviceID: device.ID,
			RequestDeviceID:     "1",                     //TODO: what does this mean
			ResponseTimestamp:   deviceState.Time / 1000, // Convert to seconds
			EventTimestamp:      eventTimestamp.Unix(),
			FieldName:           pair.K,
			FieldValue:          pair.V,
		})
	}
	return events, nil
}
func (c *YoLinkConnection) GetManagedDevices(dbConnection db.DBConnection) (*data.IterablePaginatedData[data.StoreDevice], error) {
	brand := YOLINK_BRAND_NAME
	devices, err := dbConnection.Devices().Get(data.DeviceFilter{Brand: &brand})
	if err != nil {
		return nil, fmt.Errorf("error while searching for devices: %w", err)
	}
	return devices, nil
}
func MakeYoLinkRequest[T any](c *YoLinkConnection, simpleBDDP SimpleBDDP) (*T, error) {
	BDDPMap, err := utils.ToMap[any](simpleBDDP)
	if err != nil {
		return nil, fmt.Errorf("error converting body %v to map: %w", simpleBDDP, err)
	}
	BDDPMap["time"] = strconv.FormatInt(utils.TimeSeconds(), 10)

	err = c.Open() // Ensure tokens are up to date
	if err != nil {
		return nil, fmt.Errorf("error while opening yoLink connection while preparing for request %v: %w", BDDPMap, err)
	}
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", c.accessToken),
	}
	response, err := utils.PostJson[T](API_URL, headers, BDDPMap)
	if err != nil {
		return nil, fmt.Errorf("error making request with body %v and headers %v: %w", BDDPMap, headers, err)
	}
	return response, nil
}
func (c *YoLinkConnection) UpdateManagedDevices(dbConnection db.DBConnection) error {
	// Get device List
	result, err := MakeYoLinkRequest[TypedBUDP[YoLinkDeviceList]](c, SimpleBDDP{Method: HomeGetDeviceList})
	if err != nil {
		return fmt.Errorf("error while getting YoLink device list: %w", err)
	}
	if result == nil {
		return errors.New("YoLink device list null without associated error")
	}

	// Store unique devices
	numDevicesAdded := 0
	for _, device := range result.Data.Devices {
		// Check if device exists
		existingDevices, err := dbConnection.Devices().Get(data.DeviceFilter{ID: &device.DeviceID})
		if err != nil {
			return fmt.Errorf("error while scanning Devices for device ID %v: %w", device.DeviceID, err)
		}
		firstItem, err := existingDevices.Next()
		if err != nil {
			return fmt.Errorf("error getting first item: %w", err)
		}
		secondItem, err := existingDevices.Next()
		if err != nil {
			return fmt.Errorf("error getting second item: %w", err)
		}

		// Duplicates exist
		if firstItem != nil && secondItem != nil {
			utils.DefaultSafeLog(fmt.Sprintf("Device with ID %v has duplicate entries!", device.DeviceID))
		}
		// Item already exists
		if firstItem != nil {
			continue
		}

		// Add device otherwise
		err = dbConnection.Devices().Add(data.Device{
			Brand:     YOLINK_BRAND_NAME,
			Kind:      device.Kind,
			Name:      device.Name,
			Token:     device.Token,
			BrandID:   device.DeviceID,
			Timestamp: utils.TimeSeconds(),
		})
		if err != nil {
			return fmt.Errorf("error adding device %v: %w", device, err)
		}
		numDevicesAdded++
	}
	fmt.Printf("%v devices added\n", numDevicesAdded)

	return nil
}

// Refresh the current token. Requires an existing token to exist.
func (c *YoLinkConnection) refreshCurrentToken() error {
	response, err := utils.PostForm[AuthenticationResponse](
		TOKEN_URL,
		map[string]string{
			"grant_type":    "refresh_token",
			"client_id":     c.userId,
			"refresh_token": c.refreshToken,
		},
	)
	if err != nil {
		return fmt.Errorf("error refreshing token with user id %v and refresh token %v: %w", c.userId, c.refreshToken, err)
	}
	c.accessToken = response.AccessToken
	c.refreshToken = response.RefreshToken
	c.tokenExpirationTime = utils.TimeSeconds() + int64(response.ExpiresIn)
	return nil
}

type AuthenticationResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type YoLinkMethod string

const (
	HomeGetDeviceList YoLinkMethod = "Home.getDeviceList"
	THSensorGetState  YoLinkMethod = "THSensor.getState"
)

// General request types from https://doc.yosmart.com/docs/protocol/datapacket
// BDDP (Basid Data Download Packet) (request) from YoLink with timestamp made optional. External usages of BDDP shouldn't need to worry about the timestamp.
type SimpleBDDP struct {
	Time         *int64          `json:"time"`                   // Current timestamp, necessary, but might not matter
	Method       YoLinkMethod    `json:"method"`                 // Method to invoke, necessary
	MsgID        *string         `json:"msgid,omitempty"`        // Optional, defaults to timestamp
	TargetDevice *string         `json:"targetDevice,omitempty"` // Optional, needed if sending to a device
	Token        *string         `json:"token,omitempty"`        // Optional, needed if sending to a device
	Params       *map[string]any `json:"params,omitempty"`       // Optional, special methods require
}

// Basic Uplink Data Packet (response).
type BUDP struct {
	Time   int64           `json:"time"`           // Current timestamp in epoch milliseconds
	Method YoLinkMethod    `json:"method"`         // Method invoked
	MsgID  int             `json:"msgid"`          // Same as request
	Code   string          `json:"code"`           // Status code, '000000' = success
	Desc   *string         `json:"desc,omitempty"` // Optional description of status code
	Data   *map[string]any `json:"data,omitempty"` // Optional result data
}

type TypedBUDP[T any] struct {
	Time   int64        `json:"time"`           // Current timestamp in epoch milliseconds
	Method YoLinkMethod `json:"method"`         // Method invoked
	MsgID  int          `json:"msgid"`          // Same as request
	Code   string       `json:"code"`           // Status code, '000000' = success
	Desc   *string      `json:"desc,omitempty"` // Optional description of status code
	Data   *T           `json:"data,omitempty"` // Optional result data
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
