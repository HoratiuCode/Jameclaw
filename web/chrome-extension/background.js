const DOCK_STORAGE_KEY = "jameclaw-extension-dock-enabled"

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
        sendResponse({ ok: true, enabled })
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
              ok: true,
              enabled,
              warning: chrome.runtime.lastError.message,
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
              error: chrome.runtime.lastError.message,
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

  return false
})
