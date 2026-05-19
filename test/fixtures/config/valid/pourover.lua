return {
  packages = {
    formulae = { "git", "fzf" },
    casks = { "raycast" },
  },
  files = {
    links = {
      { source = "config/nvim", target = "~/.config/nvim" },
    },
  },
  policy = {
    uninstall_mode = "safe",
  },
  backup = {
    icloud = {
      enabled = true,
    },
  },
}
