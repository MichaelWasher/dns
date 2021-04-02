package check

import (
	"context"
	"fmt"
	"net"
	"time"
)

func WaitForDNS(ctx context.Context, resolver *net.Resolver) (err error) {
	const maxTries = 10
	const hostToResolve = "github.com"
	const waitTime = 300 * time.Millisecond
	timer := time.NewTimer(waitTime)
	select {
	case <-timer.C:
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		return ctx.Err()
	}
	for try := 1; try <= maxTries; try++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err = resolver.LookupIP(ctx, "ip", hostToResolve)
		if err == nil {
			return nil
		}
		const msStep = 50
		waitTime := maxTries * msStep * time.Millisecond
		timer := time.NewTimer(waitTime)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		}
	}
	return fmt.Errorf("DNS is not working after %d tries: %s", maxTries, err)
}
