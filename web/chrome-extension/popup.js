const iframe = document.getElementById("jameclaw-frame")

function postContextToFrame(context) {
  iframe.contentWindow?.postMessage(
    {
      type: "jameclaw-extension-context",
      context,
    },
    "http://localhost:18800",
  )
}

function requestContext() {
  chrome.runtime.sendMessage(
    { type: "jameclaw-extension-request-context" },
    (response) => {
      if (!response?.ok) {
        return
      }

      postContextToFrame(response.context)
    },
  )
}

iframe.addEventListener("load", () => {
  requestContext()
})

window.addEventListener("message", (event) => {
  if (event.origin !== "http://localhost:18800") {
    return
  }

  if (event.data?.type === "jameclaw-extension-refresh") {
    requestContext()
  }
})
