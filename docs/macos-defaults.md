# macOS defaults catalog

PourOver applies nix-darwin-style user preferences with `defaults write`.
Unset keys are **unmanaged** — only keys you declare in `macos.defaults` are reconciled.

Full nix-darwin option tree (Homebrew, launchd, services, …): [nix-darwin-options.md](nix-darwin-options.md).
Upstream index: [MyNixOS nix-darwin options](https://mynixos.com/nix-darwin/options/system.defaults).

## Lua syntax

Named sections match nix-darwin (`system.defaults.dock.autohide` → `macos.defaults.dock.autohide`).
Hyphenated or spaced keys use Lua brackets.

```lua
macos = {
  defaults = {
    dock = {
      autohide = true,
      orientation = "left",
      ["show-recents"] = false,
      tilesize = 48,
      ["persistent-apps"] = {
        "/Applications/Safari.app",
        "/System/Applications/Utilities/Terminal.app",
      },
      ["persistent-others"] = {
        "~/Downloads",
        "~/Desktop",
      },
    },
    finder = {
      ShowPathbar = true,
      AppleShowAllExtensions = true,
    },
    NSGlobalDomain = {
      AppleShowAllExtensions = true,
      ["com.apple.swipescrolldirection"] = false,
    },
    screencapture = {
      location = "~/Desktop",
      type = "png",
    },
    trackpad = {
      Clicking = true,
    },
    -- CustomUserPreferences: any domain/key not in this catalog
    custom = {
      ["com.apple.Safari"] = {
        ShowFullURLInSmartSearchField = true,
      },
    },
  },
}
```

Types: **bool**, **int**, **float**, **string**, **array** (Dock `persistent-apps` / `persistent-others` path lists; encoded as nix-darwin-style tiles). After writes, PourOver restarts Dock / Finder / SystemUIServer / Calendar / Activity Monitor when that domain changed.

`macos.defaults.custom` is nix-darwin `CustomUserPreferences`. Machine-wide domains (`loginwindow`, `smb`, `SoftwareUpdate`) write under `/Library/Preferences` and need admin. `controlcenter` is ByHost on nix-darwin; PourOver writes `com.apple.controlcenter` (may need a logout).

**Not applied:** wallpaper, Finder sidebar Favorites.

## Discover more keys

```bash
defaults find autohide
defaults read-type com.apple.dock autohide
defaults read com.apple.dock autohide
```

Also: [macos-defaults.com](https://macos-defaults.com) and [nix-darwin defaults modules](https://github.com/nix-darwin/nix-darwin/tree/master/modules/system/defaults).

## Catalog

### `dock`

- Apple domain: `com.apple.dock`
- Scope: user
- Restart: `Dock`
- MyNixOS: [dock](https://mynixos.com/nix-darwin/options/system.defaults.dock)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `appswitcher-all-displays` | bool | `macos.defaults.dock["appswitcher-all-displays"]` | `defaults write com.apple.dock appswitcher-all-displays -bool <value>` |
| | | Whether to display the appswitcher on all displays or only the main one. The default is false. | |
| `autohide` | bool | `macos.defaults.dock.autohide` | `defaults write com.apple.dock autohide -bool <value>` |
| | | Whether to automatically hide and show the dock. The default is false. | |
| `autohide-delay` | float | `macos.defaults.dock["autohide-delay"]` | `defaults write com.apple.dock autohide-delay -float <value>` |
| | | Sets the speed of the autohide delay. The default is given in the example. | |
| `autohide-time-modifier` | float | `macos.defaults.dock["autohide-time-modifier"]` | `defaults write com.apple.dock autohide-time-modifier -float <value>` |
| | | Sets the speed of the animation when hiding/showing the Dock. The default is given in the example. | |
| `dashboard-in-overlay` | bool | `macos.defaults.dock["dashboard-in-overlay"]` | `defaults write com.apple.dock dashboard-in-overlay -bool <value>` |
| | | Whether to hide Dashboard as a Space. The default is false. | |
| `enable-spring-load-actions-on-all-items` | bool | `macos.defaults.dock["enable-spring-load-actions-on-all-items"]` | `defaults write com.apple.dock enable-spring-load-actions-on-all-items -bool <value>` |
| | | Enable spring loading for all Dock items. The default is false. | |
| `expose-animation-duration` | float | `macos.defaults.dock["expose-animation-duration"]` | `defaults write com.apple.dock expose-animation-duration -float <value>` |
| | | Sets the speed of the Mission Control animations. The default is given in the example. | |
| `expose-group-apps` | bool | `macos.defaults.dock["expose-group-apps"]` | `defaults write com.apple.dock expose-group-apps -bool <value>` |
| | | Whether to group windows by application in Mission Control's Exposé. The default is false. | |
| `largesize` | int | `macos.defaults.dock.largesize` | `defaults write com.apple.dock largesize -int <value>` |
| | | Magnified icon size on hover. The default is 16. | |
| `launchanim` | bool | `macos.defaults.dock.launchanim` | `defaults write com.apple.dock launchanim -bool <value>` |
| | | Animate opening applications from the Dock. The default is true. | |
| `magnification` | bool | `macos.defaults.dock.magnification` | `defaults write com.apple.dock magnification -bool <value>` |
| | | Magnify icon on hover. The default is false. | |
| `mineffect` | string | `macos.defaults.dock.mineffect` | `defaults write com.apple.dock mineffect -string <value>` |
| | | Set the minimize/maximize window effect. The default is genie. | |
| `minimize-to-application` | bool | `macos.defaults.dock["minimize-to-application"]` | `defaults write com.apple.dock minimize-to-application -bool <value>` |
| | | Whether to minimize windows into their application icon. The default is false. | |
| `mouse-over-hilite-stack` | bool | `macos.defaults.dock["mouse-over-hilite-stack"]` | `defaults write com.apple.dock mouse-over-hilite-stack -bool <value>` |
| | | Enable highlight hover effect for the grid view of a stack in the Dock. | |
| `mru-spaces` | bool | `macos.defaults.dock["mru-spaces"]` | `defaults write com.apple.dock mru-spaces -bool <value>` |
| | | Whether to automatically rearrange spaces based on most recent use. The default is true. | |
| `orientation` | string | `macos.defaults.dock.orientation` | `defaults write com.apple.dock orientation -string <value>` |
| | | Position of the dock on screen. The default is "bottom". | |
| `persistent-apps` | array | `macos.defaults.dock["persistent-apps"]` | `defaults write com.apple.dock persistent-apps <plist array>` |
| | | Persistent applications in the Dock (list of .app paths). Encoded as dock tiles like nix-darwin. | |
| `persistent-others` | array | `macos.defaults.dock["persistent-others"]` | `defaults write com.apple.dock persistent-others <plist array>` |
| | | Persistent folders/files on the Dock (right side). Paths; folders vs files inferred like nix-darwin. | |
| `scroll-to-open` | bool | `macos.defaults.dock["scroll-to-open"]` | `defaults write com.apple.dock scroll-to-open -bool <value>` |
| | | Scroll up on a Dock icon to show all Space's opened windows for an app, or open stack. The default is false. | |
| `show-process-indicators` | bool | `macos.defaults.dock["show-process-indicators"]` | `defaults write com.apple.dock show-process-indicators -bool <value>` |
| | | Show indicator lights for open applications in the Dock. The default is true. | |
| `show-recents` | bool | `macos.defaults.dock["show-recents"]` | `defaults write com.apple.dock show-recents -bool <value>` |
| | | Show recent applications in the dock. The default is true. | |
| `showAppExposeGestureEnabled` | bool | `macos.defaults.dock.showAppExposeGestureEnabled` | `defaults write com.apple.dock showAppExposeGestureEnabled -bool <value>` |
| | | Whether to enable trackpad gestures (three- or four-finger vertical swipe) to show App Exposé. The default is false. This feature interacts with `system.defaults.trackpad.TrackpadF | |
| `showDesktopGestureEnabled` | bool | `macos.defaults.dock.showDesktopGestureEnabled` | `defaults write com.apple.dock showDesktopGestureEnabled -bool <value>` |
| | | Whether to enable four-finger spread gesture to show the Desktop. The default is false. | |
| `showLaunchpadGestureEnabled` | bool | `macos.defaults.dock.showLaunchpadGestureEnabled` | `defaults write com.apple.dock showLaunchpadGestureEnabled -bool <value>` |
| | | Whether to enable four-finger pinch gesture to show the Launchpad. The default is false. | |
| `showMissionControlGestureEnabled` | bool | `macos.defaults.dock.showMissionControlGestureEnabled` | `defaults write com.apple.dock showMissionControlGestureEnabled -bool <value>` |
| | | Whether to enable trackpad gestures (three- or four-finger vertical swipe) to show Mission Control. The default is false. This feature interacts with `system.defaults.trackpad.Trac | |
| `showhidden` | bool | `macos.defaults.dock.showhidden` | `defaults write com.apple.dock showhidden -bool <value>` |
| | | Whether to make icons of hidden applications tranclucent. The default is false. | |
| `slow-motion-allowed` | bool | `macos.defaults.dock["slow-motion-allowed"]` | `defaults write com.apple.dock slow-motion-allowed -bool <value>` |
| | | Allow for slow-motion minimize effect while holding Shift key. The default is false. | |
| `static-only` | bool | `macos.defaults.dock["static-only"]` | `defaults write com.apple.dock static-only -bool <value>` |
| | | Show only open applications in the Dock. The default is false. | |
| `tilesize` | int | `macos.defaults.dock.tilesize` | `defaults write com.apple.dock tilesize -int <value>` |
| | | Size of the icons in the dock. The default is 64. | |
| `wvous-bl-corner` | int | `macos.defaults.dock["wvous-bl-corner"]` | `defaults write com.apple.dock wvous-bl-corner -int <value>` |
| | | Hot corner action for bottom left corner. Valid values include: * `1`: Disabled * `2`: Mission Control * `3`: Application Windows * `4`: Desktop * `5`: Start Screen Saver * `6`: Di | |
| `wvous-br-corner` | int | `macos.defaults.dock["wvous-br-corner"]` | `defaults write com.apple.dock wvous-br-corner -int <value>` |
| | | Hot corner action for bottom right corner. Valid values include: * `1`: Disabled * `2`: Mission Control * `3`: Application Windows * `4`: Desktop * `5`: Start Screen Saver * `6`: D | |
| `wvous-tl-corner` | int | `macos.defaults.dock["wvous-tl-corner"]` | `defaults write com.apple.dock wvous-tl-corner -int <value>` |
| | | Hot corner action for top left corner. Valid values include: * `1`: Disabled * `2`: Mission Control * `3`: Application Windows * `4`: Desktop * `5`: Start Screen Saver * `6`: Disab | |
| `wvous-tr-corner` | int | `macos.defaults.dock["wvous-tr-corner"]` | `defaults write com.apple.dock wvous-tr-corner -int <value>` |
| | | Hot corner action for top right corner. Valid values include: * `1`: Disabled * `2`: Mission Control * `3`: Application Windows * `4`: Desktop * `5`: Start Screen Saver * `6`: Disa | |

### `finder`

- Apple domain: `com.apple.finder`
- Scope: user
- Restart: `Finder`
- MyNixOS: [finder](https://mynixos.com/nix-darwin/options/system.defaults.finder)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `AppleShowAllExtensions` | bool | `macos.defaults.finder.AppleShowAllExtensions` | `defaults write com.apple.finder AppleShowAllExtensions -bool <value>` |
| | | Whether to always show file extensions. The default is false. | |
| `AppleShowAllFiles` | bool | `macos.defaults.finder.AppleShowAllFiles` | `defaults write com.apple.finder AppleShowAllFiles -bool <value>` |
| | | Whether to always show hidden files. The default is false. | |
| `CreateDesktop` | bool | `macos.defaults.finder.CreateDesktop` | `defaults write com.apple.finder CreateDesktop -bool <value>` |
| | | Whether to show icons on the desktop or not. The default is true. | |
| `FXDefaultSearchScope` | string | `macos.defaults.finder.FXDefaultSearchScope` | `defaults write com.apple.finder FXDefaultSearchScope -string <value>` |
| | | Change the default search scope. Use "SCcf" to default to current folder. The default is unset ("This Mac"). | |
| `FXEnableExtensionChangeWarning` | bool | `macos.defaults.finder.FXEnableExtensionChangeWarning` | `defaults write com.apple.finder FXEnableExtensionChangeWarning -bool <value>` |
| | | Whether to show warnings when change the file extension of files. The default is true. | |
| `FXPreferredViewStyle` | string | `macos.defaults.finder.FXPreferredViewStyle` | `defaults write com.apple.finder FXPreferredViewStyle -string <value>` |
| | | Change the default finder view. "icnv" = Icon view, "Nlsv" = List view, "clmv" = Column View, "Flwv" = Gallery View The default is icnv. | |
| `FXRemoveOldTrashItems` | bool | `macos.defaults.finder.FXRemoveOldTrashItems` | `defaults write com.apple.finder FXRemoveOldTrashItems -bool <value>` |
| | | Remove items in the trash after 30 days. The default is false. | |
| `NewWindowTarget` | string | `macos.defaults.finder.NewWindowTarget` | `defaults write com.apple.finder NewWindowTarget -string <value>` |
| | | Change the default folder shown in Finder windows. "Other" corresponds to the value of NewWindowTargetPath. The default is unset ("Recents"). | |
| `NewWindowTargetPath` | string | `macos.defaults.finder.NewWindowTargetPath` | `defaults write com.apple.finder NewWindowTargetPath -string <value>` |
| | | Sets the URI to open when NewWindowTarget is "Other". Spaces and similar characters must be escaped. If the value is invalid, Finder will open your home directory. Example: "file:/ | |
| `QuitMenuItem` | bool | `macos.defaults.finder.QuitMenuItem` | `defaults write com.apple.finder QuitMenuItem -bool <value>` |
| | | Whether to allow quitting of the Finder. The default is false. | |
| `ShowExternalHardDrivesOnDesktop` | bool | `macos.defaults.finder.ShowExternalHardDrivesOnDesktop` | `defaults write com.apple.finder ShowExternalHardDrivesOnDesktop -bool <value>` |
| | | Whether to show external disks on desktop. The default is true. | |
| `ShowHardDrivesOnDesktop` | bool | `macos.defaults.finder.ShowHardDrivesOnDesktop` | `defaults write com.apple.finder ShowHardDrivesOnDesktop -bool <value>` |
| | | Whether to show hard disks on desktop. The default is false. | |
| `ShowMountedServersOnDesktop` | bool | `macos.defaults.finder.ShowMountedServersOnDesktop` | `defaults write com.apple.finder ShowMountedServersOnDesktop -bool <value>` |
| | | Whether to show connected servers on desktop. The default is false. | |
| `ShowPathbar` | bool | `macos.defaults.finder.ShowPathbar` | `defaults write com.apple.finder ShowPathbar -bool <value>` |
| | | Show path breadcrumbs in finder windows. The default is false. | |
| `ShowRemovableMediaOnDesktop` | bool | `macos.defaults.finder.ShowRemovableMediaOnDesktop` | `defaults write com.apple.finder ShowRemovableMediaOnDesktop -bool <value>` |
| | | Whether to show removable media (CDs, DVDs and iPods) on desktop. The default is true. | |
| `ShowStatusBar` | bool | `macos.defaults.finder.ShowStatusBar` | `defaults write com.apple.finder ShowStatusBar -bool <value>` |
| | | Show status bar at bottom of finder windows with item/disk space stats. The default is false. | |
| `_FXEnableColumnAutoSizing` | bool | `macos.defaults.finder._FXEnableColumnAutoSizing` | `defaults write com.apple.finder _FXEnableColumnAutoSizing -bool <value>` |
| | | Resize columns to fit filenames. The default is false. | |
| `_FXShowPosixPathInTitle` | bool | `macos.defaults.finder._FXShowPosixPathInTitle` | `defaults write com.apple.finder _FXShowPosixPathInTitle -bool <value>` |
| | | Whether to show the full POSIX filepath in the window title. The default is false. | |
| `_FXSortFoldersFirst` | bool | `macos.defaults.finder._FXSortFoldersFirst` | `defaults write com.apple.finder _FXSortFoldersFirst -bool <value>` |
| | | Keep folders on top when sorting by name. The default is false. | |
| `_FXSortFoldersFirstOnDesktop` | bool | `macos.defaults.finder._FXSortFoldersFirstOnDesktop` | `defaults write com.apple.finder _FXSortFoldersFirstOnDesktop -bool <value>` |
| | | Keep folders on top when sorting by name on the desktop. The default is false. | |

### `NSGlobalDomain`

- Apple domain: `-g`
- Scope: user
- Restart: `SystemUIServer`
- MyNixOS: [NSGlobalDomain](https://mynixos.com/nix-darwin/options/system.defaults.NSGlobalDomain)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `AppleEnableMouseSwipeNavigateWithScrolls` | bool | `macos.defaults.NSGlobalDomain.AppleEnableMouseSwipeNavigateWithScrolls` | `defaults write -g AppleEnableMouseSwipeNavigateWithScrolls -bool <value>` |
| | | Enables swiping left or right with two fingers to navigate backward or forward. The default is true. | |
| `AppleEnableSwipeNavigateWithScrolls` | bool | `macos.defaults.NSGlobalDomain.AppleEnableSwipeNavigateWithScrolls` | `defaults write -g AppleEnableSwipeNavigateWithScrolls -bool <value>` |
| | | Enables swiping left or right with two fingers to navigate backward or forward. The default is true. | |
| `AppleFontSmoothing` | int | `macos.defaults.NSGlobalDomain.AppleFontSmoothing` | `defaults write -g AppleFontSmoothing -int <value>` |
| | | Sets the level of font smoothing (sub-pixel font rendering). | |
| `AppleICUForce24HourTime` | bool | `macos.defaults.NSGlobalDomain.AppleICUForce24HourTime` | `defaults write -g AppleICUForce24HourTime -bool <value>` |
| | | Whether to use 24-hour or 12-hour time. The default is based on region settings. | |
| `AppleIconAppearanceTheme` | string | `macos.defaults.NSGlobalDomain.AppleIconAppearanceTheme` | `defaults write -g AppleIconAppearanceTheme -string <value>` |
| | | Set icon and widget style To set to default mode, set this to `null` and you'll need to manually run {command}`defaults delete -g AppleIconAppearanceTheme`. This option requires lo | |
| `AppleInterfaceStyle` | string | `macos.defaults.NSGlobalDomain.AppleInterfaceStyle` | `defaults write -g AppleInterfaceStyle -string <value>` |
| | | Set to 'Dark' to enable dark mode. To set to light mode, set this to `null` and you'll need to manually run {command}`defaults delete -g AppleInterfaceStyle`. This option requires | |
| `AppleInterfaceStyleSwitchesAutomatically` | bool | `macos.defaults.NSGlobalDomain.AppleInterfaceStyleSwitchesAutomatically` | `defaults write -g AppleInterfaceStyleSwitchesAutomatically -bool <value>` |
| | | Whether to automatically switch between light and dark mode. The default is false. | |
| `AppleKeyboardUIMode` | int | `macos.defaults.NSGlobalDomain.AppleKeyboardUIMode` | `defaults write -g AppleKeyboardUIMode -int <value>` |
| | | Configures the keyboard control behavior. The default is 0. 0 = Disabled 2 = Enabled on Sonoma or later 3 = Enabled on older macOS versions | |
| `AppleMeasurementUnits` | string | `macos.defaults.NSGlobalDomain.AppleMeasurementUnits` | `defaults write -g AppleMeasurementUnits -string <value>` |
| | | Whether to use centimeters (metric) or inches (US, UK) as the measurement unit. The default is based on region settings. | |
| `AppleMetricUnits` | int | `macos.defaults.NSGlobalDomain.AppleMetricUnits` | `defaults write -g AppleMetricUnits -int <value>` |
| | | Whether to use the metric system. The default is based on region settings. | |
| `ApplePressAndHoldEnabled` | bool | `macos.defaults.NSGlobalDomain.ApplePressAndHoldEnabled` | `defaults write -g ApplePressAndHoldEnabled -bool <value>` |
| | | Whether to enable the press-and-hold feature. The default is true. | |
| `AppleScrollerPagingBehavior` | bool | `macos.defaults.NSGlobalDomain.AppleScrollerPagingBehavior` | `defaults write -g AppleScrollerPagingBehavior -bool <value>` |
| | | Jump to the spot that's clicked on the scroll bar. The default is false. | |
| `AppleShowAllExtensions` | bool | `macos.defaults.NSGlobalDomain.AppleShowAllExtensions` | `defaults write -g AppleShowAllExtensions -bool <value>` |
| | | Whether to show all file extensions in Finder. The default is false. | |
| `AppleShowAllFiles` | bool | `macos.defaults.NSGlobalDomain.AppleShowAllFiles` | `defaults write -g AppleShowAllFiles -bool <value>` |
| | | Whether to always show hidden files. The default is false. | |
| `AppleShowScrollBars` | string | `macos.defaults.NSGlobalDomain.AppleShowScrollBars` | `defaults write -g AppleShowScrollBars -string <value>` |
| | | When to show the scrollbars. Options are 'WhenScrolling', 'Automatic' and 'Always'. | |
| `AppleSpacesSwitchOnActivate` | bool | `macos.defaults.NSGlobalDomain.AppleSpacesSwitchOnActivate` | `defaults write -g AppleSpacesSwitchOnActivate -bool <value>` |
| | | Whether or not to switch to a workspace that has a window of the application open, that is switched to. The default is true. | |
| `AppleTemperatureUnit` | string | `macos.defaults.NSGlobalDomain.AppleTemperatureUnit` | `defaults write -g AppleTemperatureUnit -string <value>` |
| | | Whether to use Celsius or Fahrenheit. The default is based on region settings. | |
| `AppleWindowTabbingMode` | string | `macos.defaults.NSGlobalDomain.AppleWindowTabbingMode` | `defaults write -g AppleWindowTabbingMode -string <value>` |
| | | Sets the window tabbing when opening a new document: 'manual', 'always', or 'fullscreen'. The default is 'fullscreen'. | |
| `InitialKeyRepeat` | int | `macos.defaults.NSGlobalDomain.InitialKeyRepeat` | `defaults write -g InitialKeyRepeat -int <value>` |
| | | Apple menu > System Preferences > Keyboard If you press and hold certain keyboard keys when in a text area, the key’s character begins to repeat. For example, the Delete key contin | |
| `KeyRepeat` | int | `macos.defaults.NSGlobalDomain.KeyRepeat` | `defaults write -g KeyRepeat -int <value>` |
| | | Apple menu > System Preferences > Keyboard If you press and hold certain keyboard keys when in a text area, the key’s character begins to repeat. For example, the Delete key contin | |
| `NSAutomaticCapitalizationEnabled` | bool | `macos.defaults.NSGlobalDomain.NSAutomaticCapitalizationEnabled` | `defaults write -g NSAutomaticCapitalizationEnabled -bool <value>` |
| | | Whether to enable automatic capitalization. The default is true. | |
| `NSAutomaticDashSubstitutionEnabled` | bool | `macos.defaults.NSGlobalDomain.NSAutomaticDashSubstitutionEnabled` | `defaults write -g NSAutomaticDashSubstitutionEnabled -bool <value>` |
| | | Whether to enable smart dash substitution. The default is true. | |
| `NSAutomaticInlinePredictionEnabled` | bool | `macos.defaults.NSGlobalDomain.NSAutomaticInlinePredictionEnabled` | `defaults write -g NSAutomaticInlinePredictionEnabled -bool <value>` |
| | | Whether to enable inline predictive text. The default is true. | |
| `NSAutomaticPeriodSubstitutionEnabled` | bool | `macos.defaults.NSGlobalDomain.NSAutomaticPeriodSubstitutionEnabled` | `defaults write -g NSAutomaticPeriodSubstitutionEnabled -bool <value>` |
| | | Whether to enable smart period substitution. The default is true. | |
| `NSAutomaticQuoteSubstitutionEnabled` | bool | `macos.defaults.NSGlobalDomain.NSAutomaticQuoteSubstitutionEnabled` | `defaults write -g NSAutomaticQuoteSubstitutionEnabled -bool <value>` |
| | | Whether to enable smart quote substitution. The default is true. | |
| `NSAutomaticSpellingCorrectionEnabled` | bool | `macos.defaults.NSGlobalDomain.NSAutomaticSpellingCorrectionEnabled` | `defaults write -g NSAutomaticSpellingCorrectionEnabled -bool <value>` |
| | | Whether to enable automatic spelling correction. The default is true. | |
| `NSAutomaticWindowAnimationsEnabled` | bool | `macos.defaults.NSGlobalDomain.NSAutomaticWindowAnimationsEnabled` | `defaults write -g NSAutomaticWindowAnimationsEnabled -bool <value>` |
| | | Whether to animate opening and closing of windows and popovers. The default is true. | |
| `NSDisableAutomaticTermination` | bool | `macos.defaults.NSGlobalDomain.NSDisableAutomaticTermination` | `defaults write -g NSDisableAutomaticTermination -bool <value>` |
| | | Whether to disable the automatic termination of inactive apps. | |
| `NSDocumentSaveNewDocumentsToCloud` | bool | `macos.defaults.NSGlobalDomain.NSDocumentSaveNewDocumentsToCloud` | `defaults write -g NSDocumentSaveNewDocumentsToCloud -bool <value>` |
| | | Whether to save new documents to iCloud by default. The default is true. | |
| `NSNavPanelExpandedStateForSaveMode` | bool | `macos.defaults.NSGlobalDomain.NSNavPanelExpandedStateForSaveMode` | `defaults write -g NSNavPanelExpandedStateForSaveMode -bool <value>` |
| | | Whether to use expanded save panel by default. The default is false. | |
| `NSNavPanelExpandedStateForSaveMode2` | bool | `macos.defaults.NSGlobalDomain.NSNavPanelExpandedStateForSaveMode2` | `defaults write -g NSNavPanelExpandedStateForSaveMode2 -bool <value>` |
| | | Whether to use expanded save panel by default. The default is false. | |
| `NSScrollAnimationEnabled` | bool | `macos.defaults.NSGlobalDomain.NSScrollAnimationEnabled` | `defaults write -g NSScrollAnimationEnabled -bool <value>` |
| | | Whether to enable smooth scrolling. The default is true. | |
| `NSStatusItemSelectionPadding` | int | `macos.defaults.NSGlobalDomain.NSStatusItemSelectionPadding` | `defaults write -g NSStatusItemSelectionPadding -int <value>` |
| | | Sets the padding around status icons in the menu bar. | |
| `NSStatusItemSpacing` | int | `macos.defaults.NSGlobalDomain.NSStatusItemSpacing` | `defaults write -g NSStatusItemSpacing -int <value>` |
| | | Sets the spacing between status icons in the menu bar. | |
| `NSTableViewDefaultSizeMode` | int | `macos.defaults.NSGlobalDomain.NSTableViewDefaultSizeMode` | `defaults write -g NSTableViewDefaultSizeMode -int <value>` |
| | | Sets the size of the finder sidebar icons: 1 (small), 2 (medium) or 3 (large). The default is 3. | |
| `NSTextShowsControlCharacters` | bool | `macos.defaults.NSGlobalDomain.NSTextShowsControlCharacters` | `defaults write -g NSTextShowsControlCharacters -bool <value>` |
| | | Whether to display ASCII control characters using caret notation in standard text views. The default is false. | |
| `NSUseAnimatedFocusRing` | bool | `macos.defaults.NSGlobalDomain.NSUseAnimatedFocusRing` | `defaults write -g NSUseAnimatedFocusRing -bool <value>` |
| | | Whether to enable the focus ring animation. The default is true. | |
| `NSWindowResizeTime` | float | `macos.defaults.NSGlobalDomain.NSWindowResizeTime` | `defaults write -g NSWindowResizeTime -float <value>` |
| | | Sets the speed speed of window resizing. The default is given in the example. | |
| `NSWindowShouldDragOnGesture` | bool | `macos.defaults.NSGlobalDomain.NSWindowShouldDragOnGesture` | `defaults write -g NSWindowShouldDragOnGesture -bool <value>` |
| | | Whether to enable moving window by holding anywhere on it like on Linux. The default is false. | |
| `PMPrintingExpandedStateForPrint` | bool | `macos.defaults.NSGlobalDomain.PMPrintingExpandedStateForPrint` | `defaults write -g PMPrintingExpandedStateForPrint -bool <value>` |
| | | Whether to use the expanded print panel by default. The default is false. | |
| `PMPrintingExpandedStateForPrint2` | bool | `macos.defaults.NSGlobalDomain.PMPrintingExpandedStateForPrint2` | `defaults write -g PMPrintingExpandedStateForPrint2 -bool <value>` |
| | | Whether to use the expanded print panel by default. The default is false. | |
| `_HIHideMenuBar` | bool | `macos.defaults.NSGlobalDomain._HIHideMenuBar` | `defaults write -g _HIHideMenuBar -bool <value>` |
| | | Whether to autohide the menu bar. The default is false. | |
| `com.apple.keyboard.fnState` | bool | `macos.defaults.NSGlobalDomain["com.apple.keyboard.fnState"]` | `defaults write -g com.apple.keyboard.fnState -bool <value>` |
| | | Use F1, F2, etc. keys as standard function keys. | |
| `com.apple.mouse.tapBehavior` | int | `macos.defaults.NSGlobalDomain["com.apple.mouse.tapBehavior"]` | `defaults write -g com.apple.mouse.tapBehavior -int <value>` |
| | | Configures the trackpad tap behavior. Mode 1 enables tap to click. | |
| `com.apple.sound.beep.feedback` | int | `macos.defaults.NSGlobalDomain["com.apple.sound.beep.feedback"]` | `defaults write -g com.apple.sound.beep.feedback -int <value>` |
| | | Apple menu > System Preferences > Sound Make a feedback sound when the system volume changed. This setting accepts the integers 0 or 1. Defaults to 1. | |
| `com.apple.sound.beep.volume` | float | `macos.defaults.NSGlobalDomain["com.apple.sound.beep.volume"]` | `defaults write -g com.apple.sound.beep.volume -float <value>` |
| | | Apple menu > System Preferences > Sound Sets the beep/alert volume level from 0.000 (muted) to 1.000 (100% volume). 75% = 0.7788008 50% = 0.6065307 25% = 0.4723665 | |
| `com.apple.springing.delay` | float | `macos.defaults.NSGlobalDomain["com.apple.springing.delay"]` | `defaults write -g com.apple.springing.delay -float <value>` |
| | | Set the spring loading delay for directories. The default is given in the example. | |
| `com.apple.springing.enabled` | bool | `macos.defaults.NSGlobalDomain["com.apple.springing.enabled"]` | `defaults write -g com.apple.springing.enabled -bool <value>` |
| | | Whether to enable spring loading (expose) for directories. | |
| `com.apple.swipescrolldirection` | bool | `macos.defaults.NSGlobalDomain["com.apple.swipescrolldirection"]` | `defaults write -g com.apple.swipescrolldirection -bool <value>` |
| | | Whether to enable "Natural" scrolling direction. The default is true. | |
| `com.apple.trackpad.enableSecondaryClick` | bool | `macos.defaults.NSGlobalDomain["com.apple.trackpad.enableSecondaryClick"]` | `defaults write -g com.apple.trackpad.enableSecondaryClick -bool <value>` |
| | | Whether to enable trackpad secondary click. The default is true. | |
| `com.apple.trackpad.forceClick` | bool | `macos.defaults.NSGlobalDomain["com.apple.trackpad.forceClick"]` | `defaults write -g com.apple.trackpad.forceClick -bool <value>` |
| | | Whether to enable trackpad force click. | |
| `com.apple.trackpad.scaling` | float | `macos.defaults.NSGlobalDomain["com.apple.trackpad.scaling"]` | `defaults write -g com.apple.trackpad.scaling -float <value>` |
| | | Configures the trackpad tracking speed (0 to 3). The default is "1". | |
| `com.apple.trackpad.trackpadCornerClickBehavior` | int | `macos.defaults.NSGlobalDomain["com.apple.trackpad.trackpadCornerClickBehavior"]` | `defaults write -g com.apple.trackpad.trackpadCornerClickBehavior -int <value>` |
| | | Configures the trackpad corner click behavior. Mode 1 enables right click. | |

### `.GlobalPreferences`

- Apple domain: `.GlobalPreferences`
- Scope: user
- Restart: `SystemUIServer`
- MyNixOS: [.GlobalPreferences](https://mynixos.com/nix-darwin/options/system.defaults.%22.GlobalPreferences%22)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `com.apple.mouse.scaling` | float | `macos.defaults[".GlobalPreferences"]["com.apple.mouse.scaling"]` | `defaults write .GlobalPreferences com.apple.mouse.scaling -float <value>` |
| | | Sets the mouse tracking speed. Found in the "Mouse" section of "System Preferences". Set to -1.0 to disable mouse acceleration. | |
| `com.apple.sound.beep.sound` | string | `macos.defaults[".GlobalPreferences"]["com.apple.sound.beep.sound"]` | `defaults write .GlobalPreferences com.apple.sound.beep.sound -string <value>` |
| | | Sets the system-wide alert sound. Found under "Sound Effects" in the "Sound" section of "System Preferences". Look in "/System/Library/Sounds" for possible candidates. | |

### `trackpad`

- Apple domain: `com.apple.AppleMultitouchTrackpad`
- Also writes: `com.apple.driver.AppleBluetoothMultitouch.trackpad`
- Scope: user
- MyNixOS: [trackpad](https://mynixos.com/nix-darwin/options/system.defaults.trackpad)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `ActuateDetents` | bool | `macos.defaults.trackpad.ActuateDetents` | `defaults write com.apple.AppleMultitouchTrackpad ActuateDetents -bool <value>` |
| | | Whether to enable haptic feedback. The default is true. | |
| `ActuationStrength` | int | `macos.defaults.trackpad.ActuationStrength` | `defaults write com.apple.AppleMultitouchTrackpad ActuationStrength -int <value>` |
| | | 0 to enable Silent Clicking, 1 to disable. The default is 1. | |
| `Clicking` | bool | `macos.defaults.trackpad.Clicking` | `defaults write com.apple.AppleMultitouchTrackpad Clicking -bool <value>` |
| | | Whether to enable tap to click. The default is false. | |
| `DragLock` | bool | `macos.defaults.trackpad.DragLock` | `defaults write com.apple.AppleMultitouchTrackpad DragLock -bool <value>` |
| | | Whether to enable drag lock. The default is false. | |
| `Dragging` | bool | `macos.defaults.trackpad.Dragging` | `defaults write com.apple.AppleMultitouchTrackpad Dragging -bool <value>` |
| | | Whether to enable tap to drag. The default is false. | |
| `FirstClickThreshold` | int | `macos.defaults.trackpad.FirstClickThreshold` | `defaults write com.apple.AppleMultitouchTrackpad FirstClickThreshold -int <value>` |
| | | For normal click: 0 for light clicking, 1 for medium, 2 for firm. The default is 1. | |
| `ForceSuppressed` | bool | `macos.defaults.trackpad.ForceSuppressed` | `defaults write com.apple.AppleMultitouchTrackpad ForceSuppressed -bool <value>` |
| | | Whether to disable force click. The default is false. | |
| `SecondClickThreshold` | int | `macos.defaults.trackpad.SecondClickThreshold` | `defaults write com.apple.AppleMultitouchTrackpad SecondClickThreshold -int <value>` |
| | | For force touch: 0 for light clicking, 1 for medium, 2 for firm. The default is 1. | |
| `TrackpadCornerSecondaryClick` | int | `macos.defaults.trackpad.TrackpadCornerSecondaryClick` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadCornerSecondaryClick -int <value>` |
| | | Whether to enable secondary click: 0 to disable, 1 to set bottom-left corner, 2 to set bottom-right corner. The default is 0. | |
| `TrackpadFourFingerHorizSwipeGesture` | int | `macos.defaults.trackpad.TrackpadFourFingerHorizSwipeGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadFourFingerHorizSwipeGesture -int <value>` |
| | | Whether to enable four-finger horizontal swipe gesture: 0 to disable, 2 to swipe between full-screen applications. The default is 0. | |
| `TrackpadFourFingerPinchGesture` | int | `macos.defaults.trackpad.TrackpadFourFingerPinchGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadFourFingerPinchGesture -int <value>` |
| | | Whether to enable four-finger pinch gesture (spread shows the Desktop, pinch shows the Launchpad): 0 to disable, 2 to enable. The default is 0. This setting interacts with `system. | |
| `TrackpadFourFingerVertSwipeGesture` | int | `macos.defaults.trackpad.TrackpadFourFingerVertSwipeGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadFourFingerVertSwipeGesture -int <value>` |
| | | 0 to disable four finger vertical swipe gestures, 2 to enable (down for Mission Control, up for App Exposé). The default is 2. When both three- and four-finger vertical swipe gestu | |
| `TrackpadMomentumScroll` | bool | `macos.defaults.trackpad.TrackpadMomentumScroll` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadMomentumScroll -bool <value>` |
| | | Whether to use inertia when scrolling. The default is true. | |
| `TrackpadPinch` | bool | `macos.defaults.trackpad.TrackpadPinch` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadPinch -bool <value>` |
| | | Whether to enable two-finger pinch gesture for zooming in and out. The default is false. | |
| `TrackpadRightClick` | bool | `macos.defaults.trackpad.TrackpadRightClick` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadRightClick -bool <value>` |
| | | Whether to enable trackpad right click (two-finger tap/click). The default is false. | |
| `TrackpadRotate` | bool | `macos.defaults.trackpad.TrackpadRotate` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadRotate -bool <value>` |
| | | Whether to enable two-finger rotation gesture. The default is false. | |
| `TrackpadThreeFingerDrag` | bool | `macos.defaults.trackpad.TrackpadThreeFingerDrag` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadThreeFingerDrag -bool <value>` |
| | | Whether to enable three-finger drag. The default is false. | |
| `TrackpadThreeFingerHorizSwipeGesture` | int | `macos.defaults.trackpad.TrackpadThreeFingerHorizSwipeGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadThreeFingerHorizSwipeGesture -int <value>` |
| | | Whether to enable three-finger horizontal swipe gesture: 0 to disable, 1 to swipe between pages, 2 to swipe between full-screen applications. The default is 2. | |
| `TrackpadThreeFingerTapGesture` | int | `macos.defaults.trackpad.TrackpadThreeFingerTapGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadThreeFingerTapGesture -int <value>` |
| | | Whether to enable three-finger tap gesture: 0 to disable, 2 to trigger Look up & data detectors. The default is 2. | |
| `TrackpadThreeFingerVertSwipeGesture` | int | `macos.defaults.trackpad.TrackpadThreeFingerVertSwipeGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadThreeFingerVertSwipeGesture -int <value>` |
| | | Whether to enable three-finger vertical swipe gesture (down for Mission Control, up for App Exposé): 0 to disable, 2 to enable. The default is 2. This setting interacts with `syste | |
| `TrackpadTwoFingerDoubleTapGesture` | bool | `macos.defaults.trackpad.TrackpadTwoFingerDoubleTapGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadTwoFingerDoubleTapGesture -bool <value>` |
| | | Whether to enable smart zoom when double-tapping with two fingers. The default is false. | |
| `TrackpadTwoFingerFromRightEdgeSwipeGesture` | int | `macos.defaults.trackpad.TrackpadTwoFingerFromRightEdgeSwipeGesture` | `defaults write com.apple.AppleMultitouchTrackpad TrackpadTwoFingerFromRightEdgeSwipeGesture -int <value>` |
| | | Whether to enable two-finger swipe-from-right-edge gesture: 0 to disable, 3 to open Notification Center. The default is 0. | |

### `magicmouse`

- Apple domain: `com.apple.AppleMultitouchMouse`
- Also writes: `com.apple.driver.AppleMultitouchMouse.mouse`
- Scope: user
- MyNixOS: [magicmouse](https://mynixos.com/nix-darwin/options/system.defaults.magicmouse)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `MouseButtonMode` | string | `macos.defaults.magicmouse.MouseButtonMode` | `defaults write com.apple.AppleMultitouchMouse MouseButtonMode -string <value>` |
| | | "OneButton": any tap is a left click. "TwoButton": allow left- and right-clicking. | |

### `screencapture`

- Apple domain: `com.apple.screencapture`
- Scope: user
- Restart: `SystemUIServer`
- MyNixOS: [screencapture](https://mynixos.com/nix-darwin/options/system.defaults.screencapture)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `disable-shadow` | bool | `macos.defaults.screencapture["disable-shadow"]` | `defaults write com.apple.screencapture disable-shadow -bool <value>` |
| | | Disable drop shadow border around screencaptures. The default is false. | |
| `include-date` | bool | `macos.defaults.screencapture["include-date"]` | `defaults write com.apple.screencapture include-date -bool <value>` |
| | | Include date and time in screenshot filenames. The default is true. Screenshot 2024-01-09 at 13.27.20.png would be an example for true. Screenshot.png Screenshot 1.png would be an | |
| `location` | string | `macos.defaults.screencapture.location` | `defaults write com.apple.screencapture location -string <value>` |
| | | The filesystem path to which screencaptures should be written. | |
| `save-selections` | bool | `macos.defaults.screencapture["save-selections"]` | `defaults write com.apple.screencapture save-selections -bool <value>` |
| | | Remember the selection window of the last screencapture. The default is true. | |
| `show-thumbnail` | bool | `macos.defaults.screencapture["show-thumbnail"]` | `defaults write com.apple.screencapture show-thumbnail -bool <value>` |
| | | Show thumbnail after screencapture before writing to file. The default is true. | |
| `target` | string | `macos.defaults.screencapture.target` | `defaults write com.apple.screencapture target -string <value>` |
| | | Target to which screencapture should save screenshot to. The default is "file". Valid values include: * `file`: Saves as a file in location specified by `system.defaults.screencapt | |
| `type` | string | `macos.defaults.screencapture.type` | `defaults write com.apple.screencapture type -string <value>` |
| | | The image format to use, such as "jpg". | |

### `screensaver`

- Apple domain: `com.apple.screensaver`
- Scope: user
- MyNixOS: [screensaver](https://mynixos.com/nix-darwin/options/system.defaults.screensaver)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `askForPassword` | bool | `macos.defaults.screensaver.askForPassword` | `defaults write com.apple.screensaver askForPassword -bool <value>` |
| | | If true, the user is prompted for a password when the screen saver is unlocked or stopped. The default is false. | |
| `askForPasswordDelay` | int | `macos.defaults.screensaver.askForPasswordDelay` | `defaults write com.apple.screensaver askForPasswordDelay -int <value>` |
| | | The number of seconds to delay before the password will be required to unlock or stop the screen saver (the grace period). | |

### `spaces`

- Apple domain: `com.apple.spaces`
- Scope: user
- Restart: `Dock`
- MyNixOS: [spaces](https://mynixos.com/nix-darwin/options/system.defaults.spaces)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `spans-displays` | bool | `macos.defaults.spaces["spans-displays"]` | `defaults write com.apple.spaces spans-displays -bool <value>` |
| | | Apple menu > System Preferences > Mission Control Displays have separate Spaces (note a logout is required before this setting will take effect). false = each physical display has | |

### `menuExtraClock`

- Apple domain: `com.apple.menuextra.clock`
- Scope: user
- Restart: `SystemUIServer`
- MyNixOS: [menuExtraClock](https://mynixos.com/nix-darwin/options/system.defaults.menuExtraClock)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `FlashDateSeparators` | bool | `macos.defaults.menuExtraClock.FlashDateSeparators` | `defaults write com.apple.menuextra.clock FlashDateSeparators -bool <value>` |
| | | When enabled, the clock indicator (which by default is the colon) will flash on and off each second. Default is null. | |
| `IsAnalog` | bool | `macos.defaults.menuExtraClock.IsAnalog` | `defaults write com.apple.menuextra.clock IsAnalog -bool <value>` |
| | | Show an analog clock instead of a digital one. Default is null. | |
| `Show24Hour` | bool | `macos.defaults.menuExtraClock.Show24Hour` | `defaults write com.apple.menuextra.clock Show24Hour -bool <value>` |
| | | Show a 24-hour clock, instead of a 12-hour clock. Default is null. | |
| `ShowAMPM` | bool | `macos.defaults.menuExtraClock.ShowAMPM` | `defaults write com.apple.menuextra.clock ShowAMPM -bool <value>` |
| | | Show the AM/PM label. Useful if Show24Hour is false. Default is null. | |
| `ShowDate` | int | `macos.defaults.menuExtraClock.ShowDate` | `defaults write com.apple.menuextra.clock ShowDate -int <value>` |
| | | Show the full date. Default is null. 0 = When space allows 1 = Always 2 = Never | |
| `ShowDayOfMonth` | bool | `macos.defaults.menuExtraClock.ShowDayOfMonth` | `defaults write com.apple.menuextra.clock ShowDayOfMonth -bool <value>` |
| | | Show the day of the month. Default is null. | |
| `ShowDayOfWeek` | bool | `macos.defaults.menuExtraClock.ShowDayOfWeek` | `defaults write com.apple.menuextra.clock ShowDayOfWeek -bool <value>` |
| | | Show the day of the week. Default is null. | |
| `ShowSeconds` | bool | `macos.defaults.menuExtraClock.ShowSeconds` | `defaults write com.apple.menuextra.clock ShowSeconds -bool <value>` |
| | | Show the clock with second precision, instead of minutes. Default is null. | |

### `hitoolbox`

- Apple domain: `com.apple.HIToolbox`
- Scope: user
- MyNixOS: [hitoolbox](https://mynixos.com/nix-darwin/options/system.defaults.hitoolbox)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `AppleFnUsageType` | int | `macos.defaults.hitoolbox.AppleFnUsageType` | `defaults write com.apple.HIToolbox AppleFnUsageType -int <value>` |
| | | Fn key action (0=Do Nothing, 1=Change Input Source, 2=Show Emoji & Symbols, 3=Start Dictation). Restart required. | |

### `iCal`

- Apple domain: `com.apple.iCal`
- Scope: user
- Restart: `Calendar`
- MyNixOS: [iCal](https://mynixos.com/nix-darwin/options/system.defaults.iCal)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `CalendarSidebarShown` | bool | `macos.defaults.iCal.CalendarSidebarShown` | `defaults write com.apple.iCal CalendarSidebarShown -bool <value>` |
| | | Show calendar list. Restart Calendar.app to apply. | |
| `TimeZone support enabled` | bool | `macos.defaults.iCal["TimeZone support enabled"]` | `defaults write com.apple.iCal "TimeZone support enabled" -bool <value>` |
| | | Turn on time zone support in Calendar. | |
| `first day of week` | int | `macos.defaults.iCal["first day of week"]` | `defaults write com.apple.iCal "first day of week" -int <value>` |
| | | First day of week in Calendar (0=System Setting, 1=Sunday … 7=Saturday). | |

### `LaunchServices`

- Apple domain: `com.apple.LaunchServices`
- Scope: user
- MyNixOS: [LaunchServices](https://mynixos.com/nix-darwin/options/system.defaults.LaunchServices)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `LSQuarantine` | bool | `macos.defaults.LaunchServices.LSQuarantine` | `defaults write com.apple.LaunchServices LSQuarantine -bool <value>` |
| | | Whether to enable quarantine for downloaded applications. The default is true. | |

### `ActivityMonitor`

- Apple domain: `com.apple.ActivityMonitor`
- Scope: user
- Restart: `Activity Monitor`
- MyNixOS: [ActivityMonitor](https://mynixos.com/nix-darwin/options/system.defaults.ActivityMonitor)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `IconType` | int | `macos.defaults.ActivityMonitor.IconType` | `defaults write com.apple.ActivityMonitor IconType -int <value>` |
| | | Change the icon in the dock when running. * 0: Application Icon * 2: Network Usage * 3: Disk Activity * 5: CPU Usage * 6: CPU History Default is null. | |
| `OpenMainWindow` | bool | `macos.defaults.ActivityMonitor.OpenMainWindow` | `defaults write com.apple.ActivityMonitor OpenMainWindow -bool <value>` |
| | | Open the main window when opening Activity Monitor. Default is true. | |
| `ShowCategory` | int | `macos.defaults.ActivityMonitor.ShowCategory` | `defaults write com.apple.ActivityMonitor ShowCategory -int <value>` |
| | | Change which processes to show. * 100: All Processes * 101: All Processes, Hierarchally * 102: My Processes * 103: System Processes * 104: Other User Processes * 105: Active Proces | |
| `SortColumn` | string | `macos.defaults.ActivityMonitor.SortColumn` | `defaults write com.apple.ActivityMonitor SortColumn -string <value>` |
| | | Which column to sort the main activity page (such as "CPUUsage"). Default is null. | |
| `SortDirection` | int | `macos.defaults.ActivityMonitor.SortDirection` | `defaults write com.apple.ActivityMonitor SortDirection -int <value>` |
| | | The sort direction of the sort column (0 is decending). Default is null. | |

### `WindowManager`

- Apple domain: `com.apple.WindowManager`
- Scope: user
- MyNixOS: [WindowManager](https://mynixos.com/nix-darwin/options/system.defaults.WindowManager)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `AppWindowGroupingBehavior` | bool | `macos.defaults.WindowManager.AppWindowGroupingBehavior` | `defaults write com.apple.WindowManager AppWindowGroupingBehavior -bool <value>` |
| | | Grouping strategy when showing windows from an application. false means "One at a time" true means "All at once" | |
| `AutoHide` | bool | `macos.defaults.WindowManager.AutoHide` | `defaults write com.apple.WindowManager AutoHide -bool <value>` |
| | | Auto hide stage strip showing recent apps. Default is false. | |
| `EnableStandardClickToShowDesktop` | bool | `macos.defaults.WindowManager.EnableStandardClickToShowDesktop` | `defaults write com.apple.WindowManager EnableStandardClickToShowDesktop -bool <value>` |
| | | Click wallpaper to reveal desktop Clicking your wallpaper will move all windows out of the way to allow access to your desktop items and widgets. Default is true. false means "Only | |
| `EnableTiledWindowMargins` | bool | `macos.defaults.WindowManager.EnableTiledWindowMargins` | `defaults write com.apple.WindowManager EnableTiledWindowMargins -bool <value>` |
| | | Enable window margins when tiling windows. The default is true. | |
| `EnableTilingByEdgeDrag` | bool | `macos.defaults.WindowManager.EnableTilingByEdgeDrag` | `defaults write com.apple.WindowManager EnableTilingByEdgeDrag -bool <value>` |
| | | Enable dragging windows to screen edges to tile them. The default is true. | |
| `EnableTilingOptionAccelerator` | bool | `macos.defaults.WindowManager.EnableTilingOptionAccelerator` | `defaults write com.apple.WindowManager EnableTilingOptionAccelerator -bool <value>` |
| | | Enable holding alt to tile windows. The default is true. | |
| `EnableTopTilingByEdgeDrag` | bool | `macos.defaults.WindowManager.EnableTopTilingByEdgeDrag` | `defaults write com.apple.WindowManager EnableTopTilingByEdgeDrag -bool <value>` |
| | | Enable dragging windows to the menu bar to fill the screen. The default is true. | |
| `GloballyEnabled` | bool | `macos.defaults.WindowManager.GloballyEnabled` | `defaults write com.apple.WindowManager GloballyEnabled -bool <value>` |
| | | Enable Stage Manager Stage Manager arranges your recent windows into a single strip for reduced clutter and quick access. Default is false. | |
| `HideDesktop` | bool | `macos.defaults.WindowManager.HideDesktop` | `defaults write com.apple.WindowManager HideDesktop -bool <value>` |
| | | Hide items in Stage Manager. | |
| `StageManagerHideWidgets` | bool | `macos.defaults.WindowManager.StageManagerHideWidgets` | `defaults write com.apple.WindowManager StageManagerHideWidgets -bool <value>` |
| | | Hide widgets in Stage Manager. | |
| `StandardHideDesktopIcons` | bool | `macos.defaults.WindowManager.StandardHideDesktopIcons` | `defaults write com.apple.WindowManager StandardHideDesktopIcons -bool <value>` |
| | | Hide items on desktop. | |
| `StandardHideWidgets` | bool | `macos.defaults.WindowManager.StandardHideWidgets` | `defaults write com.apple.WindowManager StandardHideWidgets -bool <value>` |
| | | Hide widgets on desktop. | |

### `universalaccess`

- Apple domain: `com.apple.universalaccess`
- Scope: user
- MyNixOS: [universalaccess](https://mynixos.com/nix-darwin/options/system.defaults.universalaccess)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `closeViewScrollWheelToggle` | bool | `macos.defaults.universalaccess.closeViewScrollWheelToggle` | `defaults write com.apple.universalaccess closeViewScrollWheelToggle -bool <value>` |
| | | Use scroll gesture with the Ctrl (^) modifier key to zoom. The default is false. | |
| `closeViewZoomFollowsFocus` | bool | `macos.defaults.universalaccess.closeViewZoomFollowsFocus` | `defaults write com.apple.universalaccess closeViewZoomFollowsFocus -bool <value>` |
| | | Follow the keyboard focus while zoomed in. Without setting `closeViewScrollWheelToggle` this has no effect. The default is false. | |
| `mouseDriverCursorSize` | float | `macos.defaults.universalaccess.mouseDriverCursorSize` | `defaults write com.apple.universalaccess mouseDriverCursorSize -float <value>` |
| | | Set the size of cursor. 1 for normal, 4 for maximum. The default is 1. | |
| `reduceMotion` | bool | `macos.defaults.universalaccess.reduceMotion` | `defaults write com.apple.universalaccess reduceMotion -bool <value>` |
| | | Disable animation when switching screens or opening apps | |
| `reduceTransparency` | bool | `macos.defaults.universalaccess.reduceTransparency` | `defaults write com.apple.universalaccess reduceTransparency -bool <value>` |
| | | Disable transparency in the menu bar and elsewhere. The default is false. | |

### `controlcenter`

- Apple domain: `com.apple.controlcenter`
- Scope: byhost
- Restart: `SystemUIServer`
- MyNixOS: [controlcenter](https://mynixos.com/nix-darwin/options/system.defaults.controlcenter)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `AirDrop` | int | `macos.defaults.controlcenter.AirDrop` | `defaults write com.apple.controlcenter AirDrop -int <value>` |
| | | AirDrop icon in menu bar: 18=show, 24=hide. ByHost on nix-darwin. | |
| `BatteryShowPercentage` | bool | `macos.defaults.controlcenter.BatteryShowPercentage` | `defaults write com.apple.controlcenter BatteryShowPercentage -bool <value>` |
| | | Show battery percentage in menu bar. nix-darwin writes ByHost com.apple.controlcenter. | |
| `Bluetooth` | int | `macos.defaults.controlcenter.Bluetooth` | `defaults write com.apple.controlcenter Bluetooth -int <value>` |
| | | Bluetooth icon in menu bar: 18=show, 24=hide. ByHost on nix-darwin. | |
| `Display` | int | `macos.defaults.controlcenter.Display` | `defaults write com.apple.controlcenter Display -int <value>` |
| | | Display/brightness icon in menu bar: 18=show, 24=hide. ByHost on nix-darwin. | |
| `FocusModes` | int | `macos.defaults.controlcenter.FocusModes` | `defaults write com.apple.controlcenter FocusModes -int <value>` |
| | | Focus icon in menu bar: 18=show, 24=hide. ByHost on nix-darwin. | |
| `NowPlaying` | int | `macos.defaults.controlcenter.NowPlaying` | `defaults write com.apple.controlcenter NowPlaying -int <value>` |
| | | Now Playing icon in menu bar: 18=show, 24=hide. ByHost on nix-darwin. | |
| `Sound` | int | `macos.defaults.controlcenter.Sound` | `defaults write com.apple.controlcenter Sound -int <value>` |
| | | Sound icon in menu bar: 18=show, 24=hide. ByHost on nix-darwin. | |

### `loginwindow`

- Apple domain: `/Library/Preferences/com.apple.loginwindow`
- Scope: system
- MyNixOS: [loginwindow](https://mynixos.com/nix-darwin/options/system.defaults.loginwindow)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `DisableConsoleAccess` | bool | `macos.defaults.loginwindow.DisableConsoleAccess` | `defaults write /Library/Preferences/com.apple.loginwindow DisableConsoleAccess -bool <value>` |
| | | Disables the ability for a user to access the console by typing “>console” for a username at the login window. Default is false. | |
| `GuestEnabled` | bool | `macos.defaults.loginwindow.GuestEnabled` | `defaults write /Library/Preferences/com.apple.loginwindow GuestEnabled -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Allow users to login to the machine as guests using the Guest account. Default is true. | |
| `LoginwindowText` | string | `macos.defaults.loginwindow.LoginwindowText` | `defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText -string <value>` |
| | | Text to be shown on the login window. Default is "\\\\U03bb". | |
| `PowerOffDisabledWhileLoggedIn` | bool | `macos.defaults.loginwindow.PowerOffDisabledWhileLoggedIn` | `defaults write /Library/Preferences/com.apple.loginwindow PowerOffDisabledWhileLoggedIn -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options If set to true, the Power Off menu item will be disabled when the user is logged in. Default is false. | |
| `RestartDisabled` | bool | `macos.defaults.loginwindow.RestartDisabled` | `defaults write /Library/Preferences/com.apple.loginwindow RestartDisabled -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Hides the Restart button on the login screen. Default is false. | |
| `RestartDisabledWhileLoggedIn` | bool | `macos.defaults.loginwindow.RestartDisabledWhileLoggedIn` | `defaults write /Library/Preferences/com.apple.loginwindow RestartDisabledWhileLoggedIn -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Disables the “Restart” option when users are logged in. Default is false. | |
| `SHOWFULLNAME` | bool | `macos.defaults.loginwindow.SHOWFULLNAME` | `defaults write /Library/Preferences/com.apple.loginwindow SHOWFULLNAME -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Displays login window as a name and password field instead of a list of users. Default is false. | |
| `ShutDownDisabled` | bool | `macos.defaults.loginwindow.ShutDownDisabled` | `defaults write /Library/Preferences/com.apple.loginwindow ShutDownDisabled -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Hides the Shut Down button on the login screen. Default is false. | |
| `ShutDownDisabledWhileLoggedIn` | bool | `macos.defaults.loginwindow.ShutDownDisabledWhileLoggedIn` | `defaults write /Library/Preferences/com.apple.loginwindow ShutDownDisabledWhileLoggedIn -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Disables the "Shutdown" option when users are logged in. Default is false. | |
| `SleepDisabled` | bool | `macos.defaults.loginwindow.SleepDisabled` | `defaults write /Library/Preferences/com.apple.loginwindow SleepDisabled -bool <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Hides the Sleep button on the login screen. Default is false. | |
| `autoLoginUser` | string | `macos.defaults.loginwindow.autoLoginUser` | `defaults write /Library/Preferences/com.apple.loginwindow autoLoginUser -string <value>` |
| | | Apple menu > System Preferences > Users and Groups > Login Options Auto login the supplied user on boot. Default is Off. | |

### `smb`

- Apple domain: `/Library/Preferences/SystemConfiguration/com.apple.smb.server`
- Scope: system
- MyNixOS: [smb](https://mynixos.com/nix-darwin/options/system.defaults.smb)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `NetBIOSName` | string | `macos.defaults.smb.NetBIOSName` | `defaults write /Library/Preferences/SystemConfiguration/com.apple.smb.server NetBIOSName -string <value>` |
| | | Hostname to use for NetBIOS. | |
| `ServerDescription` | string | `macos.defaults.smb.ServerDescription` | `defaults write /Library/Preferences/SystemConfiguration/com.apple.smb.server ServerDescription -string <value>` |
| | | Hostname to use for sharing services. | |

### `SoftwareUpdate`

- Apple domain: `/Library/Preferences/com.apple.SoftwareUpdate`
- Scope: system
- MyNixOS: [SoftwareUpdate](https://mynixos.com/nix-darwin/options/system.defaults.SoftwareUpdate)

| Key | Type | Lua | `defaults write` |
|-----|------|-----|------------------|
| `AutomaticallyInstallMacOSUpdates` | bool | `macos.defaults.SoftwareUpdate.AutomaticallyInstallMacOSUpdates` | `defaults write /Library/Preferences/com.apple.SoftwareUpdate AutomaticallyInstallMacOSUpdates -bool <value>` |
| | | Automatically install Mac OS software updates. Defaults to false. | |

_Indexed 197 keys in 20 sections from nix-darwin `system.defaults`._
