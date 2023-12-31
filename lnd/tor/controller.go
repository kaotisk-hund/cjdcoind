package tor

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/textproto"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/btcutil/util"
)

const (
	// success is the Tor Control response code representing a successful
	// request.
	success = 250

	// nonceLen is the length of a nonce generated by either the controller
	// or the Tor server
	nonceLen = 32

	// cookieLen is the length of the authentication cookie.
	cookieLen = 32

	// ProtocolInfoVersion is the `protocolinfo` version currently supported
	// by the Tor server.
	ProtocolInfoVersion = 1

	// MinTorVersion is the minimum supported version that the Tor server
	// must be running on. This is needed in order to create v3 onion
	// services through Tor's control port.
	MinTorVersion = "0.3.3.6"

	// authSafeCookie is the name of the SAFECOOKIE authentication method.
	authSafeCookie = "SAFECOOKIE"

	// authHashedPassword is the name of the HASHEDPASSWORD authentication
	// method.
	authHashedPassword = "HASHEDPASSWORD"

	// authNull is the name of the NULL authentication method.
	authNull = "NULL"
)

var (
	// serverKey is the key used when computing the HMAC-SHA256 of a message
	// from the server.
	serverKey = []byte("Tor safe cookie authentication " +
		"server-to-controller hash")

	// controllerKey is the key used when computing the HMAC-SHA256 of a
	// message from the controller.
	controllerKey = []byte("Tor safe cookie authentication " +
		"controller-to-server hash")
)

// Controller is an implementation of the Tor Control protocol. This is used in
// order to communicate with a Tor server. Its only supported method of
// authentication is the SAFECOOKIE method.
//
// NOTE: The connection to the Tor server must be authenticated before
// proceeding to send commands. Otherwise, the connection will be closed.
//
// TODO:
//   * if adding support for more commands, extend this with a command queue?
//   * place under sub-package?
//   * support async replies from the server
type Controller struct {
	// started is used atomically in order to prevent multiple calls to
	// Start.
	started int32

	// stopped is used atomically in order to prevent multiple calls to
	// Stop.
	stopped int32

	// conn is the underlying connection between the controller and the
	// Tor server. It provides read and write methods to simplify the
	// text-based messages within the connection.
	conn *textproto.Conn

	// controlAddr is the host:port the Tor server is listening locally for
	// controller connections on.
	controlAddr string

	// password, if non-empty, signals that the controller should attempt to
	// authenticate itself with the backing Tor daemon through the
	// HASHEDPASSWORD authentication method with this value.
	password string

	// version is the current version of the Tor server.
	version string

	// targetIPAddress is the IP address which we tell the Tor server to use
	// to connect to the LND node.  This is required when the Tor server
	// runs on another host, otherwise the service will not be reachable.
	targetIPAddress string
}

// NewController returns a new Tor controller that will be able to interact with
// a Tor server.
func NewController(controlAddr string, targetIPAddress string,
	password string) *Controller {

	return &Controller{
		controlAddr:     controlAddr,
		targetIPAddress: targetIPAddress,
		password:        password,
	}
}

// Start establishes and authenticates the connection between the controller and
// a Tor server. Once done, the controller will be able to send commands and
// expect responses.
func (c *Controller) Start() er.R {
	if !atomic.CompareAndSwapInt32(&c.started, 0, 1) {
		return nil
	}

	conn, err := textproto.Dial("tcp", c.controlAddr)
	if err != nil {
		return er.Errorf("unable to connect to Tor server: %v", err)
	}

	c.conn = conn

	return c.authenticate()
}

// Stop closes the connection between the controller and the Tor server.
func (c *Controller) Stop() er.R {
	if !atomic.CompareAndSwapInt32(&c.stopped, 0, 1) {
		return nil
	}

	return er.E(c.conn.Close())
}

// sendCommand sends a command to the Tor server and returns its response, as a
// single space-delimited string, and code.
func (c *Controller) sendCommand(command string) (int, string, er.R) {
	if err := c.conn.Writer.PrintfLine(command); err != nil {
		return 0, "", er.E(err)
	}

	// We'll use ReadResponse as it has built-in support for multi-line
	// text protocol responses.
	code, reply, err := c.conn.Reader.ReadResponse(success)
	if err != nil {
		return code, reply, er.E(err)
	}

	return code, reply, nil
}

