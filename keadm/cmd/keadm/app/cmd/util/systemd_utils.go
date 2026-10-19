package util

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
)

const systemdDone = "done"

func EnableAndRunSystemdUnit(ctx context.Context, unit string, reload bool) error {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if reload {
		err = conn.ReloadContext(ctx)
		if err != nil {
			return err
		}
	}

	_, _, err = conn.EnableUnitFilesContext(ctx, []string{unit}, false, true)
	if err != nil {
		return fmt.Errorf("enable %s failed: %w", unit, err)
	}

	done := make(chan string, 1)
	_, err = conn.StartUnitContext(ctx, unit, "replace", done)
	if err != nil {
		return err
	}

	result := <-done
	if result != systemdDone {
		return fmt.Errorf("failed to start %s: %s", unit, result)
	}
	return nil
}

func DisableAndStopSystemdUnit(ctx context.Context, conn *dbus.Conn, unit string, reload bool) error {
	done := make(chan string, 1)
	_, err := conn.StopUnitContext(ctx, unit, "replace", done)
	if err != nil {
		return err
	}

	result := <-done
	if result != systemdDone {
		return fmt.Errorf("failed to stop %s: %s", unit, result)
	}

	_, err = conn.DisableUnitFilesContext(ctx, []string{unit}, false)
	if err != nil {
		return err
	}

	if reload {
		err = conn.ReloadContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
