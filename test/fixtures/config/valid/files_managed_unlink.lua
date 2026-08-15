return {
  files = {
    managed = {
      { source = "config/foo.conf", target = "~/.config/foo.conf" },
    },
    unlink = { "~/.old-dotfile" },
  },
}
