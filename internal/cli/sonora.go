package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/KyleStank/kstankcli/internal/process"
	"github.com/spf13/cobra"
)

const (
	ipsecReadinessTimeout  = 5 * time.Second
	ipsecReadinessInterval = 100 * time.Millisecond
	ipsecReadinessAttempts = int(ipsecReadinessTimeout/ipsecReadinessInterval) + 1
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
		Long:  "Start, stop, or restart the Sonora VPN. Supported platforms: Linux. Requires sudo and ipsec.",
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
	if err := options.Runner.Run(ctx, process.Command{Name: "sudo", Args: []string{"ipsec", "start"}}); err != nil {
		return fmt.Errorf("start IPsec service: %w", err)
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
		if err := wait(ctx, ipsecReadinessInterval); err != nil {
			return fmt.Errorf("wait for IPsec service readiness: %w", err)
		}
	}
	return fmt.Errorf("IPsec service did not become ready within %s: %w", ipsecReadinessTimeout, lastError)
}

func stopSonoraVPN(ctx context.Context, options Options) error {
	if err := options.Runner.Run(ctx, process.Command{Name: "sudo", Args: []string{"ipsec", "stop"}}); err != nil {
		return fmt.Errorf("stop IPsec service: %w", err)
	}
	return nil
}
