const idInstanceInput = document.getElementById("idInstance");
const apiTokenInput = document.getElementById("apiTokenInstance");
const messageChatIdInput = document.getElementById("messageChatId");
const messageTextInput = document.getElementById("messageText");
const fileChatIdInput = document.getElementById("fileChatId");
const fileUrlInput = document.getElementById("fileUrl");
const responseOutput = document.getElementById("responseOutput");

const getSettingsBtn = document.getElementById("getSettingsBtn");
const getStateBtn = document.getElementById("getStateBtn");
const sendMessageBtn = document.getElementById("sendMessageBtn");
const sendFileBtn = document.getElementById("sendFileBtn");

function setResponse(data) {
  if (typeof data === "string") {
    responseOutput.value = data;
    return;
  }

  responseOutput.value = JSON.stringify(data, null, 2);
}

function getCredentials() {
  const idInstance = idInstanceInput.value.trim();
  const apiTokenInstance = apiTokenInput.value.trim();

  if (!idInstance || !apiTokenInstance) {
    throw new Error("idInstance and ApiTokenInstance are required.");
  }

  return { idInstance, apiTokenInstance };
}

async function callBackend(endpoint, payload) {
  try {
    setResponse("Loading...");

    const response = await fetch(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
      setResponse({
        status: response.status,
        statusText: response.statusText,
        error: data,
      });
      return;
    }

    setResponse(data);
  } catch (error) {
    setResponse({ error: error.message });
  }
}

function extractFileName(url) {
  try {
    const pathname = new URL(url).pathname;
    const lastPart = pathname.split("/").filter(Boolean).pop();
    return lastPart || "file";
  } catch {
    return "file";
  }
}

function withCredentials(callback) {
  try {
    const credentials = getCredentials();
    callback(credentials);
  } catch (error) {
    setResponse({ error: error.message });
  }
}

getSettingsBtn.addEventListener("click", () => {
  withCredentials((credentials) => {
    callBackend("/api/getSettings", credentials);
  });
});

getStateBtn.addEventListener("click", () => {
  withCredentials((credentials) => {
    callBackend("/api/getStateInstance", credentials);
  });
});

sendMessageBtn.addEventListener("click", () => {
  const chatId = messageChatIdInput.value.trim();
  const message = messageTextInput.value.trim();

  if (!chatId || !message) {
    setResponse({ error: "chatId and message are required for sendMessage." });
    return;
  }

  withCredentials((credentials) => {
    callBackend("/api/sendMessage", { ...credentials, chatId, message });
  });
});

sendFileBtn.addEventListener("click", () => {
  const chatId = fileChatIdInput.value.trim();
  const urlFile = fileUrlInput.value.trim();

  if (!chatId || !urlFile) {
    setResponse({ error: "chatId and urlFile are required for sendFileByUrl." });
    return;
  }

  const fileName = extractFileName(urlFile);
  withCredentials((credentials) => {
    callBackend("/api/sendFileByUrl", {
      ...credentials,
      chatId,
      urlFile,
      fileName,
    });
  });
});
