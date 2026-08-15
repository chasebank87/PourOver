-- Homebrew and Mac App Store packages declared for PourOver.
-- Use lowercase Homebrew tokens (e.g. "raycast", not "Raycast").
-- MAS apps use display name → App Store ID; omit `mas` to leave App Store unmanaged.
return {
  taps = {
    -- "homebrew/cask-fonts",
  },
  formulae = {
    "git",
  },
  casks = {
    -- "raycast",
    -- "warp",
  },
  -- mas = {
  --   Xcode = 497799835,
  -- },
}
