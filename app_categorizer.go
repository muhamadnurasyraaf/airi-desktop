package main

import "strings"

// categorizeApplication categorizes applications into productivity categories
func categorizeApplication(appName string) string {
	appLower := strings.ToLower(appName)

	// Development tools
	devTools := []string{"code", "vscode", "visual studio", "intellij", "pycharm", "webstorm",
		"rider", "goland", "eclipse", "netbeans", "atom", "sublime", "vim", "nvim", "emacs",
		"git", "github", "gitlab", "terminal", "iterm", "powershell", "cmd", "wsl", "docker",
		"postman", "insomnia", "datagrip", "dbeaver", "mysql", "postgres", "mongodb"}
	for _, tool := range devTools {
		if strings.Contains(appLower, tool) {
			return "development"
		}
	}

	// Productivity tools
	productivity := []string{"excel", "word", "powerpoint", "outlook", "onenote", "notion",
		"evernote", "obsidian", "roam", "todoist", "trello", "asana", "jira", "confluence",
		"slack", "teams", "zoom", "meet", "calendar", "mail", "notes", "reminders"}
	for _, tool := range productivity {
		if strings.Contains(appLower, tool) {
			return "productivity"
		}
	}

	// Browsers
	browsers := []string{"chrome", "firefox", "safari", "edge", "brave", "opera", "vivaldi", "arc"}
	for _, browser := range browsers {
		if strings.Contains(appLower, browser) {
			return "browser"
		}
	}

	// Communication
	communication := []string{"discord", "telegram", "whatsapp", "signal", "messenger",
		"skype", "wechat", "line", "viber"}
	for _, comm := range communication {
		if strings.Contains(appLower, comm) {
			return "communication"
		}
	}

	// Entertainment
	entertainment := []string{"spotify", "youtube", "netflix", "twitch", "steam", "epic",
		"vlc", "media player", "itunes", "music", "video", "game"}
	for _, ent := range entertainment {
		if strings.Contains(appLower, ent) {
			return "entertainment"
		}
	}

	// Design tools
	design := []string{"photoshop", "illustrator", "figma", "sketch", "adobe", "gimp",
		"inkscape", "blender", "canva", "affinity"}
	for _, tool := range design {
		if strings.Contains(appLower, tool) {
			return "design"
		}
	}

	// File management
	fileTools := []string{"explorer", "finder", "nautilus", "dolphin", "ranger", "7zip",
		"winrar", "dropbox", "drive", "onedrive", "box"}
	for _, tool := range fileTools {
		if strings.Contains(appLower, tool) {
			return "file_management"
		}
	}

	return "unknown"
}
