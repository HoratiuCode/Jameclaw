const PAGE_TEXT_LIMIT = 5000

function getSelectionText() {
  const selection = window.getSelection()
  return selection ? selection.toString().trim() : ""
}

function getPageText() {
  const root = document.body
  if (!root) {
    return ""
  }

  const text = root.innerText.replace(/\s+/g, " ").trim()
  if (text.length <= PAGE_TEXT_LIMIT) {
    return text
  }

  return `${text.slice(0, PAGE_TEXT_LIMIT)}...`
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type !== "jameclaw-extension-get-context") {
    return false
  }

  sendResponse({
    title: document.title || "",
    url: window.location.href || "",
    selection: getSelectionText(),
    pageText: getPageText(),
  })

  return false
})
