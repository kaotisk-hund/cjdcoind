package macaroons

import (
	"context"
	"os"
	"path"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
	"github.com/kaotisk-hund/cjdcoind/btcutil/util"
	"github.com/kaotisk-hund/cjdcoind/lnd/channeldb/kvdb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/checkers"
	macaroon "gopkg.in/macaroon.v2"
)

var (
	// DBFilename is the filename within the data directory which contains
	// the macaroon stores.
	DBFilename = "macaroons.db"

	// ErrMissingRootKeyID specifies the root key ID is missing.
	ErrMissingRootKeyID = Err.CodeWithDetail("ErrMissingRootKeyID",
		"missing root key ID")

	// ErrDeletionForbidden is used when attempting to delete the
	// DefaultRootKeyID or the encryptedKeyID.
	ErrDeletionForbidden = Err.CodeWithDetail("ErrDeletionForbidden",
		"the specified ID cannot be deleted")

	// PermissionEntityCustomURI is a special entity name for a permission
	// that does not describe an entity:action pair but instead specifies a
	// specific URI that needs to be granted access to. This can be used for
	// more fine-grained permissions where a macaroon only grants access to
	// certain methods instead of a whole list of methods that define the
	// same entity:action pairs. For example: uri:/lnrpc.Lightning/GetInfo
	// only gives access to the GetInfo call.
	PermissionEntityCustomURI = "uri"
)

// MacaroonValidator is an interface type that can check if macaroons are valid.
type MacaroonValidator interface {
	// ValidateMacaroon extracts the macaroon from the context's gRPC
	// metadata, checks its signature, makes sure all specified permissions
	// for the called method are contained within and finally ensures all
	// caveat conditions are met. A non-nil error is returned if any of the
	// checks fail.
	ValidateMacaroon(ctx context.Context,
		requiredPermissions []bakery.Op, fullMethod string) er.R
}

// Service encapsulates bakery.Bakery and adds a Close() method that zeroes the
// root key service encryption keys, as well as utility methods to validate a
// macaroon against the bakery and gRPC middleware for macaroon-based auth.
type Service struct {
	bakery.Bakery

	rks *RootKeyStorage

	// externalValidators is a map between an absolute gRPC URIs and the
	// corresponding external macaroon validator to be used for that URI.
	// If no external validator for an URI is specified, the service will
	// use the internal validator.
	externalValidators map[string]MacaroonValidator

	// StatelessInit denotes if the service was initialized in the stateless
	// mode where no macaroon files should be created on disk.
	StatelessInit bool
}

// NewService returns a service backed by the macaroon Bolt DB stored in the
// passed directory. The `checks` argument can be any of the `Checker` type
// functions defined in this package, or a custom checker if desired. This
// constructor prevents double-registration of checkers to prevent panics, so
// listing the same checker more than once is not harmful. Default checkers,
// such as those for `allow`, `time-before`, `declared`, and `error` caveats
// are registered automatically and don't need to be added.
func NewService(dir, location string, statelessInit bool,
	checks ...Checker) (*Service, er.R) {

	// Ensure that the path to the directory exists.
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, er.E(err)
		}
	}

	// Open the database that we'll use to store the primary macaroon key,
	// and all generated macaroons+caveats.
	macaroonDB, err := kvdb.Create(
		kvdb.BoltBackendName, path.Join(dir, DBFilename), true,
	)
	if err != nil {
		return nil, err
	}

	rootKeyStore, errr := NewRootKeyStorage(macaroonDB)
	if errr != nil {
		return nil, errr
	}

	macaroonParams := bakery.BakeryParams{
		Location:     location,
		RootKeyStore: rootKeyStore,
		// No third-party caveat support for now.
		// TODO(aakselrod): Add third-party caveat support.
		Locator: nil,
		Key:     nil,
	}

	svc := bakery.New(macaroonParams)

	// Register all custom caveat checkers with the bakery's checker.
	// TODO(aakselrod): Add more checks as required.
	checker := svc.Checker.FirstPartyCaveatChecker.(*checkers.Checker)
	for _, check := range checks {
		cond, fun := check()
		if !isRegistered(checker, cond) {
			checker.Register(cond, "std", fun)
		}
	}

	return &Service{
		Bakery:             *svc,
		rks:                rootKeyStore,
		externalValidators: make(map[string]MacaroonValidator),
		StatelessInit:      statelessInit,
	}, nil
}

// isRegistered checks to see if the required checker has already been
// registered in order to avoid a panic caused by double registration.
func isRegistered(c *checkers.Checker, name string) bool {
	if c == nil {
		return false
	}

	for _, info := range c.Info() {
		if info.Name == name &&
			info.Prefix == "" &&
			info.Namespace == "std" {
			return true
		}
	}

	return false
}

// RegisterExternalValidator registers a custom, external macaroon validator for
// the specified absolute gRPC URI. That validator is then fully responsible to
// make sure any macaroon passed for a request to that URI is valid and
// satisfies all conditions.
func (svc *Service) RegisterExternalValidator(fullMethod string,
	validator MacaroonValidator) er.R {

	if validator == nil {
		return er.Errorf("validator cannot be nil")
	}

	_, ok := svc.externalValidators[fullMethod]
	if ok {
		return er.Errorf("external validator for method %s already "+
			"registered", fullMethod)
	}

	svc.externalValidators[fullMethod] = validator
	return nil
}

