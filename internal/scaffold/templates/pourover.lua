-- Root PourOver config. Edit packages.lua and declare file links below.
local packages = require("packages")

-- Homebrew background updates (brew autoupdate). Omit to leave unmanaged.
-- packages.auto_update = true
-- packages.auto_update = { enable = true, interval = "24h", upgrade = true, cleanup = true }

return {
  packages = packages,
  files = {
    links = {
      -- Example managed target (uncomment and adjust):
      -- { source = "config/nvim", target = "~/.config/nvim" },
    },
  },
  -- macOS defaults: copy keys from docs/macos-defaults.md (not listed here).
  -- macos = { defaults = { dock = { autohide = true } } },
  policy = {
    uninstall_mode = "safe",
  },
  backup = {
    icloud = {
      enabled = false,
    },
    git = {
      enabled = false,
      auto_push = true,
      branch = "main",
      -- remote = "git@github.com:USER/pourover-config.git",
    },
  },
}
