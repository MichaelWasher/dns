package params

import (
	"net"
	"time"

	"github.com/qdm12/golibs/logging"
	libparams "github.com/qdm12/golibs/params"
	"github.com/qdm12/golibs/verification"
)

// Reader contains methods to obtain parameters.
type Reader interface {
	// DNS getters
	GetProviders() (providers []string, err error)
	GetPrivateAddresses() (privateIPs []net.IP, privateIPNets []*net.IPNet, err error)

	// Unbound getters
	GetListeningPort() (listeningPort uint16, err error)
	GetCaching() (caching bool, err error)
	GetVerbosity() (verbosityLevel uint8, err error)
	GetVerbosityDetails() (verbosityDetailsLevel uint8, err error)
	GetValidationLogLevel() (validationLogLevel uint8, err error)
	GetCheckUnbound() (check bool, err error)
	GetIPv4() (doIPv4 bool, err error)
	GetIPv6() (doIPv6 bool, err error)

	// Blocking getters
	GetMaliciousBlocking() (blocking bool, err error)
	GetSurveillanceBlocking() (blocking bool, err error)
	GetAdsBlocking() (blocking bool, err error)
	GetUnblockedHostnames() (hostnames []string, err error)
	GetBlockedHostnames() (hostnames []string, err error)
	GetBlockedIPs() (IPs []net.IP, IPNets []*net.IPNet, err error)

	// Update getters
	GetUpdatePeriod() (period time.Duration, err error)
}

type reader struct {
	envParams libparams.Env
	logger    logging.Logger
	verifier  verification.Verifier
}

// NewParamsReader returns a paramsReadeer object to read parameters from
// environment variables.
func NewParamsReader(logger logging.Logger) Reader {
	return &reader{
		envParams: libparams.NewEnv(),
		logger:    logger,
		verifier:  verification.NewVerifier(),
	}
}

func (r *reader) onRetroActive(oldKey, newKey string) {
	r.logger.Warn(
		"You are using the old environment variable %s, please consider changing it to %s",
		oldKey, newKey,
	)
}
