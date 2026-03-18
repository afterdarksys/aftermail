package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// buildRSSTab creates the RSS Reader UI
func buildRSSTab(w fyne.Window) fyne.CanvasObject {
	// Sidebar: Feed List
	feeds := []string{
		"Hacker News",
		"Ars Technica",
		"Go Blog",
		"AfterDark Security Notes",
	}

	feedList := widget.NewList(
		func() int { return len(feeds) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Feed Name")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(feeds[i])
		})

	addFeedBtn := widget.NewButton("Add Feed", func() {
		entry := widget.NewEntry()
		entry.SetPlaceHolder("https://example.com/rss.xml")
		dialog.ShowCustomConfirm("Add RSS Bookmark", "Add", "Cancel", entry, func(b bool) {
			if b && entry.Text != "" {
				feeds = append(feeds, "New Feed")
				feedList.Refresh()
				dialog.ShowInformation("RSS Bookmark Added", fmt.Sprintf("Added %s", entry.Text), w)
			}
		}, w)
	})

	feedSidebar := container.NewBorder(addFeedBtn, nil, nil, nil, feedList)

	// Main Content: Articles List and Viewer
	articles := []string{
		"1. Multi-Engine Architecture and Implementation",
		"2. Go 1.25 Release Notes",
		"3. The Future of Zero-Knowledge Proofs",
		"4. RSS Readers making a comeback",
	}

	articleViewer := widget.NewRichTextFromMarkdown("# Welcome to RSS Reader\nSelect a feed and article to read.")
	articleScroll := container.NewVScroll(articleViewer)

	articleList := widget.NewList(
		func() int { return len(articles) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Article Title")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(articles[i])
		})

	articleList.OnSelected = func(id widget.ListItemID) {
		articleViewer.ParseMarkdown(fmt.Sprintf("# %s\n\nThis is the content for the selected RSS article. The feed logic will fetch and render HTML/Markdown here.", articles[id]))
	}

	mainArea := container.NewHSplit(articleList, articleScroll)
	mainArea.SetOffset(0.3)

	rssLayout := container.NewHSplit(feedSidebar, mainArea)
	rssLayout.SetOffset(0.2)

	header := container.NewHBox(
		widget.NewLabelWithStyle("RSS Reader", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Refresh All", func() {
			dialog.ShowInformation("RSS Sync", "Syncing all feeds...", w)
		}),
	)

	return container.NewBorder(header, nil, nil, nil, rssLayout)
}
