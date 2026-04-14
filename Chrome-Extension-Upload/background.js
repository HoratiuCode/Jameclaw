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
    chrome.windows.getCurrent({}, (currentWindow) => {
      if (chrome.runtime.lastError || !currentWindow?.id) {
        sendResponse({
          ok: false,
          error: chrome.runtime.lastError?.message || "Could not find current window.",
        })
        return
      }

      const width = 430
      const fallbackHeight = 760
      const currentLeft = typeof currentWindow.left === "number" ? currentWindow.left : 80
      const currentTop = typeof currentWindow.top === "number" ? currentWindow.top : 80
      const currentWidth = typeof currentWindow.width === "number" ? currentWindow.width : 1400
      const currentHeight =
        typeof currentWindow.height === "number" ? currentWindow.height : fallbackHeight

      const left = Math.max(currentLeft + currentWidth - width, 0)
      const top = Math.max(currentTop, 0)

      chrome.windows.create(
        {
          url: chrome.runtime.getURL("sidepanel.html"),
          type: "popup",
          width,
          height: currentHeight,
          left,
          top,
          focused: true,
        },
        () => {
          if (chrome.runtime.lastError) {
            sendResponse({
              ok: false,
              error: chrome.runtime.lastError.message || "Could not open docked chat.",
            })
            return
          }

          sendResponse({ ok: true, mode: "window" })
        },
      )
    })

    return true
  }

  return false
})
