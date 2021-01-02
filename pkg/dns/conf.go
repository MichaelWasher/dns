package dns

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/qdm12/dns/internal/models"
)

func (c *configurator) MakeUnboundConf(settings models.UnboundSettings,
	hostnamesLines, ipsLines []string, username string, puid, pgid int) (err error) {
	configFilepath := filepath.Join(c.unboundDir, unboundConfigFilename)
	file, err := c.openFile(configFilepath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	lines := generateUnboundConf(settings, hostnamesLines, ipsLines, c.unboundDir, username)
	_, err = file.WriteString(strings.Join(lines, "\n"))
	if err != nil {
		_ = file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

// generateUnboundConf generates an Unbound configuration from the user provided settings.
func generateUnboundConf(settings models.UnboundSettings,
	hostnamesLines, ipsLines []string, unboundDir, username string) (
	lines []string) {
	const (
		yes = "yes"
		no  = "no"
	)
	ipv4, ipv6 := no, no
	if settings.IPv4 {
		ipv4 = yes
	}
	if settings.IPv6 {
		ipv6 = yes
	}
	serverSection := map[string]string{
		// Logging
		"verbosity":     strconv.Itoa(int(settings.VerbosityLevel)),
		"val-log-level": strconv.Itoa(int(settings.ValidationLogLevel)),
		"use-syslog":    "no",
		// Performance
		"num-threads":       "2",
		"prefetch":          "yes",
		"prefetch-key":      "yes",
		"key-cache-size":    "32m",
		"key-cache-slabs":   "4",
		"msg-cache-size":    "8m",
		"msg-cache-slabs":   "4",
		"rrset-cache-size":  "8m",
		"rrset-cache-slabs": "4",
		"cache-min-ttl":     "3600",
		"cache-max-ttl":     "9000",
		// Privacy
		"rrset-roundrobin": "yes",
		"hide-identity":    "yes",
		"hide-version":     "yes",
		// Security
		"tls-cert-bundle":       `"` + filepath.Join(unboundDir, cacertsFilename) + `"`,
		"root-hints":            `"` + filepath.Join(unboundDir, rootHints) + `"`,
		"trust-anchor-file":     `"` + filepath.Join(unboundDir, rootKey) + `"`,
		"harden-below-nxdomain": "yes",
		"harden-referral-path":  "yes",
		"harden-algo-downgrade": "yes",
		// Network
		"do-ip4":    ipv4,
		"do-ip6":    ipv6,
		"interface": "0.0.0.0",
		"port":      strconv.Itoa(int(settings.ListeningPort)),
		// Other
		"username": `"` + username + `"`,
		"include":  "include.conf",
	}

	for _, provider := range settings.Providers {
		if provider == LibreDNS {
			delete(serverSection, "trust-anchor-file")
		}
	}

	// Server
	lines = append(lines, "server:")
	serverLines := make([]string, len(serverSection))
	i := 0
	for k, v := range serverSection {
		serverLines[i] = "  " + k + ": " + v
		i++
	}
	sort.Slice(serverLines, func(i, j int) bool {
		return serverLines[i] < serverLines[j]
	})
	lines = append(lines, serverLines...)
	lines = append(lines, hostnamesLines...)
	lines = append(lines, ipsLines...)

	// Forward zone
	lines = append(lines, "forward-zone:")
	forwardZoneSection := map[string]string{
		"name":                 "\".\"",
		"forward-tls-upstream": "yes",
	}
	if settings.Caching {
		forwardZoneSection["forward-no-cache"] = "no"
	} else {
		forwardZoneSection["forward-no-cache"] = "yes"
	}
	forwardZoneLines := make([]string, len(forwardZoneSection))
	i = 0
	for k, v := range forwardZoneSection {
		forwardZoneLines[i] = "  " + k + ": " + v
		i++
	}
	sort.Slice(forwardZoneLines, func(i, j int) bool {
		return forwardZoneLines[i] < forwardZoneLines[j]
	})
	for _, provider := range settings.Providers {
		providerData, _ := GetProviderData(provider)
		for _, IP := range providerData.IPs {
			forwardZoneLines = append(forwardZoneLines,
				fmt.Sprintf("  forward-addr: %s@853#%s", IP.String(), providerData.Host))
		}
	}
	lines = append(lines, forwardZoneLines...)
	return lines
}

func (c *configurator) BuildBlocked(ctx context.Context, client *http.Client,
	blockMalicious, blockAds, blockSurveillance bool,
	blockedHostnames, blockedIPs, allowedHostnames []string) (
	hostnamesLines, ipsLines []string, errs []error) {
	chHostnames := make(chan []string)
	chIPs := make(chan []string)
	chErrors := make(chan []error)
	go func() {
		lines, errs := buildBlockedHostnames(ctx, client,
			blockMalicious, blockAds, blockSurveillance, blockedHostnames,
			allowedHostnames)
		chHostnames <- lines
		chErrors <- errs
	}()
	go func() {
		lines, errs := buildBlockedIPs(ctx, client, blockMalicious, blockAds, blockSurveillance, blockedIPs)
		chIPs <- lines
		chErrors <- errs
	}()
	n := 2
	for n > 0 {
		select {
		case lines := <-chHostnames:
			hostnamesLines = append(hostnamesLines, lines...)
		case lines := <-chIPs:
			ipsLines = append(ipsLines, lines...)
		case routineErrs := <-chErrors:
			errs = append(errs, routineErrs...)
			n--
		}
	}
	sort.Slice(hostnamesLines, func(i, j int) bool { // for unit tests really
		return hostnamesLines[i] < hostnamesLines[j]
	})
	sort.Slice(ipsLines, func(i, j int) bool { // for unit tests really
		return ipsLines[i] < ipsLines[j]
	})
	return hostnamesLines, ipsLines, errs
}

var ErrBadStatusCode = errors.New("bad HTTP status code")

func getList(ctx context.Context, client *http.Client, url string) (results []string, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	} else if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("%w: %d %s", ErrBadStatusCode, response.StatusCode, response.Status)
	}

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		return nil, err
	}

	if err := response.Body.Close(); err != nil {
		return nil, err
	}

	results = strings.Split(string(content), "\n")

	// remove empty lines
	last := len(results) - 1
	for i := range results {
		if len(results[i]) == 0 {
			results[i] = results[last]
			last--
		}
	}
	results = results[:last+1]

	if len(results) == 0 {
		return nil, nil
	}
	return results, nil
}

