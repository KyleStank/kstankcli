package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KyleStank/kstankcli/internal/process"
	"github.com/spf13/cobra"
)

const (
	strongSwanService      = "strongswan-starter.service"
	ipsecReadinessTimeout  = 5 * time.Second
	ipsecShutdownTimeout   = 30 * time.Second
	ipsecPollingInterval   = 100 * time.Millisecond
	ipsecReadinessAttempts = int(ipsecReadinessTimeout/ipsecPollingInterval) + 1
	ipsecShutdownAttempts  = int(ipsecShutdownTimeout/ipsecPollingInterval) + 1
)

func newSonoraCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "sonora",
		Short: "Run Sonora development workflows",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newSonoraVPNCommand(options))
	return command
}

func newSonoraVPNCommand(options Options) *cobra.Command {
	command := &cobra.Command{
		Use:   "vpn",
		Short: "Control the Sonora VPN (Linux only)",
		Long:  "Start, stop, or restart the Sonora VPN. Supported platforms: Linux with systemd. Requires sudo, systemctl, systemd-run, ipsec, and pkill.",
		Args:  cobra.NoArgs,
	}
	for _, action := range []string{"start", "stop", "restart"} {
		command.AddCommand(&cobra.Command{
			Use:   action,
			Short: vpnActionDescription(action),
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				if options.GOOS != "linux" {
					return fmt.Errorf("sonora vpn %s is unsupported on %s (supported: linux)", action, options.GOOS)
				}
				return runSonoraVPN(cmd.Context(), options, action)
			},
		})
	}
	return command
}

func vpnActionDescription(action string) string {
	switch action {
	case "start":
		return "Start the Sonora VPN (Linux only)"
	case "stop":
		return "Stop the Sonora VPN (Linux only)"
	default:
		return "Restart the Sonora VPN (Linux only)"
	}
}

func runSonoraVPN(ctx context.Context, options Options, action string) error {
	switch action {
	case "start":
		return startSonoraVPN(ctx, options)
	case "stop":
		return stopSonoraVPN(ctx, options)
	case "restart":
		if err := stopSonoraVPN(ctx, options); err != nil {
			return err
		}
		return startSonoraVPN(ctx, options)
	default:
		return fmt.Errorf("unknown Sonora VPN action %q", action)
	}
}

func startSonoraVPN(ctx context.Context, options Options) error {
	if err := options.Runner.Run(ctx, process.Command{Name: "sudo", Args: []string{"systemctl", "start", strongSwanService}}); err != nil {
		return fmt.Errorf("start strongSwan service: %w", err)
	}
	if err := waitForIPsec(ctx, options); err != nil {
		return err
	}
	if err := options.Runner.Run(ctx, process.Command{Name: "sudo", Args: []string{"ipsec", "up", "sonoraquestv10"}}); err != nil {
		return fmt.Errorf("connect Sonora VPN: %w", err)
	}
	return nil
}

func waitForIPsec(ctx context.Context, options Options) error {
	wait := options.Wait
	if wait == nil {
		wait = waitWithContext
	}
	var lastError error
	for attempt := 0; attempt < ipsecReadinessAttempts; attempt++ {
		lastError = options.Runner.Run(ctx, process.Command{
			Name:  "sudo",
			Args:  []string{"ipsec", "listplugins"},
			Quiet: true,
		})
		if lastError == nil {
			return nil
		}
		if attempt == ipsecReadinessAttempts-1 {
			break
		}
		if err := wait(ctx, ipsecPollingInterval); err != nil {
			return fmt.Errorf("wait for IPsec service readiness: %w", err)
		}
	}
	return fmt.Errorf("IPsec service did not become ready within %s: %w", ipsecReadinessTimeout, lastError)
}

func stopSonoraVPN(ctx context.Context, options Options) error {
	// Continue to the service stop if the connection is already down. A confirmed
	// daemon shutdown guarantees that no Sonora security association remains.
	disconnectErr := options.Runner.Run(ctx, process.Command{Name: "sudo", Args: []string{"ipsec", "down", "sonoraquestv10"}})
	if err := options.Runner.Run(ctx, process.Command{Name: "sudo", Args: []string{"systemctl", "stop", strongSwanService}}); err != nil {
		return errors.Join(wrapSonoraDisconnectError(disconnectErr), fmt.Errorf("stop strongSwan service: %w", err))
	}
	if !isIPsecRunning(ctx, options) {
		return nil
	}
	if err := options.Runner.Run(ctx, process.Command{
		Name: "sudo",
		Args: []string{
			"systemd-run",
			"--quiet",
			"--wait",
			"--collect",
			"--service-type=exec",
			"/usr/bin/pkill",
			"--signal",
			"TERM",
			"--exact",
			"charon",
		},
	}); err != nil {
		if !isIPsecRunning(ctx, options) {
			return nil
		}
		return errors.Join(wrapSonoraDisconnectError(disconnectErr), fmt.Errorf("terminate lingering IPsec daemon through systemd: %w", err))
	}
	if err := waitForIPsecStop(ctx, options); err != nil {
		return errors.Join(wrapSonoraDisconnectError(disconnectErr), err)
	}
	return nil
}

func waitForIPsecStop(ctx context.Context, options Options) error {
	wait := options.Wait
	if wait == nil {
		wait = waitWithContext
	}
	for attempt := 0; attempt < ipsecShutdownAttempts; attempt++ {
		if !isIPsecRunning(ctx, options) {
			return nil
		}
		if attempt == ipsecShutdownAttempts-1 {
			break
		}
		if err := wait(ctx, ipsecPollingInterval); err != nil {
			return fmt.Errorf("wait for IPsec service shutdown: %w", err)
		}
	}
	return fmt.Errorf("IPsec daemon was still running %s after SIGTERM", ipsecShutdownTimeout)
}

func isIPsecRunning(ctx context.Context, options Options) bool {
	return options.Runner.Run(ctx, process.Command{
		Name:  "sudo",
		Args:  []string{"ipsec", "listplugins"},
		Quiet: true,
	}) == nil
}

func wrapSonoraDisconnectError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("disconnect Sonora VPN: %w", err)
}
