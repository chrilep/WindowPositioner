# WindowPositioner

**Never move your windows manually again!**  
WindowPositioner lets you save and restore window positions automatically, optimizing your workflow.

## Overview

WindowPositioner helps you save window layouts according to your preferences and restore them whenever needed. Perfect for multi-monitor setups or frequently used applications you want always positioned just right.

![WindowPositioner App Screenshot](https://raw.githubusercontent.com/chrilep/WindowPositioner/refs/heads/main/res/gui.png)

- Save positions of arbitrary windows.  
- Positions are stored in a JSON file inside your roaming AppData folder, ideal for user profiles.  
- Automatic logging of actions and errors for easy debugging.  
- Optional Windows startup to have the app ready immediately after login.

## Data Storage Location

Positions are saved under:  
`%APPDATA%\Lancer\WindowPositioner\positions.json`  
Example:  
`C:\Users\User\AppData\Roaming\Lancer\WindowPositioner\positions.json`

The log file is located at:  
`%LOCALAPPDATA%\Lancer\WindowPositioner\log.txt`  
Example:  
`C:\Users\User\AppData\Local\Lancer\WindowPositioner\log.txt`

## Features & Ideas

Already implemented:

- Save, load, and delete window positions.  
- Automatic saving of window positions as desired by the user.  
- Run at Windows startup (autostart).  
- Clear lists showing all visible windows and saved positions.

Planned enhancements (time and motivation permitting):

- [ ] Capture and display window screenshots for better orientation.  
- [ ] Support for wildcards or regular expressions in window titles (e.g., grouping windows).

## Installation & Usage

1. Launch the application.  
2. Select a visible window from the automatically updated list.  
3. Save its position via the provided button.  
4. The position is stored in the JSON file and can be restored at any time.  
5. Optionally enable autostart for automatic launching with Windows.

## Changelog

**1.0.0 — 31.07.2025**  
- Initial release with core functionality.

## Support & Contributing

Got feedback or want to contribute?  
Feel free to fork the project on GitHub and submit pull requests with improvements!

If you want, I can also provide a quick guide on compiling or using the app. Just ask!
