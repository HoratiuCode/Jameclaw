---
title: "TOOLS.md Template"
summary: "Workspace template for TOOLS.md"
read_when:
  - Bootstrapping a workspace manually
---

# TOOLS.md - Local Notes

Skills define how tools work. This file is for your setup-specific notes.

## What Goes Here

Things like:

- Camera names and locations
- SSH hosts and aliases
- Preferred voices for TTS
- Speaker or room names
- Device nicknames
- Anything environment-specific

## Examples

```markdown
### Cameras

- living-room -> Main area, 180 degree wide angle
- front-door -> Entrance, motion-triggered

### SSH

- home-server -> 192.168.1.100, user: admin

### TTS

- Preferred voice: "Nova" (warm, slightly British)
- Default speaker: Kitchen HomePod
```

## Why Separate

Skills are shared. Your setup is local. Keeping them apart means you can update
skills without losing your notes, and share skills without exposing your own
infrastructure details.

---

Add whatever helps the agent use your real tools correctly. This is your local cheat sheet.
