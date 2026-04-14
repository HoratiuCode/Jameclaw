const PAGE_TEXT_LIMIT = 5000
const SELECTION_LIMIT = 3000
let lastSelectionText = ""

function getSelectionText() {
  const selection = window.getSelection()
  return selection ? selection.toString().trim() : ""
}

function trimText(text, limit) {
  if (text.length <= limit) {
    return text
  }

  return `${text.slice(0, limit)}...`
}

function rememberSelection() {
  const selection = trimText(getSelectionText(), SELECTION_LIMIT)
  if (selection) {
    lastSelectionText = selection
  }
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

document.addEventListener("mouseup", rememberSelection, true)
document.addEventListener("keyup", rememberSelection, true)
document.addEventListener("selectionchange", () => {
  const selection = trimText(getSelectionText(), SELECTION_LIMIT)
  if (selection) {
    lastSelectionText = selection
  }
})

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type !== "jameclaw-extension-get-context") {
    return false
  }

  const liveSelection = trimText(getSelectionText(), SELECTION_LIMIT)
  const selection = liveSelection || lastSelectionText

  sendResponse({
    title: document.title || "",
    url: window.location.href || "",
    selection,
    pageText: getPageText(),
  })

  return false
})