// parseTorReply parses the reply from the Tor server after receiving a command
// from a controller. This will parse the relevant reply parameters into a map
// of keys and values.
func parseTorReply(reply string) map[string]string {
	params := make(map[string]string)

	// Replies can either span single or multiple lines, so we'll default
	// to stripping whitespace and newlines in order to retrieve the
	// individual contents of it. The -1 indicates that we want this to span
	// across all instances of a newline.
	contents := strings.Split(strings.Replace(reply, "\n", " ", -1), " ")
	for _, content := range contents {
		// Each parameter within the reply should be of the form
		// "KEY=VALUE". If the parameter doesn't contain "=", then we
		// can assume it does not provide any other relevant information
		// already known.
		keyValue := strings.SplitN(content, "=", 2)
		if len(keyValue) != 2 {
			continue
		}

		key := keyValue[0]
		value := keyValue[1]
		params[key] = value
	}

	return params
}

// authenticate authenticates the connection between the controller and the
// Tor server using either of the following supported authentication methods
// depending on its configuration: SAFECOOKIE, HASHEDPASSWORD, and NULL.
func (c *Controller) authenticate() er.R {
	protocolInfo, err := c.protocolInfo()
	if err != nil {
		return err
	}

	// With the version retrieved, we'll cache it now in case it needs to be
	// used later on.
	c.version = protocolInfo.version()

	switch {
	// If a password was provided, then we should attempt to use the
	// HASHEDPASSWORD authentication method.
	case c.password != "":
		if !protocolInfo.supportsAuthMethod(authHashedPassword) {
			return er.Errorf("%v authentication method not "+
				"supported", authHashedPassword)
		}

		return c.authenticateViaHashedPassword()

	// Otherwise, attempt to authentication via the SAFECOOKIE method as it
	// provides the most security.
	case protocolInfo.supportsAuthMethod(authSafeCookie):
		return c.authenticateViaSafeCookie(protocolInfo)

	// Fallback to the NULL method if any others aren't supported.
	case protocolInfo.supportsAuthMethod(authNull):
		return c.authenticateViaNull()

	// No supported authentication methods, fail.
	default:
		return er.New("the Tor server must be configured with " +
			"NULL, SAFECOOKIE, or HASHEDPASSWORD authentication")
	}
}

// authenticateViaNull authenticates the controller with the Tor server using
// the NULL authentication method.
func (c *Controller) authenticateViaNull() er.R {
	_, _, err := c.sendCommand("AUTHENTICATE")
	return err
}

// authenticateViaHashedPassword authenticates the controller with the Tor
// server using the HASHEDPASSWORD authentication method.
func (c *Controller) authenticateViaHashedPassword() er.R {
	cmd := fmt.Sprintf("AUTHENTICATE \"%s\"", c.password)
	_, _, err := c.sendCommand(cmd)
	return err
}

// authenticateViaSafeCookie authenticates the controller with the Tor server
// using the SAFECOOKIE authentication method.
func (c *Controller) authenticateViaSafeCookie(info protocolInfo) er.R {
	// Before proceeding to authenticate the connection, we'll retrieve
	// the authentication cookie of the Tor server. This will be used
	// throughout the authentication routine. We do this before as once the
	// authentication routine has begun, it is not possible to retrieve it
	// mid-way.
	cookie, err := c.getAuthCookie(info)
	if err != nil {
		return er.Errorf("unable to retrieve authentication cookie: "+
			"%v", err)
	}

	// Authenticating using the SAFECOOKIE authentication method is a two
	// step process. We'll kick off the authentication routine by sending
	// the AUTHCHALLENGE command followed by a hex-encoded 32-byte nonce.
	clientNonce := make([]byte, nonceLen)
	if _, err := rand.Read(clientNonce); err != nil {
		return er.Errorf("unable to generate client nonce: %v", err)
	}

	cmd := fmt.Sprintf("AUTHCHALLENGE SAFECOOKIE %x", clientNonce)
	_, reply, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	// If successful, the reply from the server should be of the following
	// format:
	//
	//	"250 AUTHCHALLENGE"
	//		SP "SERVERHASH=" ServerHash
	//		SP "SERVERNONCE=" ServerNonce
	//		CRLF
	//
	// We're interested in retrieving the SERVERHASH and SERVERNONCE
	// parameters, so we'll parse our reply to do so.
	replyParams := parseTorReply(reply)

	// Once retrieved, we'll ensure these values are of proper length when
	// decoded.
	serverHash, ok := replyParams["SERVERHASH"]
	if !ok {
		return er.New("server hash not found in reply")
	}
	decodedServerHash, err := util.DecodeHex(serverHash)
	if err != nil {
		return er.Errorf("unable to decode server hash: %v", err)
	}
	if len(decodedServerHash) != sha256.Size {
		return er.New("invalid server hash length")
	}

	serverNonce, ok := replyParams["SERVERNONCE"]
	if !ok {
		return er.New("server nonce not found in reply")
	}
	decodedServerNonce, err := util.DecodeHex(serverNonce)
	if err != nil {
		return er.Errorf("unable to decode server nonce: %v", err)
	}
	if len(decodedServerNonce) != nonceLen {
		return er.New("invalid server nonce length")
	}

	// The server hash above was constructed by computing the HMAC-SHA256
	// of the message composed of the cookie, client nonce, and server
	// nonce. We'll redo this computation ourselves to ensure the integrity
	// and authentication of the message.
	hmacMessage := bytes.Join(
		[][]byte{cookie, clientNonce, decodedServerNonce}, []byte{},
	)
	computedServerHash := computeHMAC256(serverKey, hmacMessage)
	if !hmac.Equal(computedServerHash, decodedServerHash) {
		return er.Errorf("expected server hash %x, got %x",
			decodedServerHash, computedServerHash)
	}

	// If the MAC check was successful, we'll proceed with the last step of
	// the authentication routine. We'll now send the AUTHENTICATE command
	// followed by a hex-encoded client hash constructed by computing the
	// HMAC-SHA256 of the same message, but this time using the controller's
	// key.
	clientHash := computeHMAC256(controllerKey, hmacMessage)
	if len(clientHash) != sha256.Size {
		return er.New("invalid client hash length")
	}

	cmd = fmt.Sprintf("AUTHENTICATE %x", clientHash)
	if _, _, err := c.sendCommand(cmd); err != nil {
		return err
	}

	return nil
}

