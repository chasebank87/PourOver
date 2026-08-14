-- Root PourOver config. Edit packages.lua and declare file links below.
local packages = require("packages")

return {
  packages = packages,
  files = {
    links = {
      -- Example managed target (uncomment and adjust):
      -- { source = "config/nvim", target = "~/.config/nvim" },
    },
  },
  policy = {
    uninstall_mode = "safe",
  },
  backup = {
    icloud = {
      enabled = false,
    },
  },
}