// UnaryServerInterceptor is a GRPC interceptor that checks whether the
// request is authorized by the included macaroons.
func (svc *Service) UnaryServerInterceptor(
	permissionMap map[string][]bakery.Op) grpc.UnaryServerInterceptor {

	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		uriPermissions, ok := permissionMap[info.FullMethod]
		if !ok {
			return nil, er.Wrapped(er.Errorf("%s: unknown permissions "+
				"required for method", info.FullMethod))
		}

		// Find out if there is an external validator registered for
		// this method. Fall back to the internal one if there isn't.
		validator, ok := svc.externalValidators[info.FullMethod]
		if !ok {
			validator = svc
		}

		// Now that we know what validator to use, let it do its work.
		err := validator.ValidateMacaroon(
			ctx, uriPermissions, info.FullMethod,
		)
		if err != nil {
			return nil, er.Wrapped(err)
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor is a GRPC interceptor that checks whether the
// request is authorized by the included macaroons.
func (svc *Service) StreamServerInterceptor(
	permissionMap map[string][]bakery.Op) grpc.StreamServerInterceptor {

	return func(srv interface{}, ss grpc.ServerStream,
		info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		uriPermissions, ok := permissionMap[info.FullMethod]
		if !ok {
			return er.Wrapped(er.Errorf("%s: unknown permissions required "+
				"for method", info.FullMethod))
		}

		// Find out if there is an external validator registered for
		// this method. Fall back to the internal one if there isn't.
		validator, ok := svc.externalValidators[info.FullMethod]
		if !ok {
			validator = svc
		}

		// Now that we know what validator to use, let it do its work.
		err := validator.ValidateMacaroon(
			ss.Context(), uriPermissions, info.FullMethod,
		)
		if err != nil {
			return er.Wrapped(err)
		}

		return handler(srv, ss)
	}
}

// ValidateMacaroon validates the capabilities of a given request given a
// bakery service, context, and uri. Within the passed context.Context, we
// expect a macaroon to be encoded as request metadata using the key
// "macaroon".
func (svc *Service) ValidateMacaroon(ctx context.Context,
	requiredPermissions []bakery.Op, fullMethod string) er.R {

	// Get macaroon bytes from context and unmarshal into macaroon.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return er.Errorf("unable to get metadata from context")
	}
	if len(md["macaroon"]) != 1 {
		return er.Errorf("expected 1 macaroon, got %d",
			len(md["macaroon"]))
	}

	// With the macaroon obtained, we'll now decode the hex-string
	// encoding, then unmarshal it from binary into its concrete struct
	// representation.
	macBytes, err := util.DecodeHex(md["macaroon"][0])
	if err != nil {
		return err
	}
	mac := &macaroon.Macaroon{}
	errr := mac.UnmarshalBinary(macBytes)
	if errr != nil {
		return er.E(errr)
	}

	// Check the method being called against the permitted operation, the
	// expiration time and IP address and return the result.
	authChecker := svc.Checker.Auth(macaroon.Slice{mac})
	_, errr = authChecker.Allow(ctx, requiredPermissions...)

	// If the macaroon contains broad permissions and checks out, we're
	// done.
	if errr == nil {
		return nil
	}

	// To also allow the special permission of "uri:<FullMethod>" to be a
	// valid permission, we need to check it manually in case there is no
	// broader scope permission defined.
	_, errr = authChecker.Allow(ctx, bakery.Op{
		Entity: PermissionEntityCustomURI,
		Action: fullMethod,
	})
	return er.E(errr)
}

// Close closes the database that underlies the RootKeyStore and zeroes the
// encryption keys.
func (svc *Service) Close() er.R {
	return svc.rks.Close()
}

// CreateUnlock calls the underlying root key store's CreateUnlock and returns
// the result.
func (svc *Service) CreateUnlock(password *[]byte) er.R {
	return svc.rks.CreateUnlock(password)
}

// NewMacaroon wraps around the function Oven.NewMacaroon with the defaults,
//  - version is always bakery.LatestVersion;
//  - caveats is always nil.
// In addition, it takes a rootKeyID parameter, and puts it into the context.
// The context is passed through Oven.NewMacaroon(), in which calls the function
// RootKey(), that reads the context for rootKeyID.
func (svc *Service) NewMacaroon(
	ctx context.Context, rootKeyID []byte,
	ops ...bakery.Op) (*bakery.Macaroon, er.R) {

	// Check rootKeyID is not called with nil or empty bytes. We want the
	// caller to be aware the value of root key ID used, so we won't replace
	// it with the DefaultRootKeyID if not specified.
	if len(rootKeyID) == 0 {
		return nil, ErrMissingRootKeyID.Default()
	}

	// // Pass the root key ID to context.
	ctx = ContextWithRootKeyID(ctx, rootKeyID)

	m, e := svc.Oven.NewMacaroon(ctx, bakery.LatestVersion, nil, ops...)
	return m, er.E(e)
}

// ListMacaroonIDs returns all the root key ID values except the value of
// encryptedKeyID.
func (svc *Service) ListMacaroonIDs(ctxt context.Context) ([][]byte, er.R) {
	return svc.rks.ListMacaroonIDs(ctxt)
}

// DeleteMacaroonID removes one specific root key ID. If the root key ID is
// found and deleted, it will be returned.
func (svc *Service) DeleteMacaroonID(ctxt context.Context,
	rootKeyID []byte) ([]byte, er.R) {
	return svc.rks.DeleteMacaroonID(ctxt, rootKeyID)
}

// GenerateNewRootKey calls the underlying root key store's GenerateNewRootKey
// and returns the result.
func (svc *Service) GenerateNewRootKey() er.R {
	return svc.rks.GenerateNewRootKey()
}

// ChangePassword calls the underlying root key store's ChangePassword and
// returns the result.
func (svc *Service) ChangePassword(oldPw, newPw []byte) er.R {
	return svc.rks.ChangePassword(oldPw, newPw)
}
