chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message?.type !== "jameclaw-extension-request-context") {
    return false
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
})
