document.addEventListener("DOMContentLoaded", () => {
  const uploadForm = document.getElementById("upload-form");
  const selectType = document.getElementById("select-type");
  const uploadType = document.getElementById("upload-type");
  const passwordInput = document.getElementById("password-input");
  const fileInput = document.getElementById("file-input");
  const uploadBtn = document.getElementById("upload-btn");
  const logContainer = document.getElementById("log-container");
  const logOutput = document.getElementById("log-output");
  const clearLogBtn = document.getElementById("clear-log-btn");

  // State variables
  let selectedFile = null;

  // Helper function to update the button's disabled state
  const updateButtonState = () => {
    const selectedValue = selectType.value;
    const uploadTypeValue = uploadType.value;
    const isPasswordRequired = uploadTypeValue === "zip";
    const isPasswordSet = passwordInput.value.length > 0;
    const hasFile = selectedFile !== null;

    if (
      hasFile &&
      selectedValue &&
      (!isPasswordRequired || (isPasswordRequired && isPasswordSet))
    ) {
      uploadBtn.removeAttribute("aria-busy");
      uploadBtn.removeAttribute("disabled");
    } else {
      uploadBtn.setAttribute("disabled", "true");
    }
  };

  // Helper function for syntax highlighting
  const highlightJson = (jsonString) => {
    try {
      const json = JSON.parse(jsonString);
      const jsonFormatted = JSON.stringify(json, undefined, 2);
      return jsonFormatted.replace(
        /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
        (match) => {
          let cls = "syntax-number";
          if (/^"/.test(match)) {
            if (/:$/.test(match)) {
              cls = "syntax-key";
            } else {
              cls = "syntax-string";
            }
          } else if (/true|false/.test(match)) {
            cls = "syntax-boolean";
          } else if (/null/.test(match)) {
            cls = "syntax-null";
          }
          return `<span class="${cls}">${match}</span>`;
        }
      );
    } catch (e) {
      return jsonString; // Not a valid JSON, return as is
    }
  };

  // Event listeners
  selectType.addEventListener("change", updateButtonState);

  uploadType.addEventListener("change", () => {
    if (uploadType.value === "zip") {
      passwordInput.style.display = "block";
      fileInput.accept = ".zip";
    } else {
      passwordInput.style.display = "none";
      fileInput.accept = ".csv";
    }
    updateButtonState();
  });

  passwordInput.addEventListener("input", updateButtonState);

  fileInput.addEventListener("change", (event) => {
    selectedFile = event.target.files.length > 0 ? event.target.files[0] : null;
    updateButtonState();
  });

  uploadForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    if (
      !selectedFile ||
      !selectType.value ||
      (uploadType.value === "zip" && !passwordInput.value)
    ) {
      return;
    }

    // Set loading state
    uploadBtn.setAttribute("aria-busy", "true");
    uploadBtn.textContent = "上传中...";
    logContainer.style.display = "none";
    logOutput.innerHTML = "";

    const formData = new FormData();
    formData.append("file", selectedFile);
    if (uploadType.value === "zip") {
      formData.append("password", passwordInput.value);
    }

    let url = "";
    if (selectType.value === "alipay") {
      url =
        uploadType.value === "zip"
          ? "http://127.0.0.1:10777/upload/alipay_zip"
          : "http://127.0.0.1:10777/upload/alipay_csv";
    } else if (selectType.value === "wechat") {
      url = "http://127.0.0.1:10777/upload/wechat_csv";
    }

    try {
      const response = await axios.post(url, formData, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      logOutput.innerHTML = highlightJson(
        JSON.stringify(response.data, null, 2)
      );
    } catch (error) {
      const errorMessage = error?.response?.data?.error || error.message;
      logOutput.textContent = `请求失败：${errorMessage}`;
    } finally {
      // Reset button state and show log
      uploadBtn.removeAttribute("aria-busy");
      uploadBtn.textContent = "上传";
      logContainer.style.display = "block";
    }
  });

  clearLogBtn.addEventListener("click", (event) => {
    event.preventDefault();
    logOutput.innerHTML = "";
    logContainer.style.display = "none";
  });

  // Initial state setup
  updateButtonState();
});
