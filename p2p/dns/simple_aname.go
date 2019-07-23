package dns

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net"
)

func Lookup(domain string) (ip net.IP, err error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		logrus.WithError(err).WithField("domain", domain).Warn("cound not resolve IP")
		err = errors.New("could not resolve IP")
		return
	}
	if len(ips) > 0 {
		return ips[0], nil
	}
	logrus.WithError(err).WithField("domain", domain).Warn("resolving IP returns no result")

	return nil, errors.New("resolving IP returns no result")
}
