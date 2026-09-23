# Changelog

## v0.1.3
### 09/23/2026
- Show version number in app header (injected at build time via ldflags)

## v0.1.2
### 09/23/2026
- Fix snoozing acknowledged incidents — PagerDuty only allows snoozing triggered incidents, so the app now re-triggers them first before snoozing

## v0.1.1
### 09/15/2026
- Add "ack all" keybinding (Shift-A) to acknowledge all triggered incidents in the current view

## v0.1.0
### 09/10/2026
- Initial release