func buildBlockedHostnames(ctx context.Context, client *http.Client, blockMalicious, blockAds, blockSurveillance bool,
	blockedHostnames, allowedHostnames []string) (lines []string, errs []error) {
	chResults := make(chan []string)
	chError := make(chan error)
	listsLeftToFetch := 0
	if blockMalicious {
		listsLeftToFetch++
		go func() {
			results, err := getList(ctx, client, string(maliciousBlockListHostnamesURL))
			chResults <- results
			chError <- err
		}()
	}
	if blockAds {
		listsLeftToFetch++
		go func() {
			results, err := getList(ctx, client, string(adsBlockListHostnamesURL))
			chResults <- results
			chError <- err
		}()
	}
	if blockSurveillance {
		listsLeftToFetch++
		go func() {
			results, err := getList(ctx, client, string(surveillanceBlockListHostnamesURL))
			chResults <- results
			chError <- err
		}()
	}
	uniqueResults := make(map[string]struct{})
	for listsLeftToFetch > 0 {
		select {
		case results := <-chResults:
			for _, result := range results {
				uniqueResults[result] = struct{}{}
			}
		case err := <-chError:
			listsLeftToFetch--
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, blockedHostname := range blockedHostnames {
		allowed := false
		for _, allowedHostname := range allowedHostnames {
			if blockedHostname == allowedHostname || strings.HasSuffix(blockedHostname, "."+allowedHostname) {
				allowed = true
			}
		}
		if allowed {
			continue
		}
		uniqueResults[blockedHostname] = struct{}{}
	}
	for _, allowedHostname := range allowedHostnames {
		delete(uniqueResults, allowedHostname)
	}
	for result := range uniqueResults {
		lines = append(lines, "  local-zone: \""+result+"\" static")
	}
	return lines, errs
}

func buildBlockedIPs(ctx context.Context, client *http.Client, blockMalicious, blockAds, blockSurveillance bool,
	blockedIPs []string) (lines []string, errs []error) {
	chResults := make(chan []string)
	chError := make(chan error)
	listsLeftToFetch := 0
	if blockMalicious {
		listsLeftToFetch++
		go func() {
			results, err := getList(ctx, client, string(maliciousBlockListIPsURL))
			chResults <- results
			chError <- err
		}()
	}
	if blockAds {
		listsLeftToFetch++
		go func() {
			results, err := getList(ctx, client, string(adsBlockListIPsURL))
			chResults <- results
			chError <- err
		}()
	}
	if blockSurveillance {
		listsLeftToFetch++
		go func() {
			results, err := getList(ctx, client, string(surveillanceBlockListIPsURL))
			chResults <- results
			chError <- err
		}()
	}
	uniqueResults := make(map[string]struct{})
	for listsLeftToFetch > 0 {
		select {
		case results := <-chResults:
			for _, result := range results {
				uniqueResults[result] = struct{}{}
			}
		case err := <-chError:
			listsLeftToFetch--
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	for _, blockedIP := range blockedIPs {
		uniqueResults[blockedIP] = struct{}{}
	}
	for result := range uniqueResults {
		lines = append(lines, "  private-address: "+result)
	}
	return lines, errs
}
