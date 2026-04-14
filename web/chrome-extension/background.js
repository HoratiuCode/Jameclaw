chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
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

  if (message?.type === "jameclaw-extension-open-sidepanel") {
    chrome.tabs.query({ active: true, lastFocusedWindow: true }, async (tabs) => {
      const tab = tabs[0]

      if (chrome.runtime.lastError || !tab?.windowId) {
        sendResponse({ ok: false, error: "No active browser window." })
        return
      }

      try {
        if (chrome.sidePanel?.open) {
          await chrome.sidePanel.setOptions({
            path: "sidepanel.html",
            enabled: true,
            tabId: tab.id,
          })
          await chrome.sidePanel.open({ tabId: tab.id })
          sendResponse({ ok: true, mode: "sidepanel" })
          return
        }
      } catch (error) {
        chrome.tabs.create({ url: chrome.runtime.getURL("sidepanel.html") }, () => {
          if (chrome.runtime.lastError) {
            sendResponse({
              ok: false,
              error:
                error instanceof Error
                  ? error.message
                  : "Failed to open side panel.",
            })
            return
          }

          sendResponse({ ok: true, mode: "tab" })
        })
      }
    })

    return true
  }

  return false
})