// getAuthCookie retrieves the authentication cookie in bytes from the Tor
// server. Cookie authentication must be enabled for this to work.
func (c *Controller) getAuthCookie(info protocolInfo) ([]byte, er.R) {
	// Retrieve the cookie file path from the PROTOCOLINFO reply.
	cookieFilePath, ok := info["COOKIEFILE"]
	if !ok {
		return nil, er.New("COOKIEFILE not found in PROTOCOLINFO " +
			"reply")
	}
	cookieFilePath = strings.Trim(cookieFilePath, "\"")

	// Read the cookie from the file and ensure it has the correct length.
	cookie, err := ioutil.ReadFile(cookieFilePath)
	if err != nil {
		return nil, er.E(err)
	}

	if len(cookie) != cookieLen {
		return nil, er.New("invalid authentication cookie length")
	}

	return cookie, nil
}

// computeHMAC256 computes the HMAC-SHA256 of a key and message.
func computeHMAC256(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// supportsV3 is a helper function that parses the current version of the Tor
// server and determines whether it supports creationg v3 onion services through
// Tor's control port. The version string should be of the format:
//	major.minor.revision.build
func supportsV3(version string) er.R {
	// We'll split the minimum Tor version that's supported and the given
	// version in order to individually compare each number.
	parts := strings.Split(version, ".")
	if len(parts) != 4 {
		return er.New("version string is not of the format " +
			"major.minor.revision.build")
	}

	// It's possible that the build number (the last part of the version
	// string) includes a pre-release string, e.g. rc, beta, etc., so we'll
	// parse that as well.
	build := strings.Split(parts[len(parts)-1], "-")
	parts[len(parts)-1] = build[0]

	// Ensure that each part of the version string corresponds to a number.
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return er.E(err)
		}
	}

	// Once we've determined we have a proper version string of the format
	// major.minor.revision.build, we can just do a string comparison to
	// determine if it satisfies the minimum version supported.
	if version < MinTorVersion {
		return er.Errorf("version %v below minimum version supported "+
			"%v", version, MinTorVersion)
	}

	return nil
}

// protocolInfo is encompasses the details of a response to a PROTOCOLINFO
// command.
type protocolInfo map[string]string

// version returns the Tor version as reported by the server.
func (i protocolInfo) version() string {
	version := i["Tor"]
	return strings.Trim(version, "\"")
}

// supportsAuthMethod determines whether the Tor server supports the given
// authentication method.
func (i protocolInfo) supportsAuthMethod(method string) bool {
	methods, ok := i["METHODS"]
	if !ok {
		return false
	}
	return strings.Contains(methods, method)
}

// protocolInfo sends a "PROTOCOLINFO" command to the Tor server and returns its
// response.
func (c *Controller) protocolInfo() (protocolInfo, er.R) {
	cmd := fmt.Sprintf("PROTOCOLINFO %d", ProtocolInfoVersion)
	_, reply, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}

	return protocolInfo(parseTorReply(reply)), nil
}
