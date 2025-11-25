/*
   Better Posture - A posture reminder utility to promote ergonomic habits.
   Copyright (C) 2025  Rodrigo Toraño Valle

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/getlantern/systray"
	"github.com/go-toast/toast"
)

//go:embed assets/icon.ico
var iconData []byte

//go:embed LICENSE
var licenseData []byte

//go:embed THIRD_PARTY_LICENSES
var thirdPartyLicenses embed.FS

const (
	defaultInterval        = 3 // minutes
	defaultReminderTitle   = "Posture Reminder"
	defaultReminderMessage = "Time to check your posture!"

	minInterval = 1       // minutes
	maxInterval = 24 * 60 // 1440 minutes (24 hours)
)

const (
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_TOPMOST         = 0x00040000
	MB_SYSTEMMODAL     = 0x00001000
)

var (
	user32          = windows.NewLazySystemDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

type Config struct {
	IntervalMinutes int    `json:"interval_minutes"`
	ReminderTitle   string `json:"reminder_title"`
	ReminderMessage string `json:"reminder_message"`
}

func settingsPath() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			appdata = dir
		} else {
			if cwd, err := os.Getwd(); err == nil {
				appdata = cwd
			} else {
				appdata = os.TempDir()
			}
		}
	}
	dir := filepath.Join(appdata, "BetterPosture")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Warning: could not create config dir %s: %v\n", dir, err)
	}
	return filepath.Join(dir, "settings.json")
}

func licenseFilePath() string {
	dir := filepath.Dir(settingsPath())
	return filepath.Join(dir, "LICENSE.txt")
}

func iconPath() string {
	dir := filepath.Dir(settingsPath())
	return filepath.Join(dir, "better-posture.ico")
}

func ensureResourceFiles() {
	p := licenseFilePath()
	diskData, err := os.ReadFile(p)
	writeRequired := false

	if os.IsNotExist(err) || err != nil {
		writeRequired = true
	} else {
		if !bytes.Equal(diskData, licenseData) {
			writeRequired = true
		}
	}

	if writeRequired {
		if err := os.WriteFile(p, licenseData, 0644); err != nil {
			fmt.Printf("Error writing main license file to %s: %v\n", p, err)
		}
	}

	p = iconPath()
	diskData, err = os.ReadFile(p)
	writeRequired = false

	if os.IsNotExist(err) || err != nil {
		writeRequired = true
	} else {
		if !bytes.Equal(diskData, iconData) {
			writeRequired = true
		}
	}

	if writeRequired {
		if err := os.WriteFile(p, iconData, 0644); err != nil {
			fmt.Printf("Error writing icon to %s: %v\n", p, err)
		}
	}

	appDataDir := filepath.Dir(settingsPath())
	thirdPartyDir := filepath.Join(appDataDir, "THIRD_PARTY_LICENSES")
	if err := os.MkdirAll(thirdPartyDir, 0755); err != nil {
		fmt.Printf("Warning: could not create third party license directory %s: %v\n", thirdPartyDir, err)
	}

	_ = fs.WalkDir(thirdPartyLicenses, "THIRD_PARTY_LICENSES", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("Error walking embedded licenses: %v\n", err)
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath := strings.TrimPrefix(path, "THIRD_PARTY_LICENSES/")
		destPath := filepath.Join(thirdPartyDir, relPath)

		embeddedContent, readErr := thirdPartyLicenses.ReadFile(path)
		if readErr != nil {
			fmt.Printf("Error reading embedded license file %s: %v\n", path, readErr)
			return nil
		}

		diskContent, diskErr := os.ReadFile(destPath)

		shouldWrite := false

		if os.IsNotExist(diskErr) || !bytes.Equal(diskContent, embeddedContent) {
			shouldWrite = true
		}

		if shouldWrite {
			if writeErr := os.WriteFile(destPath, embeddedContent, 0644); writeErr != nil {
				fmt.Printf("Error writing third-party license file to %s: %v\n", destPath, writeErr)
			}
		}

		return nil
	})
}

func loadConfig() Config {
	defaultCfg := Config{
		IntervalMinutes: defaultInterval,
		ReminderTitle:   defaultReminderTitle,
		ReminderMessage: defaultReminderMessage,
	}

	p := settingsPath()
	data, err := os.ReadFile(p)
	if err != nil {
		if saveErr := saveConfig(defaultCfg); saveErr != nil {
			fmt.Printf("Warning: could not write default config: %v\n", saveErr)
		}
		return defaultCfg
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("Warning: invalid config JSON: %v — resetting to defaults\n", err)
		if saveErr := saveConfig(defaultCfg); saveErr != nil {
			fmt.Printf("Warning: could not write default config: %v\n", saveErr)
		}
		return defaultCfg
	}

	needsSave := false

	if cfg.IntervalMinutes < minInterval {
		cfg.IntervalMinutes = minInterval
		needsSave = true
	}

	if cfg.IntervalMinutes > maxInterval {
		cfg.IntervalMinutes = maxInterval
		needsSave = true
	}

	if cfg.ReminderTitle == "" {
		cfg.ReminderTitle = defaultReminderTitle
		needsSave = true
	}

	if cfg.ReminderMessage == "" {
		cfg.ReminderMessage = defaultReminderMessage
		needsSave = true
	}

	if needsSave {
		if saveErr := saveConfig(cfg); saveErr != nil {
			fmt.Printf("Warning: could not save adjusted config: %v\n", saveErr)
		}
	}

	return cfg
}

func saveConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(settingsPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func showMessage(title, message string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(message)

	procMessageBoxW.Call(0,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(MB_ICONWARNING))
}

func showToast(title, message string) error {
	notification := toast.Notification{
		AppID:    "Better Posture",
		Title:    title,
		Message:  message,
		Icon:     iconPath(),
		Audio:    toast.IM,
		Duration: toast.Long,
		Actions: []toast.Action{
			{Type: "protocol", Label: "OK", Arguments: ""},
		},
	}

	err := notification.Push()
	if err != nil {
		fmt.Printf("Error showing toast notification: %v\n", err)
	}

	return err
}

func showAbout() {
	mainLicensePath := licenseFilePath()
	thirdPartyLicensesDir := filepath.Join(filepath.Dir(settingsPath()), "THIRD_PARTY_LICENSES")

	aboutMessage := fmt.Sprintf(
		"Better Posture - A posture reminder utility to promote ergonomic habits.\n\n"+
			"Copyright (C) 2025  Rodrigo Toraño Valle\n\n"+
			"This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.\n\n"+
			"This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more details.\n\n"+
			"You should have received a copy of the GNU General Public License along with this program.  If not, see <https://www.gnu.org/licenses/>.\n\n"+
			"You can find the full GPLv3 license text in:\n%s\n\n"+
			"Required notices for third-party components (Apache-2.0, BSD-3-Clause) are located in the following folder:\n%s\n\n\n\n",
		mainLicensePath,
		thirdPartyLicensesDir,
	)

	t, _ := windows.UTF16PtrFromString("About Better Posture")
	m, _ := windows.UTF16PtrFromString(aboutMessage)

	procMessageBoxW.Call(0,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(MB_ICONINFORMATION))

	// go func() {
	// 	_ = exec.Command("notepad", mainLicensePath).Start()
	// }()
}

var instanceMutex windows.Handle
var cfgMutex sync.RWMutex

func enforceSingleInstance() bool {
	const mutexName = "Global\\BetterPostureMutex"
	h, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if err != nil {
		fmt.Printf("CreateMutex error: %v\n", err)
		return false
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(h)
		return false
	}
	instanceMutex = h
	return true
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	totalSeconds := int(d.Seconds())
	if totalSeconds < 0 {
		totalSeconds = 0
	}

	h := totalSeconds / 3600
	m := (totalSeconds / 60) % 60
	s := totalSeconds % 60

	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}

func main() {
	if !enforceSingleInstance() {
		return
	}
	systray.Run(onReady, onExit)
}

func onReady() {
	ensureResourceFiles()

	systray.SetIcon(iconData)
	systray.SetTitle("Better Posture")

	const baseTooltip = "✨Sit smart. Move often. Feel better."
	systray.SetTooltip(baseTooltip)

	cfg := loadConfig()

	mInfo := systray.AddMenuItem("About Better Posture", "Show application and licensing information")
	systray.AddSeparator()
	intervalDuration := time.Duration(cfg.IntervalMinutes) * time.Minute
	mIntervalLabel := systray.AddMenuItem(fmt.Sprintf("Interval: %s", formatDuration(intervalDuration)), "")
	mIntervalLabel.Disable()
	mCountdown := systray.AddMenuItem("Countdown:", "")
	mCountdown.Disable()
	systray.AddSeparator()
	mPlus1h := systray.AddMenuItem("Increase (+1 hour)", "")
	mMinus1h := systray.AddMenuItem("Decrease (-1 hour)", "")
	mPlus30m := systray.AddMenuItem("Increase (+30 min)", "")
	mMinus30m := systray.AddMenuItem("Decrease (-30 min)", "")
	mPlus5m := systray.AddMenuItem("Increase (+5 min)", "")
	mMinus5m := systray.AddMenuItem("Decrease (-5 min)", "")
	mPlus1m := systray.AddMenuItem("Increase (+1 min)", "")
	mMinus1m := systray.AddMenuItem("Decrease (-1 min)", "")
	systray.AddSeparator()
	mResetDefault := systray.AddMenuItem(fmt.Sprintf("Reset interval (%d min)", defaultInterval), "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit program")

	var lastTriggeredUnix int64
	atomic.StoreInt64(&lastTriggeredUnix, time.Now().UnixNano())

	var isMessageShowing atomic.Bool

	updateInterval := func(newInterval int) {
		if newInterval < minInterval {
			newInterval = minInterval
		}
		if newInterval > maxInterval {
			newInterval = maxInterval
		}

		cfgMutex.Lock()
		cfg.IntervalMinutes = newInterval
		saveErr := saveConfig(cfg)
		cfgMutex.Unlock()

		if saveErr != nil {
			fmt.Printf("Warning: could not save config: %v\n", saveErr)
		}

		d := time.Duration(cfg.IntervalMinutes) * time.Minute
		mIntervalLabel.SetTitle(fmt.Sprintf("Interval: %s", formatDuration(d)))
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if isMessageShowing.Load() {
					systray.SetTooltip(baseTooltip)
					mCountdown.SetTitle("Countdown:")
				} else {
					cfgMutex.RLock()
					intervalMinutes := cfg.IntervalMinutes
					cfgMutex.RUnlock()

					last := time.Unix(0, atomic.LoadInt64(&lastTriggeredUnix))
					nextTrigger := last.Add(time.Duration(intervalMinutes) * time.Minute)
					remaining := time.Until(nextTrigger)

					if remaining <= 0 {
						systray.SetTooltip(baseTooltip)
						mCountdown.SetTitle("Countdown:")
					} else {
						countdown := formatDuration(remaining)
						systray.SetTooltip(fmt.Sprintf("%s (%s)", baseTooltip, countdown))
						mCountdown.SetTitle(fmt.Sprintf("Countdown: %s", countdown))
					}
				}

				last := time.Unix(0, atomic.LoadInt64(&lastTriggeredUnix))

				cfgMutex.RLock()
				intervalMinutes := cfg.IntervalMinutes
				cfgMutex.RUnlock()

				if time.Since(last) >= time.Duration(intervalMinutes)*time.Minute && !isMessageShowing.Load() {
					isMessageShowing.Store(true)

					cfgMutex.RLock()
					title := cfg.ReminderTitle
					message := cfg.ReminderMessage
					cfgMutex.RUnlock()

					go func(tit, msg string) {
						err := showToast(tit, msg)
						if err != nil {
							showMessage(tit, msg)
						}
						isMessageShowing.Store(false)
						atomic.StoreInt64(&lastTriggeredUnix, time.Now().UnixNano())
					}(title, message)
				}

			case <-mPlus1m.ClickedCh:
				updateInterval(cfg.IntervalMinutes + 1)

			case <-mMinus1m.ClickedCh:
				updateInterval(cfg.IntervalMinutes - 1)

			case <-mPlus5m.ClickedCh:
				updateInterval(cfg.IntervalMinutes + 5)

			case <-mMinus5m.ClickedCh:
				updateInterval(cfg.IntervalMinutes - 5)

			case <-mPlus30m.ClickedCh:
				updateInterval(cfg.IntervalMinutes + 30)

			case <-mMinus30m.ClickedCh:
				updateInterval(cfg.IntervalMinutes - 30)

			case <-mPlus1h.ClickedCh:
				updateInterval(cfg.IntervalMinutes + 60)

			case <-mMinus1h.ClickedCh:
				updateInterval(cfg.IntervalMinutes - 60)

			case <-mResetDefault.ClickedCh:
				updateInterval(defaultInterval)

			case <-mInfo.ClickedCh:
				go showAbout()

			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	if instanceMutex != 0 {
		_ = windows.CloseHandle(instanceMutex)
		instanceMutex = 0
	}
}
