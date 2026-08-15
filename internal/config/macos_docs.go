package config

import (
	"fmt"
	"strings"
)

const macosDefaultsDocIntro = `# macOS defaults catalog

PourOver applies nix-darwin-style user preferences with ` + "`defaults write`" + `.
Unset keys are **unmanaged** — only keys you declare in ` + "`macos.defaults`" + ` are reconciled.

Full nix-darwin option tree (Homebrew, launchd, services, …): [nix-darwin-options.md](nix-darwin-options.md).
Upstream index: [MyNixOS nix-darwin options](https://mynixos.com/nix-darwin/options/system.defaults).

## Lua syntax

Named sections match nix-darwin (` + "`system.defaults.dock.autohide`" + ` → ` + "`macos.defaults.dock.autohide`" + `).
Hyphenated or spaced keys use Lua brackets.

` + "```lua" + `
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
` + "```" + `

Types: **bool**, **int**, **float**, **string**, **array** (Dock ` + "`persistent-apps`" + ` / ` + "`persistent-others`" + ` path lists; encoded as nix-darwin-style tiles). After writes, PourOver restarts Dock / Finder / SystemUIServer / Calendar / Activity Monitor when that domain changed.

` + "`macos.defaults.custom`" + ` is nix-darwin ` + "`CustomUserPreferences`" + `. Machine-wide domains (` + "`loginwindow`" + `, ` + "`smb`" + `, ` + "`SoftwareUpdate`" + `) write under ` + "`/Library/Preferences`" + ` and need admin. ` + "`controlcenter`" + ` is ByHost on nix-darwin; PourOver writes ` + "`com.apple.controlcenter`" + ` (may need a logout).

**Not applied:** wallpaper, Finder sidebar Favorites.

## Discover more keys

` + "```bash" + `
defaults find autohide
defaults read-type com.apple.dock autohide
defaults read com.apple.dock autohide
` + "```" + `

Also: [macos-defaults.com](https://macos-defaults.com) and [nix-darwin defaults modules](https://github.com/nix-darwin/nix-darwin/tree/master/modules/system/defaults).

## Catalog

`

// RenderMacOSDefaultsMarkdown builds the searchable catalog page from the embedded YAML.
func RenderMacOSDefaultsMarkdown() (string, error) {
	if err := CatalogLoadError(); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(macosDefaultsDocIntro)
	n := 0
	for _, sec := range Catalog() {
		n += len(sec.Keys)
		fmt.Fprintf(&b, "### `%s`\n\n", sec.Name)
		fmt.Fprintf(&b, "- Apple domain: `%s`\n", sec.Domain)
		if len(sec.AlsoDomains) > 0 {
			fmt.Fprintf(&b, "- Also writes: `%s`\n", strings.Join(sec.AlsoDomains, "`, `"))
		}
		fmt.Fprintf(&b, "- Scope: %s\n", sec.Scope)
		if sec.Killall != "" {
			fmt.Fprintf(&b, "- Restart: `%s`\n", sec.Killall)
		}
		if sec.MyNixOS != "" {
			fmt.Fprintf(&b, "- MyNixOS: [%s](%s)\n", sec.Name, sec.MyNixOS)
		}
		b.WriteString("\n")
		b.WriteString("| Key | Type | Lua | `defaults write` |\n")
		b.WriteString("|-----|------|-----|------------------|\n")
		for _, k := range sec.Keys {
			luaPath := luaPathFor(sec.Name, k.Key)
			write := fmt.Sprintf("defaults write %s %s -%s <value>", sec.Domain, shellKey(k.Key), k.Kind)
			if k.Kind == "array" {
				write = fmt.Sprintf("defaults write %s %s <plist array>", sec.Domain, shellKey(k.Key))
			}
			desc := strings.ReplaceAll(k.Description, "|", "\\|")
			fmt.Fprintf(&b, "| `%s` | %s | `%s` | `%s` |\n", k.Key, k.Kind, luaPath, write)
			if desc != "" {
				fmt.Fprintf(&b, "| | | %s | |\n", desc)
			}
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "_Indexed %d keys in %d sections from nix-darwin `system.defaults`._\n", n, len(Catalog()))
	return b.String(), nil
}

func luaPathFor(section, key string) string {
	base := "macos.defaults"
	if isLuaIdent(section) {
		base += "." + section
	} else {
		base += fmt.Sprintf("[%q]", section)
	}
	if isLuaIdent(key) {
		return base + "." + key
	}
	return fmt.Sprintf("%s[%q]", base, key)
}

func isLuaIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		ok := r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || (i > 0 && r >= '0' && r <= '9')
		if !ok {
			return false
		}
	}
	return true
}

func shellKey(key string) string {
	if strings.ContainsAny(key, " ") {
		return `"` + key + `"`
	}
	return key
}

// CatalogKeyCount is the number of leaf keys in the embedded catalog.
func CatalogKeyCount() int {
	n := 0
	for _, s := range Catalog() {
		n += len(s.Keys)
	}
	return n
}
