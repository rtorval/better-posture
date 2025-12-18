# Better Posture ✨

_`A posture reminder utility to promote ergonomic habits.`_

Better Posture is a lightweight and robust Windows system-tray utility that helps you maintain better posture and take regular breaks. Never forget to stretch or adjust your position again!

**Better Posture automatically adapts its interface to your system language.** 


## 💡 Features

* **Lightweight and Minimal:** As a native Go application that consumes minimal CPU and memory resources.
* **Non-Blocking Notifications:** Uses native Windows notifications (`MessageBoxW` + `toast`) in a non-blocking way, so the app remains responsive even while a message is visible.
* **Live Countdown Timer:** Displays the time remaining until the next reminder directly in the system tray tooltip and the menu itself.
* **Customizable Intervals:** Easily increase, decrease, or reset the reminder interval directly from the tray menu (from 1 minute up to 24 hours).
* **Pause & Resume:** You can pause and resume the reminders at any time.
* **Portable:** The application is a single executable file with no external dependencies.
* **Persistent Configuration:** Stores your settings in %APPDATA%/BetterPosture/settings.json.
* **Resilient & Safe:** Automatically recovers from missing/corrupted settings files and sanitizes invalid values.
* **Single-Instance Enforcement:** Prevents multiple copies from running at once.
* **System-Language Aware:** Automatically translates its interface based on your Windows system language.


## 🛠️ Installation & Usage

### 🚀 Download Ready-to-Use Executable

1.  Download the latest `BetterPosture.exe` from the [Releases page](https://github.com/rtorval/better-posture/releases).
2.  Run the executable. The app immediately starts in the background.
3.  Look for the **Posture Reminder icon** in your system tray (near the clock).

No installer, no setup.

### ⚙️ Build from Source (For Developers)

To build this project, you need Go 1.21+ installed.

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/rtorval/better-posture.git
    cd better-posture
    ```

2.  **Install Dependencies:**
    ```sh
    go mod tidy
    ```

3.  **Embed Icon and Windows Resources:**
    ```sh
    # Install the tool
    go install github.com/tc-hib/go-winres@latest

    # Generate Windows resource files (icon, metadata)
    go-winres make
    ```

4.  **Build the Executable:**
    The `-H=windowsgui` flag is crucial to prevent the console window from flashing/appearing upon execution.
    ```sh
    go build -ldflags="-H=windowsgui" -o BetterPosture.exe .
    ```


## 📖 Configuration

Your settings are saved automatically.

* **Path:** `%APPDATA%\BetterPosture\settings.json`

Example:

```json
{
    "interval_minutes": 3,
    "reminder_title": "Time to Move!",
    "reminder_message": "Stand up, stretch your back, and adjust your chair.",
    "system_language": "en-US"
}
```
Notes:
* `system_language` only reflects the language detected from your Windows system.
* Changing the system language or deleting the config will update the default reminder title and messag e automatically.
* Manual edits are possible while the app is closed.

## 💻 Technical Details

Better Posture is built entirely in Go and uses:

* systray — cross-platform tray integration
    [github.com/getlantern/systray](https://github.com/getlantern/systray)

* windows (syscall wrapper) — for native Win32 API calls
    [golang.org/x/sys/windows](https://pkg.go.dev/golang.org/x/sys/windows)

* toast — Windows 10+ toast notifications
    ["github.com/go-toast/toast"](https://github.com/go-toast/toast)

* go-winres — embeds icons and metadata into the Windows executable
    [github.com/tc-hib/go-winres](https://github.com/tc-hib/go-winres)

* BurntSushi/toml — TOML parser for Golang
    [github.com/BurntSushi/toml](https://github.com/BurntSushi/toml)


## ⚖️ License

**© 2025 Rodrigo Toraño Valle**

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