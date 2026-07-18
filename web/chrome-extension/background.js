const DOCK_STORAGE_KEY = "jameclaw-extension-dock-enabled"

function clearDockOnActiveTab(callback) {
  chrome.storage.local.set({ [DOCK_STORAGE_KEY]: false }, () => {
    chrome.tabs.query({ active: true, lastFocusedWindow: true }, (tabs) => {
      const tab = tabs[0]
      if (!tab?.id) {
        callback()
        return
      }

      chrome.tabs.sendMessage(
        tab.id,
        {
          type: "jameclaw-extension-dock-state",
          enabled: false,
        },
        () => callback(),
      )
    })
  })
}

function openPanelWindow(sendResponse) {
  clearDockOnActiveTab(() => {
    chrome.windows.create(
      {
        url: chrome.runtime.getURL("sidepanel.html?mode=window"),
        type: "popup",
        width: 420,
        height: 660,
        focused: true,
      },
      (windowInfo) => {
        if (chrome.runtime.lastError) {
          sendResponse({
            ok: false,
            error:
              chrome.runtime.lastError.message ||
              "Could not open JameClaw window.",
          })
          return
        }

        sendResponse({ ok: true, windowId: windowInfo?.id })
      },
    )
  })
}

function setDockEnabled(enabled, sendResponse) {
  chrome.storage.local.set({ [DOCK_STORAGE_KEY]: enabled }, () => {
    if (chrome.runtime.lastError) {
      sendResponse({
        ok: false,
        error: chrome.runtime.lastError.message || "Could not save dock state.",
      })
      return
    }

    chrome.tabs.query({ active: true, lastFocusedWindow: true }, (tabs) => {
      const tab = tabs[0]
      if (!tab?.id) {
        sendResponse({
          ok: false,
          error: "No active tab is available.",
        })
        return
      }

      chrome.tabs.sendMessage(
        tab.id,
        {
          type: "jameclaw-extension-dock-state",
          enabled,
        },
        () => {
          if (chrome.runtime.lastError) {
            sendResponse({
              ok: false,
              error:
                chrome.runtime.lastError.message ||
                "Could not update the dock on the active tab.",
            })
            return
          }

          sendResponse({ ok: true, enabled })
        },
      )
    })
  })
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
	if (message?.type === "jameclaw-extension-browser-client") {
		const tabID = _sender.tab?.id
		if (!Number.isInteger(tabID)) {
			sendResponse({ ok: false, error: "No tab is available." })
			return false
		}
		sendResponse({ ok: true, clientId: `chrome-tab-${tabID}` })
		return false
	}

  if (message?.type === "jameclaw-extension-request-context") {
    chrome.tabs.query({ active: true, lastFocusedWindow: true }, (tabs) => {
      const tab = tabs[0]

      if (!tab?.id) {
        sendResponse({
          ok: false,
          error: "No active tab is available.",
        })
        return
      }

      chrome.tabs.sendMessage(
        tab.id,
        { type: "jameclaw-extension-get-context" },
        (response) => {
          if (chrome.runtime.lastError) {
            sendResponse({
              ok: false,
              error:
                chrome.runtime.lastError.message ||
                "Could not read the active page context.",
            })
            return
          }

          sendResponse({
            ok: true,
            context: {
              title: tab.title || response?.title || "",
              url: tab.url || response?.url || "",
              selection: response?.selection || "",
              pageText: response?.pageText || "",
            },
          })
        },
      )
    })

    return true
  }

  if (message?.type === "jameclaw-extension-set-dock-state") {
    setDockEnabled(Boolean(message.enabled), sendResponse)
    return true
  }

  if (message?.type === "jameclaw-extension-open-panel-window") {
    openPanelWindow(sendResponse)
    return true
  }

  return false
})
